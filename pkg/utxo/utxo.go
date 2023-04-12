package utxo

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/inx-api-core-v1/pkg/milestone"
	iotago "github.com/iotaledger/iota.go/v2"
)

type Manager struct {
	utxoStorage kvstore.KVStore

	// ledgerIndex
	ledgerIndex     milestone.Index
	ledgerIndexOnce sync.Once
}

func New(store kvstore.KVStore) *Manager {
	return &Manager{
		utxoStorage:     store,
		ledgerIndex:     0,
		ledgerIndexOnce: sync.Once{},
	}
}

func (u *Manager) ReadLedgerIndex() milestone.Index {
	u.ledgerIndexOnce.Do(func() {
		value, err := u.utxoStorage.Get([]byte{UTXOStoreKeyPrefixLedgerMilestoneIndex})
		if err != nil {
			panic(fmt.Errorf("failed to load ledger milestone index: %w", err))
		}

		u.ledgerIndex = milestone.Index(binary.LittleEndian.Uint32(value))
	})

	return u.ledgerIndex
}

type IterateOptions struct {
	address          iotago.Address
	maxResultCount   int
	filterOutputType *iotago.OutputType
}

type IterateOption func(*IterateOptions)

func FilterAddress(address iotago.Address) IterateOption {
	return func(args *IterateOptions) {
		args.address = address
	}
}

func MaxResultCount(count int) IterateOption {
	return func(args *IterateOptions) {
		args.maxResultCount = count
	}
}

func FilterOutputType(outputType iotago.OutputType) IterateOption {
	return func(args *IterateOptions) {
		args.filterOutputType = &outputType
	}
}

func iterateOptions(optionalOptions []IterateOption) *IterateOptions {
	result := &IterateOptions{
		address:          nil,
		maxResultCount:   0,
		filterOutputType: nil,
	}

	for _, optionalOption := range optionalOptions {
		optionalOption(result)
	}

	return result
}

func (u *Manager) SpentOutputs(options ...IterateOption) (Spents, error) {

	var spents []*Spent

	consumerFunc := func(spent *Spent) bool {
		spents = append(spents, spent)

		return true
	}

	if err := u.ForEachSpentOutput(consumerFunc, options...); err != nil {
		return nil, err
	}

	return spents, nil
}

func (u *Manager) UnspentOutputs(options ...IterateOption) ([]*Output, error) {

	var outputs []*Output
	consumerFunc := func(output *Output) bool {
		outputs = append(outputs, output)

		return true
	}

	if err := u.ForEachUnspentOutput(consumerFunc, options...); err != nil {
		return nil, err
	}

	return outputs, nil
}
