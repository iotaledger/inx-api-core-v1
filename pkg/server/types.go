package server

import (
	"encoding/json"

	"github.com/iotaledger/inx-api-core-v1/pkg/database"
	"github.com/iotaledger/inx-api-core-v1/pkg/milestone"
	"github.com/iotaledger/inx-api-core-v1/pkg/utxo"
)

// infoResponse defines the response of a GET info REST API call.
type infoResponse struct {
	// The name of the node software.
	Name string `json:"name"`
	// The semver version of the node software.
	Version string `json:"version"`
	// Whether the node is healthy.
	IsHealthy bool `json:"isHealthy"`
	// The ID of the network.
	NetworkID string `json:"networkId"`
	// The Bech32 HRP used.
	//nolint:tagliatelle
	Bech32HRP string `json:"bech32HRP"`
	// The minimum pow score of the network.
	MinPoWScore float64 `json:"minPoWScore"`
	// The current rate of new messages per second.
	MessagesPerSecond float64 `json:"messagesPerSecond"`
	// The current rate of referenced messages per second.
	ReferencedMessagesPerSecond float64 `json:"referencedMessagesPerSecond"`
	// The ratio of referenced messages in relation to new messages of the last confirmed milestone.
	ReferencedRate float64 `json:"referencedRate"`
	// The timestamp of the latest known milestone.
	LatestMilestoneTimestamp int64 `json:"latestMilestoneTimestamp"`
	// The latest known milestone index.
	LatestMilestoneIndex milestone.Index `json:"latestMilestoneIndex"`
	// The current confirmed milestone's index.
	ConfirmedMilestoneIndex milestone.Index `json:"confirmedMilestoneIndex"`
	// The milestone index at which the last pruning commenced.
	PruningIndex milestone.Index `json:"pruningIndex"`
	// The features this node exposes.
	Features []string `json:"features"`
}

// receiptsResponse defines the response of a receipts REST API call.
type receiptsResponse struct {
	Receipts []*utxo.ReceiptTuple `json:"receipts"`
}

// messageMetadataResponse defines the response of a GET message metadata REST API call.
type messageMetadataResponse struct {
	// The hex encoded message ID of the message.
	MessageID string `json:"messageId"`
	// The hex encoded message IDs of the parents the message references.
	Parents []string `json:"parentMessageIds"`
	// Whether the message is solid.
	Solid bool `json:"isSolid"`
	// The milestone index that references this message.
	ReferencedByMilestoneIndex *milestone.Index `json:"referencedByMilestoneIndex,omitempty"`
	// If this message represents a milestone this is the milestone index
	MilestoneIndex *milestone.Index `json:"milestoneIndex,omitempty"`
	// The ledger inclusion state of the transaction payload.
	LedgerInclusionState *string `json:"ledgerInclusionState,omitempty"`
	// The reason why this message is marked as conflicting.
	ConflictReason *database.Conflict `json:"conflictReason,omitempty"`
	// Whether the message should be promoted.
	ShouldPromote *bool `json:"shouldPromote,omitempty"`
	// Whether the message should be reattached.
	ShouldReattach *bool `json:"shouldReattach,omitempty"`
}

// childrenResponse defines the response of a GET children REST API call.
type childrenResponse struct {
	// The hex encoded message ID of the message.
	MessageID string `json:"messageId"`
	// The maximum count of results that are returned by the node.
	MaxResults uint32 `json:"maxResults"`
	// The actual count of results that are returned.
	Count uint32 `json:"count"`
	// The hex encoded message IDs of the children of this message.
	Children []string `json:"childrenMessageIds"`
}

// messageIDsByIndexResponse defines the response of a GET messages REST API call.
type messageIDsByIndexResponse struct {
	// The index of the messages.
	Index string `json:"index"`
	// The maximum count of results that are returned by the node.
	MaxResults uint32 `json:"maxResults"`
	// The actual count of results that are returned.
	Count uint32 `json:"count"`
	// The hex encoded message IDs of the found messages with this index.
	MessageIDs []string `json:"messageIds"`
}

// milestoneResponse defines the response of a GET milestones REST API call.
type milestoneResponse struct {
	// The index of the milestone.
	Index uint32 `json:"index"`
	// The hex encoded ID of the message containing the milestone.
	MessageID string `json:"messageId"`
	// The unix time of the milestone payload.
	Time int64 `json:"timestamp"`
}

