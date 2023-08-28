package database

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/kvstore"
	hivedb "github.com/iotaledger/hive.go/kvstore/database"
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/inx-api-core-v1/pkg/database/engine"
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
	//nolint:godot,gocritic
	//StorePrefixUTXODeprecated          byte = 8
	StorePrefixConflictingTransactions byte = 9
	StorePrefixHealth                  byte = 255
)

type Database struct {
	// databases
	tangleDatabase kvstore.KVStore
	utxoDatabase   kvstore.KVStore

	// kv stores
	messagesStore                kvstore.KVStore
	metadataStore                kvstore.KVStore
	milestonesStore              kvstore.KVStore
	snapshotStore                kvstore.KVStore
	childrenStore                kvstore.KVStore
	indexationStore              kvstore.KVStore
	conflictingTransactionsStore kvstore.KVStore

	// snapshot info
	snapshot *SnapshotInfo

	// utxo
	utxoManager *utxo.Manager

	// syncstate
	syncState     *SyncState
	syncStateOnce sync.Once
}

func New(log *logger.Logger, tangleDatabasePath string, utxoDatabasePath string, networkID uint64, skipHealthCheck bool) (*Database, error) {

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

	initDatabase := func(readonly bool) (*Database, error) {
		tangleDatabase, err := engine.StoreWithDefaultSettings(tangleDatabasePath, false, hivedb.EngineAuto, readonly, engine.AllowedEnginesStorageAuto...)
		if err != nil {
			return nil, fmt.Errorf("opening tangle database failed: %w", err)
		}

		utxoDatabase, err := engine.StoreWithDefaultSettings(utxoDatabasePath, false, hivedb.EngineAuto, readonly, engine.AllowedEnginesStorageAuto...)
		if err != nil {
			return nil, fmt.Errorf("opening utxo database failed: %w", err)
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
			tangleDatabase:               tangleDatabase,
			utxoDatabase:                 utxoDatabase,
			messagesStore:                lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixMessages})),
			metadataStore:                lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixMessageMetadata})),
			milestonesStore:              lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixMilestones})),
			snapshotStore:                lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixSnapshot})),
			childrenStore:                lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixChildren})),
			indexationStore:              lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixIndexation})),
			conflictingTransactionsStore: lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixConflictingTransactions})),
			snapshot:                     nil,
			utxoManager:                  utxo.New(utxoDatabase),
			syncState:                    nil,
			syncStateOnce:                sync.Once{},
		}

		if err := db.loadSnapshotInfo(); err != nil {
			return nil, err
		}

		// check that the database matches to the config network ID
		if networkID != db.snapshot.NetworkID {
			return nil, fmt.Errorf("app is configured to operate in network with ID %d but the database corresponds to ID %d", networkID, db.snapshot.NetworkID)
		}

		return db, nil
	}

	// first we open the database in readonly mode
	db, err := initDatabase(true)
	if err != nil {
		return nil, err
	}

	// we need to check if the conflicting transactions lookup table is up to date
	conflictingTransactionsLookupTableUpToDate, err := db.checkConflictingTransactionsMessageIDsStatus()
	if err != nil {
		_ = db.CloseDatabases()
		return nil, err
	}

	// in case not, we rebuild the lookup table
	// therefore we need to reopen the database in write mode
	if !conflictingTransactionsLookupTableUpToDate {
		log.Infof("Conflicting transactions store not up to date. Updating now... (this may take some time!)")

		// first we close the readonly databases
		if err := db.CloseDatabases(); err != nil {
			return nil, fmt.Errorf("failed to close readonly databases: error: %w", err)
		}

		// we need to temporarily open the database in write mode
		db, err = initDatabase(false)
		if err != nil {
			return nil, err
		}

		if err := db.createConflictingTransactionsMessageIDsLookupTable(); err != nil {
			_ = db.CloseDatabases()
			return nil, fmt.Errorf("failed to create conflicting transactions lookup table: error: %w", err)
		}

		// close the write mode databases
		if err := db.CloseDatabases(); err != nil {
			return nil, fmt.Errorf("failed to close readonly databases: error: %w", err)
		}

		log.Infof("Updating conflicting transactions store done!")

		// initialize again in readonly mode
		db, err = initDatabase(true)
		if err != nil {
			return nil, err
		}
	}

	return db, nil
}

func (db *Database) UTXOManager() *utxo.Manager {
	return db.utxoManager
}

func (db *Database) CloseDatabases() error {
	var closeError error
	if err := db.tangleDatabase.Close(); err != nil {
		closeError = err
	}
	if err := db.utxoDatabase.Close(); err != nil {
		closeError = err
	}

	return closeError
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
