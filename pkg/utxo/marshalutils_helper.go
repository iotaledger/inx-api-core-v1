package utxo

import (
	"github.com/iotaledger/hive.go/serializer"
	"github.com/iotaledger/hive.go/serializer/v2/marshalutil"
	"github.com/iotaledger/inx-api-core-v1/pkg/hornet"
	iotago "github.com/iotaledger/iota.go/v2"
)

func ParseOutputID(ms *marshalutil.MarshalUtil) (*iotago.UTXOInputID, error) {
	bytes, err := ms.ReadBytes(OutputIDLength)
	if err != nil {
		return nil, err
	}
	o := &iotago.UTXOInputID{}
	copy(o[:], bytes)

	return o, nil
}

func parseTransactionID(ms *marshalutil.MarshalUtil) (*iotago.TransactionID, error) {
	bytes, err := ms.ReadBytes(iotago.TransactionIDLength)
	if err != nil {
		return nil, err
	}
	t := &iotago.TransactionID{}
	copy(t[:], bytes)

	return t, nil
}

func ParseMessageID(ms *marshalutil.MarshalUtil) (hornet.MessageID, error) {
	bytes, err := ms.ReadBytes(iotago.MessageIDLength)
	if err != nil {
		return nil, err
	}

	return hornet.MessageIDFromSlice(bytes), nil
}

func parseAddress(ms *marshalutil.MarshalUtil) (iotago.Address, error) {

	addrType, err := ms.ReadByte()
	if err != nil {
		return nil, err
	}

	// Move the cursor back
	ms.ReadSeek(-1)

	addr, err := iotago.AddressSelector(uint32(addrType))
	if err != nil {
		return nil, err
	}

	//nolint: forcetypeassert
	address := addr.(iotago.Address)

	pre := ms.ReadOffset()
	readBytes, err := address.Deserialize(ms.ReadRemainingBytes(), serializer.DeSeriModePerformValidation)
	if err != nil {
		return nil, err
	}
	post := ms.ReadOffset()

	bytesReadTooFar := post - pre - readBytes
	// Move the cursor back some bytes
	ms.ReadSeek(-bytesReadTooFar)

	return address, err
}
