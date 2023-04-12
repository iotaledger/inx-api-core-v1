package database

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/hive.go/serializer"
	"github.com/iotaledger/inx-api-core-v1/pkg/hornet"
	"github.com/iotaledger/inx-api-core-v1/pkg/milestone"
	iotago "github.com/iotaledger/iota.go/v2"
)

var (
	ErrMilestoneNotFound = errors.New("milestone not found")
)

func databaseKeyForMilestoneIndex(milestoneIndex milestone.Index) []byte {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, uint32(milestoneIndex))

	return bytes
}

func milestoneIndexFromDatabaseKey(key []byte) milestone.Index {
	return milestone.Index(binary.LittleEndian.Uint32(key))
}

func milestoneFactory(key []byte, data []byte) *Milestone {
	return &Milestone{
		Index:     milestoneIndexFromDatabaseKey(key),
		MessageID: hornet.MessageIDFromSlice(data[:iotago.MessageIDLength]),
		Timestamp: time.Unix(int64(binary.LittleEndian.Uint64(data[iotago.MessageIDLength:iotago.MessageIDLength+serializer.UInt64ByteSize])), 0),
	}
}

type Milestone struct {
	Index     milestone.Index
	MessageID hornet.MessageID
	Timestamp time.Time
}

// MilestoneOrNil returns a milestone object.
func (db *Database) MilestoneOrNil(milestoneIndex milestone.Index) *Milestone {
	key := databaseKeyForMilestoneIndex(milestoneIndex)
	data, err := db.milestonesStore.Get(key)
	if err != nil {
		if !errors.Is(err, kvstore.ErrKeyNotFound) {
			panic(fmt.Errorf("failed to get value from database: %w", err))
		}

		return nil
	}

	return milestoneFactory(key, data)
}

// MilestoneTimestampUnixByIndex returns the unix timestamp of a milestone.
func (db *Database) MilestoneTimestampUnixByIndex(milestoneIndex milestone.Index) (int64, error) {
	ms := db.MilestoneOrNil(milestoneIndex)
	if ms == nil {
		return 0, ErrMilestoneNotFound
	}

	return ms.Timestamp.Unix(), nil
}
