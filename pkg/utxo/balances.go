//nolint:nonamedreturns
package utxo

import (
	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/hive.go/serializer"
	"github.com/iotaledger/hive.go/serializer/v2/byteutils"
	"github.com/iotaledger/hive.go/serializer/v2/marshalutil"
	"github.com/iotaledger/inx-api-core-v1/pkg/milestone"
	iotago "github.com/iotaledger/iota.go/v2"
)

func balanceFromBytes(value []byte) (balance uint64, dustAllowanceBalance uint64, outputCount int64, err error) {
	marshalUtil := marshalutil.New(value)

	if balance, err = marshalUtil.ReadUint64(); err != nil {
		return
	}

	if dustAllowanceBalance, err = marshalUtil.ReadUint64(); err != nil {
		return
	}

	if outputCount, err = marshalUtil.ReadInt64(); err != nil {
		return
	}

	return
}

func (u *Manager) AddressBalance(address iotago.Address) (balance uint64, dustAllowed bool, ledgerIndex milestone.Index, err error) {

	ledgerIndex = u.ReadLedgerIndex()

	balance, dustAllowed, err = u.AddressBalanceWithoutLocking(address)
	if err != nil {
		return 0, false, 0, err
	}

	return balance, dustAllowed, ledgerIndex, err
}

func (u *Manager) AddressBalanceWithoutLocking(address iotago.Address) (balance uint64, dustAllowed bool, err error) {

	addressKey, err := address.Serialize(serializer.DeSeriModeNoValidation)
	if err != nil {
		return 0, false, err
	}

	b, dustAllowance, dustOutputCount, err := u.readBalanceForAddress(addressKey)
	if err != nil {
		return 0, false, err
	}

	// There is no built-in min function for int64, so inline one here
	min := func(x, y int64) int64 {
		if x > y {
			return y
		}

		return x
	}

	dustAllowed = min(int64(dustAllowance)/iotago.DustAllowanceDivisor, iotago.MaxDustOutputsOnAddress) > dustOutputCount

	return b, dustAllowed, nil
}

func (u *Manager) readBalanceForAddress(addressKey []byte) (balance uint64, dustAllowanceBalance uint64, dustOutputCount int64, err error) {

	dbKey := byteutils.ConcatBytes([]byte{UTXOStoreKeyPrefixBalances}, addressKey)

	value, err := u.utxoStorage.Get(dbKey)
	if err != nil {
		if errors.Is(err, kvstore.ErrKeyNotFound) {
			// No dust information found in the database for this address
			return 0, 0, 0, nil
		}

		return 0, 0, 0, err
	}

	return balanceFromBytes(value)
}
