package server

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/inx-api-core-v1/pkg/database"
	"github.com/iotaledger/inx-api-core-v1/pkg/hornet"
	"github.com/iotaledger/inx-api-core-v1/pkg/milestone"
	"github.com/iotaledger/inx-api-core-v1/pkg/restapi"
	"github.com/iotaledger/inx-api-core-v1/pkg/utxo"
	"github.com/iotaledger/inx-app/pkg/httpserver"
	iotago "github.com/iotaledger/iota.go/v2"
)

func (s *DatabaseServer) messageIDByTransactionID(c echo.Context) (hornet.MessageID, error) {
	transactionID, err := restapi.ParseTransactionIDParam(c)
	if err != nil {
		return nil, err
	}

	// Get the first output of that transaction (using index 0)
	outputID := &iotago.UTXOInputID{}
	copy(outputID[:], transactionID[:])

	output, err := s.UTXOManager.ReadOutputByOutputID(outputID)
	if err != nil {
		if errors.Is(err, kvstore.ErrKeyNotFound) {
			return nil, errors.WithMessagef(echo.ErrNotFound, "output for transaction not found: %s", hex.EncodeToString(transactionID[:]))
		}

		return nil, errors.WithMessagef(echo.ErrInternalServerError, "failed to load output for transaction: %s", hex.EncodeToString(transactionID[:]))
	}

	return output.MessageID(), nil
}

