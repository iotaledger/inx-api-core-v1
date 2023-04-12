package app

import (
	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/hive.go/app/components/profiling"
	"github.com/iotaledger/hive.go/app/components/shutdown"
	"github.com/iotaledger/inx-api-core-v1/core/coreapi"
	"github.com/iotaledger/inx-api-core-v1/core/database"
	"github.com/iotaledger/inx-api-core-v1/plugins/inx"
	"github.com/iotaledger/inx-api-core-v1/plugins/prometheus"
)

var (
	// Name of the app.
	Name = "inx-api-core-v1"

	// Version of the app.
	Version = "1.0.0-rc.2"
)

func App() *app.App {
	return app.New(Name, Version,
		app.WithInitComponent(InitComponent),
		app.WithComponents(
			shutdown.Component,
			database.Component,
			coreapi.Component,
			inx.Component,
			profiling.Component,
			prometheus.Component,
		),
	)
}

var (
	InitComponent *app.InitComponent
)

func init() {
	InitComponent = &app.InitComponent{
		Component: &app.Component{
			Name: "App",
		},
		NonHiddenFlags: []string{
			"config",
			"help",
			"version",
		},
	}
}
