package database

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/ds/bitmask"
	"github.com/iotaledger/hive.go/serializer/v2/marshalutil"
	"github.com/iotaledger/inx-api-core-v1/pkg/milestone"
)

var (
	ErrParseSnapshotInfoFailed = errors.New("Parsing of snapshot info failed")
)

type SnapshotInfo struct {
	NetworkID       uint64
	SnapshotIndex   milestone.Index
	EntryPointIndex milestone.Index
	PruningIndex    milestone.Index
	Timestamp       time.Time
	Metadata        bitmask.BitMask
}

func (db *Database) readSnapshotInfo() (*SnapshotInfo, error) {
	value, err := db.snapshotStore.Get([]byte("snapshotInfo"))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to retrieve snapshot info", err)
	}

	info, err := snapshotInfoFromBytes(value)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to convert snapshot info", err)
	}

	return info, nil
}

func (db *Database) loadSnapshotInfo() error {
	info, err := db.readSnapshotInfo()
	if err != nil {
		return err
	}
	db.snapshot = info

	return nil
}

func (db *Database) PrintSnapshotInfo() {
	if db.snapshot != nil {
		println(fmt.Sprintf(`SnapshotInfo:
    NetworkID: %d
    SnapshotIndex: %d
    EntryPointIndex: %d
    PruningIndex: %d
    Timestamp: %v`, db.snapshot.NetworkID, db.snapshot.SnapshotIndex, db.snapshot.EntryPointIndex, db.snapshot.PruningIndex, db.snapshot.Timestamp.Truncate(time.Second)))
	}
}

func snapshotInfoFromBytes(bytes []byte) (*SnapshotInfo, error) {

	if len(bytes) != 29 {
		return nil, errors.Wrapf(ErrParseSnapshotInfoFailed, "invalid length %d != %d", len(bytes), 54)
	}

	marshalUtil := marshalutil.New(bytes)

	networkID, err := marshalUtil.ReadUint64()
	if err != nil {
		return nil, err
	}

	snapshotIndex, err := marshalUtil.ReadUint32()
	if err != nil {
		return nil, err
	}

	entryPointIndex, err := marshalUtil.ReadUint32()
	if err != nil {
		return nil, err
	}

	pruningIndex, err := marshalUtil.ReadUint32()
	if err != nil {
		return nil, err
	}

	timestamp, err := marshalUtil.ReadUint64()
	if err != nil {
		return nil, err
	}

	metadata, err := marshalUtil.ReadByte()
	if err != nil {
		return nil, err
	}

	return &SnapshotInfo{
		NetworkID:       networkID,
		SnapshotIndex:   milestone.Index(snapshotIndex),
		EntryPointIndex: milestone.Index(entryPointIndex),
		PruningIndex:    milestone.Index(pruningIndex),
		Timestamp:       time.Unix(int64(timestamp), 0),
		Metadata:        bitmask.BitMask(metadata),
	}, nil
}
