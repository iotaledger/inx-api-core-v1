package utxo

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/hive.go/serializer/v2/marshalutil"
	iotago "github.com/iotaledger/iota.go/v2"
)

const (
	// A prefix which denotes a spent treasury output.
	// Do not modify the value since we're writing this as a bool.
	TreasuryOutputSpentPrefix = 1
	// A prefix which denotes an unspent treasury output.
	// Do not modify the value since we're writing this as a bool.
	TreasuryOutputUnspentPrefix = 0
)

var (
	// ErrInvalidTreasuryState is returned when the state of the treasury is invalid.
	ErrInvalidTreasuryState = errors.New("invalid treasury state")
)

// TreasuryOutput represents the output of a treasury transaction.
type TreasuryOutput struct {
	// The ID of the milestone which generated this output.
	MilestoneID iotago.MilestoneID
	// The amount residing on this output.
	Amount uint64
	// Whether this output was already spent
	Spent bool
}

func (t *TreasuryOutput) kvStorableLoad(_ *Manager, key []byte, value []byte) error {
	keyExt := marshalutil.New(key)
	// skip prefix
	if _, err := keyExt.ReadByte(); err != nil {
		return err
	}

	spent, err := keyExt.ReadBool()
	if err != nil {
		return err
	}

	milestoneID, err := keyExt.ReadBytes(iotago.MilestoneIDLength)
	if err != nil {
		return err
	}
	copy(t.MilestoneID[:], milestoneID)

	val := marshalutil.New(value)
	t.Amount, err = val.ReadUint64()
	if err != nil {
		return err
	}

	t.Spent = spent

	return nil
}

func (u *Manager) readSpentTreasuryOutput(msHash []byte) (*TreasuryOutput, error) {
	key := append([]byte{UTXOStoreKeyPrefixTreasuryOutput, TreasuryOutputSpentPrefix}, msHash...)
	val, err := u.utxoStorage.Get(key)
	if err != nil {
		return nil, err
	}
	to := &TreasuryOutput{}
	if err := to.kvStorableLoad(u, key, val); err != nil {
		return nil, err
	}

	return to, nil
}

func (u *Manager) readUnspentTreasuryOutput(msHash []byte) (*TreasuryOutput, error) {
	key := append([]byte{UTXOStoreKeyPrefixTreasuryOutput, TreasuryOutputUnspentPrefix}, msHash...)
	val, err := u.utxoStorage.Get(key)
	if err != nil {
		return nil, err
	}
	to := &TreasuryOutput{}
	if err := to.kvStorableLoad(u, key, val); err != nil {
		return nil, err
	}

	return to, nil
}

// UnspentTreasuryOutput returns the unspent treasury output.
func (u *Manager) UnspentTreasuryOutput() (*TreasuryOutput, error) {
	var i int
	var innerErr error
	var unspentTreasuryOutput *TreasuryOutput
	if err := u.utxoStorage.Iterate([]byte{UTXOStoreKeyPrefixTreasuryOutput, TreasuryOutputUnspentPrefix}, func(key kvstore.Key, value kvstore.Value) bool {
		i++
		unspentTreasuryOutput = &TreasuryOutput{}
		if err := unspentTreasuryOutput.kvStorableLoad(u, key, value); err != nil {
			innerErr = err

			return false
		}

		return true
	}); err != nil {
		return nil, err
	}

	if innerErr != nil {
		return nil, innerErr
	}

	switch {
	case i > 1:
		return nil, fmt.Errorf("%w: more than one unspent treasury output exists", ErrInvalidTreasuryState)
	case i == 0:
		return nil, fmt.Errorf("%w: no treasury output exists", ErrInvalidTreasuryState)
	}

	return unspentTreasuryOutput, nil
}
