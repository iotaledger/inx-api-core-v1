package database

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/hive.go/runtime/contextutils"
	"github.com/iotaledger/hive.go/serializer"
	"github.com/iotaledger/hive.go/serializer/v2/byteutils"
	"github.com/iotaledger/inx-api-core-v1/pkg/hornet"
	"github.com/iotaledger/inx-api-core-v1/pkg/milestone"
	iotago "github.com/iotaledger/iota.go/v2"
)

const (
	// printStatusInterval is the interval for printing status messages.
	printStatusInterval = 2 * time.Second
)

var (
	// ErrOperationAborted is returned when the operation was aborted e.g. by a shutdown signal.
	ErrOperationAborted = errors.New("operation was aborted")
)

func (db *Database) checkConflictingTransactionsMessageIDsStatus() (bool, error) {
	value, err := db.conflictingTransactionsStore.Get([]byte("status"))
	if err != nil {
		if errors.Is(err, kvstore.ErrKeyNotFound) {
			return false, nil
		}

		return false, fmt.Errorf("reading conflicting transactions store status failed: %w", err)
	}

	conflictingTransactionsStoreIndex := milestone.Index(binary.LittleEndian.Uint32(value))
	ledgerIndex := db.utxoManager.ReadLedgerIndex()

	return conflictingTransactionsStoreIndex == ledgerIndex, nil
}

func (db *Database) createConflictingTransactionsMessageIDsLookupTable(ctx context.Context) error {
	// first we need to delete the old table before we rebuild the lookup table
	if err := db.conflictingTransactionsStore.DeletePrefix([]byte{}); err != nil {
		return fmt.Errorf("deleting conflicting transactions store failed: %w", err)
	}

	// set the entry in the lookup table to find conflicting transactions per address
	setConflictingTransactionsStoreEntry := func(address iotago.Address, messageID hornet.MessageID) error {
		addrBytes, err := address.Serialize(serializer.DeSeriModeNoValidation)
		if err != nil {
			return fmt.Errorf("failed to serialize address, msgID: %s, address: %s, error: %w", messageID.ToHex(), address.String(), err)
		}

		if err := db.conflictingTransactionsStore.Set(byteutils.ConcatBytes(addrBytes, messageID), []byte{}); err != nil {
			return fmt.Errorf("setting entry in conflicting transactions store failed, msgID: %s, error: %w", messageID.ToHex(), err)
		}

		return nil
	}

	// we loop over all existing messages and filter messages that contain conflicting transactions to create the lookup table
	var innerErr error

	lastStatusTime := time.Now()
	var metadataCounter int64
	if err := db.metadataStore.Iterate(kvstore.EmptyPrefix, func(key []byte, data []byte) bool {
		metadataCounter++

		if time.Since(lastStatusTime) >= printStatusInterval {
			lastStatusTime = time.Now()

			if err := contextutils.ReturnErrIfCtxDone(ctx, ErrOperationAborted); err != nil {
				return false
			}

			db.LogInfof("analyzed %d messages", metadataCounter)
		}

		messageID := hornet.MessageIDFromSlice(key[:iotago.MessageIDLength])

		msgMeta, err := metadataFactory(messageID, data)
		if err != nil {
			innerErr = fmt.Errorf("failed to deserialize message metadata: %s, error: %w", messageID.ToHex(), err)
			return false
		}

		// check if the message is a conflicting transaction
		if !msgMeta.IsConflictingTx() {
			return true
		}

		// get the conflicting message
		msg := db.MessageOrNil(messageID)
		if msg == nil {
			innerErr = fmt.Errorf("message not found: %s", messageID.ToHex())
			return false
		}

		txEssence := msg.TransactionEssence()
		if txEssence == nil {
			innerErr = fmt.Errorf("transaction does not contain a valid transactionEssence: msgID: %s", messageID.ToHex())
			return false
		}

		for _, input := range txEssence.Inputs {
			utxoInput, ok := input.(*iotago.UTXOInput)
			if !ok {
				innerErr = fmt.Errorf("transaction contains an unsupported input type: msgID: %s", messageID.ToHex())
				return false
			}

			utxoInputID := utxoInput.ID()
			output, err := db.utxoManager.ReadOutputByOutputID(&utxoInputID)
			if err != nil {
				// if we don't have the input, we don't have the history, which is fine.
				continue
			}

			if err := setConflictingTransactionsStoreEntry(output.Address(), messageID); err != nil {
				innerErr = err
				return false
			}
		}

		for _, txOutput := range txEssence.Outputs {
			switch output := txOutput.(type) {
			case *iotago.SigLockedSingleOutput:
				//nolint:forcetypeassert
				if err := setConflictingTransactionsStoreEntry(output.Address.(iotago.Address), messageID); err != nil {
					innerErr = err
					return false
				}
			case *iotago.SigLockedDustAllowanceOutput:
				//nolint:forcetypeassert
				if err := setConflictingTransactionsStoreEntry(output.Address.(iotago.Address), messageID); err != nil {
					innerErr = err
					return false
				}
			default:
				innerErr = fmt.Errorf("transaction contains an unsupported output type: msgID: %s", messageID.ToHex())
				return false
			}
		}

		return true
	}, kvstore.IterDirectionForward); err != nil {
		return fmt.Errorf("iterating over all existing messages failed: %w", err)
	}
	if innerErr != nil {
		return innerErr
	}

	// set store status
	ledgerIndex := db.utxoManager.ReadLedgerIndex()

	value := make([]byte, 4)
	binary.LittleEndian.PutUint32(value, uint32(ledgerIndex))

	if err := db.conflictingTransactionsStore.Set([]byte("status"), value); err != nil {
		return fmt.Errorf("setting conflicting transactions store status failed: %w", err)
	}

	// flush the table
	if err := db.conflictingTransactionsStore.Flush(); err != nil {
		return fmt.Errorf("flushing conflicting transactions store failed: %w", err)
	}

	return nil
}

// ConflictingTransactionsMessageIDs returns the message IDs of conflicting transactions of the given address.
func (db *Database) ConflictingTransactionsMessageIDs(address iotago.Address, maxResults ...int) (hornet.MessageIDs, error) {
	var conflictingTransactionsMessageIDs hornet.MessageIDs

	addrBytes, err := address.Serialize(serializer.DeSeriModeNoValidation)
	if err != nil {
		return nil, err
	}

	iterations := 0
	if err := db.conflictingTransactionsStore.IterateKeys(addrBytes, func(key []byte) bool {
		iterations++

		conflictingTransactionsMessageIDs = append(conflictingTransactionsMessageIDs, hornet.MessageIDFromSlice(key[len(addrBytes):len(addrBytes)+iotago.MessageIDLength]))

		if len(maxResults) > 0 {
			// stop if maximum amount of iterations reached
			return iterations <= maxResults[0]
		}

		return true
	}); err != nil {
		return nil, err
	}

	return conflictingTransactionsMessageIDs, nil
}
