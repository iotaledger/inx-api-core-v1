package utxo

import (
	"github.com/iotaledger/hive.go/serializer"
	"github.com/iotaledger/hive.go/serializer/v2/byteutils"
	"github.com/iotaledger/hive.go/serializer/v2/marshalutil"
	"github.com/iotaledger/inx-api-core-v1/pkg/hornet"
	iotago "github.com/iotaledger/iota.go/v2"
)

const (
	OutputIDLength = iotago.TransactionIDLength + serializer.UInt16ByteSize
)

type Outputs []*Output

type Output struct {
	kvStorable

	outputID   *iotago.UTXOInputID
	messageID  hornet.MessageID
	outputType iotago.OutputType
	address    iotago.Address
	amount     uint64
}

func (o *Output) OutputID() *iotago.UTXOInputID {
	return o.outputID
}

func (o *Output) MessageID() hornet.MessageID {
	return o.messageID
}

func (o *Output) OutputType() iotago.OutputType {
	return o.outputType
}

func (o *Output) Address() iotago.Address {
	return o.address
}

func (o *Output) Amount() uint64 {
	return o.amount
}

func (o *Output) AddressBytes() []byte {
	// This never throws an error for current Ed25519 addresses
	bytes, _ := o.address.Serialize(serializer.DeSeriModeNoValidation)

	return bytes
}

func (o *Output) kvStorableLoad(_ *Manager, key []byte, value []byte) error {

	// Parse key
	keyUtil := marshalutil.New(key)

	// Read prefix output
	_, err := keyUtil.ReadByte()
	if err != nil {
		return err
	}

	// Read OutputID
	if o.outputID, err = ParseOutputID(keyUtil); err != nil {
		return err
	}

	// Parse value
	valueUtil := marshalutil.New(value)

	// Read MessageID
	if o.messageID, err = ParseMessageID(valueUtil); err != nil {
		return err
	}

	// Read OutputType
	o.outputType, err = valueUtil.ReadByte()
	if err != nil {
		return err
	}

	// Read Address
	if o.address, err = parseAddress(valueUtil); err != nil {
		return err
	}

	// Read Amount
	o.amount, err = valueUtil.ReadUint64()
	if err != nil {
		return err
	}

	return nil
}

func (u *Manager) ReadOutputByOutputID(outputID *iotago.UTXOInputID) (*Output, error) {
	key := byteutils.ConcatBytes([]byte{UTXOStoreKeyPrefixOutput}, outputID[:])
	value, err := u.utxoStorage.Get(key)
	if err != nil {
		return nil, err
	}

	output := &Output{}
	if err := output.kvStorableLoad(u, key, value); err != nil {
		return nil, err
	}

	return output, nil
}
