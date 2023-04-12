package utxo

import (
	"encoding/binary"

	"github.com/iotaledger/hive.go/serializer/v2/marshalutil"
	"github.com/iotaledger/inx-api-core-v1/pkg/milestone"
	iotago "github.com/iotaledger/iota.go/v2"
)

// MilestoneDiff represents the generated and spent outputs by a milestone's confirmation.
type MilestoneDiff struct {
	kvStorable
	// The index of the milestone.
	Index milestone.Index
	// The outputs newly generated with this diff.
	Outputs Outputs
	// The outputs spent with this diff.
	Spents Spents
	// The treasury output this diff generated.
	TreasuryOutput *TreasuryOutput
	// The treasury output this diff consumed.
	SpentTreasuryOutput *TreasuryOutput
}

func milestoneDiffKeyForIndex(msIndex milestone.Index) []byte {
	m := marshalutil.New(5)
	m.WriteByte(UTXOStoreKeyPrefixMilestoneDiffs)
	m.WriteUint32(uint32(msIndex))

	return m.Bytes()
}

// note that this method relies on the data being available within other "tables".
func (ms *MilestoneDiff) kvStorableLoad(utxoManager *Manager, key []byte, value []byte) error {
	marshalUtil := marshalutil.New(value)

	outputCount, err := marshalUtil.ReadUint32()
	if err != nil {
		return err
	}

	outputs := make(Outputs, int(outputCount))
	for i := 0; i < int(outputCount); i++ {
		var outputID *iotago.UTXOInputID
		if outputID, err = ParseOutputID(marshalUtil); err != nil {
			return err
		}

		output, err := utxoManager.ReadOutputByOutputID(outputID)
		if err != nil {
			return err
		}

		outputs[i] = output
	}

	spentCount, err := marshalUtil.ReadUint32()
	if err != nil {
		return err
	}

	spents := make(Spents, spentCount)
	for i := 0; i < int(spentCount); i++ {
		var outputID *iotago.UTXOInputID
		if outputID, err = ParseOutputID(marshalUtil); err != nil {
			return err
		}

		spent, err := utxoManager.ReadSpentForOutputID(outputID)
		if err != nil {
			return err
		}

		spents[i] = spent
	}

	hasTreasury, err := marshalUtil.ReadBool()
	if err != nil {
		return err
	}

	if hasTreasury {
		treasuryOutputMilestoneID, err := marshalUtil.ReadBytes(iotago.MilestoneIDLength)
		if err != nil {
			return err
		}

		// try to read from unspent and spent
		treasuryOutput, err := utxoManager.readUnspentTreasuryOutput(treasuryOutputMilestoneID)
		if err != nil {
			treasuryOutput, err = utxoManager.readSpentTreasuryOutput(treasuryOutputMilestoneID)
			if err != nil {
				return err
			}
		}

		ms.TreasuryOutput = treasuryOutput

		spentTreasuryOutputMilestoneID, err := marshalUtil.ReadBytes(iotago.MilestoneIDLength)
		if err != nil {
			return err
		}

		spentTreasuryOutput, err := utxoManager.readSpentTreasuryOutput(spentTreasuryOutputMilestoneID)
		if err != nil {
			return err
		}

		ms.SpentTreasuryOutput = spentTreasuryOutput
	}

	ms.Index = milestone.Index(binary.LittleEndian.Uint32(key[1:]))
	ms.Outputs = outputs
	ms.Spents = spents

	return nil
}

func (u *Manager) MilestoneDiff(msIndex milestone.Index) (*MilestoneDiff, error) {
	key := milestoneDiffKeyForIndex(msIndex)

	value, err := u.utxoStorage.Get(key)
	if err != nil {
		return nil, err
	}

	diff := &MilestoneDiff{}
	if err := diff.kvStorableLoad(u, key, value); err != nil {
		return nil, err
	}

	return diff, nil
}
