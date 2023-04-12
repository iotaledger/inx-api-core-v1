package utxo

import (
	"encoding/binary"

	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/hive.go/serializer"
	"github.com/iotaledger/hive.go/serializer/v2/marshalutil"
	"github.com/iotaledger/inx-api-core-v1/pkg/milestone"
	iotago "github.com/iotaledger/iota.go/v2"
)

// ReceiptTuple contains a receipt and the index of the milestone
// which contained the receipt.
type ReceiptTuple struct {
	// The actual receipt.
	Receipt *iotago.Receipt `json:"receipt"`
	// The index of the milestone which included the receipt.
	MilestoneIndex milestone.Index `json:"milestoneIndex"`
}

func (rt *ReceiptTuple) kvStorableLoad(_ *Manager, key []byte, value []byte) error {
	keyExt := marshalutil.New(key)

	// skip prefix and migrated at index
	if _, err := keyExt.ReadByte(); err != nil {
		return err
	}

	if _, err := keyExt.ReadUint32(); err != nil {
		return err
	}

	// read out index of the milestone which contained this receipt
	msIndex, err := keyExt.ReadUint32()
	if err != nil {
		return err
	}

	r := &iotago.Receipt{}
	if _, err := r.Deserialize(value, serializer.DeSeriModeNoValidation); err != nil {
		return err
	}

	rt.Receipt = r
	rt.MilestoneIndex = milestone.Index(msIndex)

	return nil
}

// ReceiptTupleConsumer is a function that consumes a receipt tuple.
type ReceiptTupleConsumer func(rt *ReceiptTuple) bool

// ForEachReceiptTuple iterates over all stored receipt tuples.
func (u *Manager) ForEachReceiptTuple(consumer ReceiptTupleConsumer, options ...IterateOption) error {
	opt := iterateOptions(options)

	var innerErr error
	var i int
	if err := u.utxoStorage.Iterate([]byte{UTXOStoreKeyPrefixReceipts}, func(key kvstore.Key, value kvstore.Value) bool {

		if (opt.maxResultCount > 0) && (i >= opt.maxResultCount) {
			return false
		}

		i++

		rt := &ReceiptTuple{}
		if err := rt.kvStorableLoad(u, key, value); err != nil {
			innerErr = err

			return false
		}

		return consumer(rt)
	}); err != nil {
		return err
	}

	return innerErr
}

// ForEachReceiptTupleMigratedAt iterates over all stored receipt tuples for a given migrated at index.
func (u *Manager) ForEachReceiptTupleMigratedAt(migratedAtIndex milestone.Index, consumer ReceiptTupleConsumer, options ...IterateOption) error {
	opt := iterateOptions(options)

	prefix := make([]byte, 5)
	prefix[0] = UTXOStoreKeyPrefixReceipts
	binary.LittleEndian.PutUint32(prefix[1:], uint32(migratedAtIndex))

	var innerErr error
	var i int
	if err := u.utxoStorage.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {

		if (opt.maxResultCount > 0) && (i >= opt.maxResultCount) {
			return false
		}

		i++

		rt := &ReceiptTuple{}
		if err := rt.kvStorableLoad(u, key, value); err != nil {
			innerErr = err

			return false
		}

		return consumer(rt)
	}); err != nil {
		return err
	}

	return innerErr
}
