package database

import (
	"context"

	"github.com/labstack/echo/v4"
	"go.uber.org/dig"

	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/hive.go/app/shutdown"
	"github.com/iotaledger/hive.go/kvstore"
	hivedb "github.com/iotaledger/hive.go/kvstore/database"
	"github.com/iotaledger/inx-api-core-v1/pkg/daemon"
	"github.com/iotaledger/inx-api-core-v1/pkg/database"
	"github.com/iotaledger/inx-api-core-v1/pkg/database/engine"
)

const (
	DBVersion uint32 = 2
)

func init() {
	Component = &app.Component{
		Name:     "database",
		DepsFunc: func(cDeps dependencies) { deps = cDeps },
		Params:   params,
		Provide:  provide,
		Run:      run,
	}
}

type dependencies struct {
	dig.In
	Database        *database.Database
	Echo            *echo.Echo
	ShutdownHandler *shutdown.ShutdownHandler
}

var (
	Component *app.Component
	deps      dependencies
)

func provide(c *dig.Container) error {

	type storageOut struct {
		dig.Out
		TangleDatabase kvstore.KVStore `name:"tangleDatabase"`
		UTXODatabase   kvstore.KVStore `name:"utxoDatabase"`
	}

	if err := c.Provide(func() (storageOut, error) {
		Component.LogInfo("Setting up database ...")

		tangleDatabase, err := engine.StoreWithDefaultSettings(ParamsDatabase.Tangle.Path, false, hivedb.EngineAuto, engine.AllowedEnginesStorageAuto...)
		if err != nil {
			return storageOut{}, err
		}

		utxoDatabase, err := engine.StoreWithDefaultSettings(ParamsDatabase.UTXO.Path, false, hivedb.EngineAuto, engine.AllowedEnginesStorageAuto...)
		if err != nil {
			return storageOut{}, err
		}

		return storageOut{
			TangleDatabase: tangleDatabase,
			UTXODatabase:   utxoDatabase,
		}, err
	}); err != nil {
		return err
	}

	type storageDeps struct {
		dig.In
		TangleDatabase kvstore.KVStore `name:"tangleDatabase"`
		UTXODatabase   kvstore.KVStore `name:"utxoDatabase"`
		NetworkID      uint64          `name:"networkId"`
	}

	return c.Provide(func(deps storageDeps) (*database.Database, error) {
		store, err := database.New(deps.TangleDatabase, deps.UTXODatabase, deps.NetworkID, ParamsDatabase.Debug)
		if err != nil {
			return nil, err
		}

		store.PrintSnapshotInfo()

		return store, nil
	})
}

func run() error {

	if err := Component.Daemon().BackgroundWorker("Close database", func(ctx context.Context) {
		<-ctx.Done()

		Component.LogInfo("Syncing databases to disk ...")
		if err := deps.Database.CloseDatabases(); err != nil {
			Component.LogPanicf("Syncing databases to disk ... failed: %s", err)
		}
		Component.LogInfo("Syncing databases to disk ... done")
	}, daemon.PriorityStopDatabase); err != nil {
		Component.LogPanicf("failed to start worker: %s", err)
	}

	return nil
}