func (s *DatabaseServer) transactionHistoryByAddress(c echo.Context, address iotago.Address) (*transactionHistoryResponse, error) {
	ledgerIndex := s.UTXOManager.ReadLedgerIndex()

	maxResults := s.maxResultsFromContext(c)

	messageIDs := make(map[string]struct{}, 0)
	if err := s.UTXOManager.ForEachUnspentOutput(func(output *utxo.Output) bool {
		// add the message that contains the transaction which created this output
		messageIDs[output.MessageID().ToMapKey()] = struct{}{}

		// we always collect all results and cap to the maximum later to have deterministic responses
		return true
	}, utxo.FilterAddress(address)); err != nil {
		return nil, errors.WithMessagef(echo.ErrInternalServerError, "reading unspent outputs failed: %s, error: %s", address, err)
	}

	// helper function to get the message ID of the transaction that spent the output
	getSpendingMessageID := func(transactionID *iotago.TransactionID) (hornet.MessageID, error) {
		// get the first output of that transaction (using index 0)
		outputID := &iotago.UTXOInputID{}
		copy(outputID[:], transactionID[:])

		output, err := s.UTXOManager.ReadOutputByOutputID(outputID)
		if err != nil {
			if errors.Is(err, kvstore.ErrKeyNotFound) {
				// if we don't have the output, we don't have the history, which is fine.
				//nolint:nilnil
				return nil, nil
			}

			return nil, errors.WithMessagef(echo.ErrInternalServerError, "failed to load output for transaction: %s", hex.EncodeToString(transactionID[:]))
		}

		return output.MessageID(), nil
	}

	var innerErr error
	if err := s.UTXOManager.ForEachSpentOutput(func(spent *utxo.Spent) bool {
		// add the message that contains the transaction which created this output
		messageIDs[spent.MessageID().ToMapKey()] = struct{}{}

		// also add the message that contains the transaction that spent this output
		spendingMessageID, err := getSpendingMessageID(spent.TargetTransactionID())
		if err != nil {
			innerErr = errors.WithMessagef(echo.ErrInternalServerError, "reading spent outputs failed: %s, error: %s", address, err)
			return false
		}
		messageIDs[spendingMessageID.ToMapKey()] = struct{}{}

		// we always collect all results and cap to the maximum later to have deterministic responses
		return true
	}, utxo.FilterAddress(address)); err != nil {
		return nil, errors.WithMessagef(echo.ErrInternalServerError, "reading spent outputs failed: %s, error: %s", address, err)
	}
	if innerErr != nil {
		return nil, innerErr
	}

	// add the messages that contain conflicting transactions for the given address
	conflictingTransactionsMessageIDs, err := s.Database.ConflictingTransactionsMessageIDs(address)
	if err != nil {
		return nil, errors.WithMessagef(echo.ErrInternalServerError, "reading conflicting transaction messageIDs failed: %s, error: %s", address, err)
	}

	for _, conflictingTransactionsMessageID := range conflictingTransactionsMessageIDs {
		messageIDs[conflictingTransactionsMessageID.ToMapKey()] = struct{}{}
	}

	getTransactionHistoryItem := func(messageID hornet.MessageID) (*transactionHistoryItem, error) {
		msg := s.Database.MessageOrNil(messageID)
		if msg == nil {
			// if we don't have the message, we don't have the history, which is fine.
			//nolint:nilnil
			return nil, nil
		}

		msgMeta := s.Database.MessageMetadataOrNil(messageID)
		if msgMeta == nil {
			return nil, fmt.Errorf("message not found: %s", messageID.ToHex())
		}

		var referencedByMilestoneIndex milestone.Index
		if referenced, referencedIndex := msgMeta.ReferencedWithIndex(); referenced {
			referencedByMilestoneIndex = referencedIndex
		}

		ledgerInclusionState := "noTransaction"
		conflict := msgMeta.Conflict()
		var conflictReason *database.Conflict

		if conflict != database.ConflictNone {
			ledgerInclusionState = "conflicting"
			conflictReason = &conflict
		} else if msgMeta.IsIncludedTxInLedger() {
			ledgerInclusionState = "included"
		}

		txPayload := msg.Transaction()
		if txPayload == nil {
			return nil, fmt.Errorf("message does not contain a transaction payload: %s", messageID.ToHex())
		}

		transactionID, err := txPayload.ID()
		if err != nil {
			return nil, fmt.Errorf("can't compute the transaction ID, msgID: %s, error: %w", messageID.ToHex(), err)
		}
		txID := *transactionID

		txEssence := msg.TransactionEssence()
		if txEssence == nil {
			return nil, fmt.Errorf("transaction does not contain a valid transactionEssence: msgID: %s", messageID.ToHex())
		}

		var addressBalanceInputs int64
		for _, input := range txEssence.Inputs {
			utxoInput, ok := input.(*iotago.UTXOInput)
			if !ok {
				return nil, fmt.Errorf("transaction contains an unsupported input type: msgID: %s", messageID.ToHex())
			}

			utxoInputID := utxoInput.ID()
			output, err := s.UTXOManager.ReadOutputByOutputID(&utxoInputID)
			if err != nil {
				// if we don't have the input, we don't have the history, which is fine.
				//nolint:nilnil,nilerr
				return nil, nil
			}

			if output.Address().String() != address.String() {
				continue
			}

			addressBalanceInputs += int64(output.Amount())
		}

		var addressBalanceOutputs int64
		for _, txOutput := range txEssence.Outputs {
			switch output := txOutput.(type) {
			case *iotago.SigLockedSingleOutput:
				//nolint:forcetypeassert
				if output.Address.(iotago.Address).String() != address.String() {
					continue
				}
				addressBalanceOutputs += int64(output.Amount)
			case *iotago.SigLockedDustAllowanceOutput:
				//nolint:forcetypeassert
				if output.Address.(iotago.Address).String() != address.String() {
					continue
				}
				addressBalanceOutputs += int64(output.Amount)
			default:
				return nil, fmt.Errorf("transaction contains an unsupported output type: msgID: %s", messageID.ToHex())
			}
		}

		milestoneTimestampReferenced, err := s.Database.MilestoneTimestampUnixByIndex(referencedByMilestoneIndex)
		if err != nil {
			return nil, err
		}

		return &transactionHistoryItem{
			MessageID:                    messageID.ToHex(),
			TransactionID:                hex.EncodeToString(txID[:]),
			ReferencedByMilestoneIndex:   referencedByMilestoneIndex,
			MilestoneTimestampReferenced: milestoneTimestampReferenced,
			LedgerInclusionState:         ledgerInclusionState,
			ConflictReason:               conflictReason,
			InputsCount:                  len(txEssence.Inputs),
			OutputsCount:                 len(txEssence.Outputs),
			AddressBalanceChange:         addressBalanceOutputs - addressBalanceInputs,
		}, nil
	}

	txHistoryItems := make([]*transactionHistoryItem, 0, len(messageIDs))
	for messageID := range messageIDs {
		txHistoryItem, err := getTransactionHistoryItem(hornet.MessageIDFromMapKey(messageID))
		if err != nil {
			return nil, errors.WithMessagef(echo.ErrInternalServerError, "get transaction history failed: %s, error: %s", address, err)
		}

		if txHistoryItem == nil {
			// skip if we don't have the history
			continue
		}

		txHistoryItems = append(txHistoryItems, txHistoryItem)
	}

	// sort the results by highest milestone index and lowest messageID
	sort.Slice(txHistoryItems, func(i, j int) bool {
		historyItemLeft := txHistoryItems[i]
		historyItemRight := txHistoryItems[j]

		// if both are referenced by the same milestone, sort by messageID
		if historyItemLeft.ReferencedByMilestoneIndex == historyItemRight.ReferencedByMilestoneIndex {
			return strings.Compare(historyItemLeft.MessageID, historyItemRight.MessageID) < 0
		}

		// sort by milestone index
		return historyItemLeft.ReferencedByMilestoneIndex > historyItemRight.ReferencedByMilestoneIndex
	})

	if len(txHistoryItems) > maxResults {
		txHistoryItems = txHistoryItems[:maxResults]
	}

	return &transactionHistoryResponse{
		AddressType: address.Type(),
		Address:     address.String(),
		MaxResults:  uint32(maxResults),
		Count:       uint32(len(txHistoryItems)),
		History:     txHistoryItems,
		LedgerIndex: ledgerIndex,
	}, nil
}

