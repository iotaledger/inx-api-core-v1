package app

import (
	"github.com/iotaledger/hive.go/core/app"
	"github.com/iotaledger/hive.go/core/app/core/shutdown"
	"github.com/iotaledger/hive.go/core/app/plugins/profiling"
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
		app.WithCoreComponents([]*app.CoreComponent{
			shutdown.CoreComponent,
			database.CoreComponent,
			coreapi.CoreComponent,
		}...),
		app.WithPlugins([]*app.Plugin{
			inx.Plugin,
			profiling.Plugin,
			prometheus.Plugin,
		}...),
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