// milestoneUTXOChangesResponse defines the response of a GET milestone UTXO changes REST API call.
type milestoneUTXOChangesResponse struct {
	// The index of the milestone.
	Index uint32 `json:"index"`
	// The output IDs (transaction hash + output index) of the newly created outputs.
	CreatedOutputs []string `json:"createdOutputs"`
	// The output IDs (transaction hash + output index) of the consumed (spent) outputs.
	ConsumedOutputs []string `json:"consumedOutputs"`
}

// OutputResponse defines the response of a GET outputs REST API call.
type OutputResponse struct {
	// The hex encoded message ID of the message.
	MessageID string `json:"messageId"`
	// The hex encoded transaction id from which this output originated.
	TransactionID string `json:"transactionId"`
	// The index of the output.
	OutputIndex uint16 `json:"outputIndex"`
	// Whether this output is spent.
	Spent bool `json:"isSpent"`
	// The milestone index at which this output was spent.
	MilestoneIndexSpent milestone.Index `json:"milestoneIndexSpent,omitempty"`
	// The transaction this output was spent with.
	TransactionIDSpent string `json:"transactionIdSpent,omitempty"`
	// The ledger index at which this output was queried at.
	LedgerIndex milestone.Index `json:"ledgerIndex"`
	// The output in its serialized form.
	RawOutput *json.RawMessage `json:"output"`
}

// addressBalanceResponse defines the response of a GET addresses REST API call.
type addressBalanceResponse struct {
	// The type of the address (0=Ed25519).
	AddressType byte `json:"addressType"`
	// The hex encoded address.
	Address string `json:"address"`
	// The balance of the address.
	Balance uint64 `json:"balance"`
	// Indicates if dust is allowed on this address.
	DustAllowed bool `json:"dustAllowed"`
	// The ledger index at which this balance was queried at.
	LedgerIndex milestone.Index `json:"ledgerIndex"`
}

// addressOutputsResponse defines the response of a GET outputs by address REST API call.
type addressOutputsResponse struct {
	// The type of the address (0=Ed25519).
	AddressType byte `json:"addressType"`
	// The hex encoded address.
	Address string `json:"address"`
	// The maximum count of results that are returned by the node.
	MaxResults uint32 `json:"maxResults"`
	// The actual count of results that are returned.
	Count uint32 `json:"count"`
	// The output IDs (transaction hash + output index) of the outputs on this address.
	OutputIDs []string `json:"outputIds"`
	// The ledger index at which these outputs where queried at.
	LedgerIndex milestone.Index `json:"ledgerIndex"`
}

// treasuryResponse defines the response of a GET treasury REST API call.
type treasuryResponse struct {
	MilestoneID string `json:"milestoneId"`
	Amount      uint64 `json:"amount"`
}

// transactionHistoryItem is an item of the transactionHistoryResponse.
type transactionHistoryItem struct {
	// The hex encoded message ID of the message in which the transaction payload was included.
	MessageID string `json:"messageId"`
	// The hex encoded transaction id.
	TransactionID string `json:"transactionId"`
	// The milestone index that references this message.
	ReferencedByMilestoneIndex milestone.Index `json:"referencedByMilestoneIndex"`
	// The milestone timestamp that references this message.
	MilestoneTimestampReferenced int64 `json:"milestoneTimestampReferenced"`
	// The ledger inclusion state of the transaction payload.
	LedgerInclusionState string `json:"ledgerInclusionState"`
	// The reason why this message is marked as conflicting.
	ConflictReason *database.Conflict `json:"conflictReason,omitempty"`
	// The amount of inputs in the transaction payload.
	InputsCount int `json:"inputsCount"`
	// The amount of outputs in the transaction payload.
	OutputsCount int `json:"outputsCount"`
	// The balance change of the address the history was queried for.
	AddressBalanceChange int64 `json:"addressBalanceChange"`
}

// transactionHistoryResponse defines the response of a GET address transaction history REST API call.
type transactionHistoryResponse struct {
	// The type of the address (0=Ed25519).
	AddressType byte `json:"addressType"`
	// The hex encoded address.
	Address string `json:"address"`
	// The maximum count of results that are returned by the node.
	MaxResults uint32 `json:"maxResults"`
	// The actual count of results that are returned.
	Count uint32 `json:"count"`
	// The transaction history of this address.
	History []*transactionHistoryItem `json:"history"`
	// The ledger index at which the history was queried at.
	LedgerIndex milestone.Index `json:"ledgerIndex"`
}