func transactionHistoryCSV(resp *transactionHistoryResponse) string {
	var csvBuilder strings.Builder

	csvBuilder.WriteString("\"Transaction History\"\n\n")

	csvBuilder.WriteString(fmt.Sprintf("\"Address:\",\"0x%s\"\n", resp.Address))
	csvBuilder.WriteString(fmt.Sprintf("\"LedgerIndex:\",%d\n", resp.LedgerIndex))
	csvBuilder.WriteString(fmt.Sprintf("\"MaxResultsLimitReached:\",\"%t\"\n", resp.MaxResults != 0 && resp.Count == resp.MaxResults))
	csvBuilder.WriteString(fmt.Sprintf("\"Date:\",\"%s\"\n", time.Now().Format(time.RFC3339)))
	csvBuilder.WriteString("\n\"MessageID\",\"TransactionID\",\"ReferencedByMilestoneIndex\",\"MilestoneTimestampReferenced\",\"LedgerInclusionState\",\"ConflictReason\",\"InputsCount\",\"OutputsCount\",\"AddressBalanceChange\"\n")

	// sort the history items by milestoneIndex and messageID to have a deterministic CSV file
	sort.Slice(resp.History, func(i, j int) bool {
		historyItemLeft := resp.History[i]
		historyItemRight := resp.History[j]

		// if both are referenced by the same milestone, sort by messageID
		if historyItemLeft.ReferencedByMilestoneIndex == historyItemRight.ReferencedByMilestoneIndex {
			return strings.Compare(historyItemLeft.MessageID, historyItemRight.MessageID) < 0
		}

		// sort by milestone index
		return historyItemLeft.ReferencedByMilestoneIndex < historyItemRight.ReferencedByMilestoneIndex
	})

	for _, historyItem := range resp.History {
		csvBuilder.WriteString(fmt.Sprintf("\"%s\",", historyItem.MessageID))
		csvBuilder.WriteString(fmt.Sprintf("\"%s\",", historyItem.TransactionID))
		csvBuilder.WriteString(fmt.Sprintf("%d,", historyItem.ReferencedByMilestoneIndex))
		csvBuilder.WriteString(fmt.Sprintf("\"%s\",", time.Unix(historyItem.MilestoneTimestampReferenced, 0).Format(time.RFC3339)))
		csvBuilder.WriteString(fmt.Sprintf("\"%s\",", historyItem.LedgerInclusionState))
		csvBuilder.WriteString(fmt.Sprintf("%d,", historyItem.ConflictReason))
		csvBuilder.WriteString(fmt.Sprintf("%d,", historyItem.InputsCount))
		csvBuilder.WriteString(fmt.Sprintf("%d,", historyItem.OutputsCount))
		csvBuilder.WriteString(fmt.Sprintf("%d\n", historyItem.AddressBalanceChange))
	}

	return csvBuilder.String()
}

func (s *DatabaseServer) transactionHistoryResponseByAddressAndMimeType(c echo.Context, address iotago.Address) error {
	resp, err := s.transactionHistoryByAddress(c, address)
	if err != nil {
		return err
	}

	mimeType, err := httpserver.GetAcceptHeaderContentType(c, MIMETextCSV, echo.MIMEApplicationJSON)
	if err != nil && !errors.Is(err, httpserver.ErrNotAcceptable) {
		return err
	}

	switch mimeType {
	case MIMETextCSV:
		return c.Blob(http.StatusOK, MIMETextCSV, []byte(transactionHistoryCSV(resp)))

	default:
		// default to echo.MIMEApplicationJSON
		return restapi.JSONResponse(c, http.StatusOK, resp)
	}
}
