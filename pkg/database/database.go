package database

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/inx-api-core-v1/pkg/milestone"
	"github.com/iotaledger/inx-api-core-v1/pkg/utxo"
)

const (
	DBVersion = 1
)

const (
	StorePrefixMessages             byte = 1
	StorePrefixMessageMetadata      byte = 2
	StorePrefixMilestones           byte = 3
	StorePrefixChildren             byte = 4
	StorePrefixSnapshot             byte = 5
	StorePrefixUnreferencedMessages byte = 6
	StorePrefixIndexation           byte = 7
	StorePrefixHealth               byte = 255
)

type Database struct {
	// databases
	tangleDatabase kvstore.KVStore
	utxoDatabase   kvstore.KVStore

	// kv stores
	messagesStore   kvstore.KVStore
	metadataStore   kvstore.KVStore
	milestonesStore kvstore.KVStore
	snapshotStore   kvstore.KVStore
	childrenStore   kvstore.KVStore
	indexationStore kvstore.KVStore

	// solid entry points
	solidEntryPoints *SolidEntryPoints

	// snapshot info
	snapshot *SnapshotInfo

	// utxo
	utxoManager *utxo.Manager

	// syncstate
	syncState     *SyncState
	syncStateOnce sync.Once
}

func New(tangleDatabase, utxoDatabase kvstore.KVStore, networkID uint64, skipHealthCheck bool) (*Database, error) {

	checkDatabaseHealth := func(store kvstore.KVStore) error {
		healthTracker, err := kvstore.NewStoreHealthTracker(store, kvstore.KeyPrefix{StorePrefixHealth}, DBVersion, nil)
		if err != nil {
			return err
		}

		if lo.PanicOnErr(healthTracker.IsCorrupted()) {
			return errors.New("database is corrupted")
		}

		if lo.PanicOnErr(healthTracker.IsTainted()) {
			return errors.New("database is tainted")
		}

		return nil
	}

	if !skipHealthCheck {
		if err := checkDatabaseHealth(tangleDatabase); err != nil {
			return nil, fmt.Errorf("opening tangle database failed: %w", err)
		}
		if err := checkDatabaseHealth(utxoDatabase); err != nil {
			return nil, fmt.Errorf("opening utxo database failed: %w", err)
		}
	}

	db := &Database{
		tangleDatabase:   tangleDatabase,
		utxoDatabase:     utxoDatabase,
		messagesStore:    lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixMessages})),
		metadataStore:    lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixMessageMetadata})),
		milestonesStore:  lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixMilestones})),
		snapshotStore:    lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixSnapshot})),
		childrenStore:    lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixChildren})),
		indexationStore:  lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixIndexation})),
		solidEntryPoints: nil,
		snapshot:         nil,
		utxoManager:      utxo.New(utxoDatabase),
		syncState:        nil,
		syncStateOnce:    sync.Once{},
	}

	if err := db.loadSnapshotInfo(); err != nil {
		return nil, err
	}

	// check that the database matches to the config network ID
	if networkID != db.snapshot.NetworkID {
		return nil, fmt.Errorf("app is configured to operate in network with ID %d but the database corresponds to ID %d", networkID, db.snapshot.NetworkID)
	}

	if err := db.loadSolidEntryPoints(); err != nil {
		return nil, err
	}

	// delete unused prefixes
	for _, prefix := range []byte{StorePrefixUnreferencedMessages} {
		if err := tangleDatabase.DeletePrefix(kvstore.KeyPrefix{prefix}); err != nil {
			return nil, err
		}
	}

	return db, nil
}

func (db *Database) UTXOManager() *utxo.Manager {
	return db.utxoManager
}

func (db *Database) CloseDatabases() error {
	var flushAndCloseError error
	if err := db.tangleDatabase.Flush(); err != nil {
		flushAndCloseError = err
	}
	if err := db.tangleDatabase.Close(); err != nil {
		flushAndCloseError = err
	}
	if err := db.utxoDatabase.Flush(); err != nil {
		flushAndCloseError = err
	}
	if err := db.utxoDatabase.Close(); err != nil {
		flushAndCloseError = err
	}

	return flushAndCloseError
}

type SyncState struct {
	LatestMilestoneIndex     milestone.Index
	LatestMilestoneTimestamp int64
	ConfirmedMilestoneIndex  milestone.Index
	PruningIndex             milestone.Index
}

func (db *Database) LatestSyncState() *SyncState {
	db.syncStateOnce.Do(func() {
		ledgerIndex := db.utxoManager.ReadLedgerIndex()
		ledgerMilestoneTimestamp := lo.PanicOnErr(db.MilestoneTimestampUnixByIndex(ledgerIndex))

		db.syncState = &SyncState{
			LatestMilestoneIndex:     ledgerIndex,
			LatestMilestoneTimestamp: ledgerMilestoneTimestamp,
			ConfirmedMilestoneIndex:  ledgerIndex,
			PruningIndex:             db.snapshot.PruningIndex,
		}
	})

	return db.syncState
}
