package utxo

import (
	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/hive.go/serializer"
	"github.com/iotaledger/hive.go/serializer/v2/byteutils"
	"github.com/iotaledger/hive.go/serializer/v2/marshalutil"
)

type OutputConsumer func(output *Output) bool

func (o *Output) unspentDatabaseKey() []byte {
	ms := marshalutil.New(69)
	ms.WriteByte(UTXOStoreKeyPrefixUnspent) // 1 byte
	ms.WriteBytes(o.AddressBytes())         // 33 bytes
	ms.WriteByte(o.outputType)              // 1 byte
	ms.WriteBytes(o.outputID[:])            // 34 bytes

	return ms.Bytes()
}

func outputIDBytesFromUnspentDatabaseKey(key []byte) ([]byte, error) {

	ms := marshalutil.New(key)
	_, err := ms.ReadByte() // prefix
	if err != nil {
		return nil, err
	}

	if _, err := parseAddress(ms); err != nil {
		return nil, err
	}

	_, err = ms.ReadByte() // output type
	if err != nil {
		return nil, err
	}

	return ms.ReadBytes(OutputIDLength)
}

func (u *Manager) IsOutputUnspent(output *Output) (bool, error) {
	return u.utxoStorage.Has(output.unspentDatabaseKey())
}

func (u *Manager) ForEachUnspentOutput(consumer OutputConsumer, options ...IterateOption) error {

	consumerFunc := consumer

	opt := iterateOptions(options)

	var innerErr error

	key := []byte{UTXOStoreKeyPrefixUnspent}

	// Filter by address
	if opt.address != nil {
		addrBytes, err := opt.address.Serialize(serializer.DeSeriModeNoValidation)
		if err != nil {
			return err
		}
		key = byteutils.ConcatBytes(key, addrBytes)

		// Filter by type
		if opt.filterOutputType != nil {
			key = byteutils.ConcatBytes(key, []byte{*opt.filterOutputType})
		}
	} else if opt.filterOutputType != nil {

		// Filter results instead of using prefix iteration
		consumerFunc = func(output *Output) bool {
			if output.OutputType() == *opt.filterOutputType {
				return consumer(output)
			}

			return true
		}
	}

	var i int

	if err := u.utxoStorage.IterateKeys(key, func(key kvstore.Key) bool {

		if (opt.maxResultCount > 0) && (i >= opt.maxResultCount) {
			return false
		}

		i++

		outputIDBytes, err := outputIDBytesFromUnspentDatabaseKey(key)
		if err != nil {
			innerErr = err

			return false
		}
		outputKey := byteutils.ConcatBytes([]byte{UTXOStoreKeyPrefixOutput}, outputIDBytes)

		value, err := u.utxoStorage.Get(outputKey)
		if err != nil {
			innerErr = err

			return false
		}

		output := &Output{}
		if err := output.kvStorableLoad(u, outputKey, value); err != nil {
			innerErr = err

			return false
		}

		return consumerFunc(output)
	}); err != nil {
		return err
	}

	return innerErr
}
