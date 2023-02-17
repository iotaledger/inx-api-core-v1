package coreapi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/dig"

	"github.com/iotaledger/hive.go/core/app"
	"github.com/iotaledger/inx-api-core-v1/pkg/daemon"
	"github.com/iotaledger/inx-api-core-v1/pkg/database"
	"github.com/iotaledger/inx-api-core-v1/pkg/restapi"
	"github.com/iotaledger/inx-api-core-v1/pkg/server"
	"github.com/iotaledger/inx-app/pkg/httpserver"
	iotago "github.com/iotaledger/iota.go/v2"
)

func init() {
	CoreComponent = &app.CoreComponent{
		Component: &app.Component{
			Name:           "CoreAPIV1",
			DepsFunc:       func(cDeps dependencies) { deps = cDeps },
			Params:         params,
			InitConfigPars: initConfigPars,
			Provide:        provide,
			Run:            run,
		},
	}
}

type dependencies struct {
	dig.In
	AppInfo       *app.Info
	Database      *database.Database
	Echo          *echo.Echo
	NetworkIDName string               `name:"networkIdName"`
	Bech32HRP     iotago.NetworkPrefix `name:"bech32HRP"`
}

var (
	CoreComponent *app.CoreComponent
	deps          dependencies
)

func initConfigPars(c *dig.Container) error {

	type cfgResult struct {
		dig.Out
		RestAPIBindAddress      string               `name:"restAPIBindAddress"`
		RestAPIAdvertiseAddress string               `name:"restAPIAdvertiseAddress"`
		NetworkID               uint64               `name:"networkId"`
		NetworkIDName           string               `name:"networkIdName"`
		Bech32HRP               iotago.NetworkPrefix `name:"bech32HRP"`
	}

	if err := c.Provide(func() cfgResult {
		return cfgResult{
			RestAPIBindAddress:      ParamsRestAPI.BindAddress,
			RestAPIAdvertiseAddress: ParamsRestAPI.AdvertiseAddress,
			NetworkID:               iotago.NetworkIDFromString(ParamsProtocol.NetworkID),
			NetworkIDName:           ParamsProtocol.NetworkID,
			Bech32HRP:               iotago.NetworkPrefix(ParamsProtocol.Bech32HRP),
		}
	}); err != nil {
		CoreComponent.LogPanic(err)
	}

	return nil
}

func provide(c *dig.Container) error {

	if err := c.Provide(func() *echo.Echo {
		e := httpserver.NewEcho(
			CoreComponent.Logger(),
			nil,
			ParamsRestAPI.DebugRequestLoggerEnabled,
		)
		e.Use(middleware.Gzip())
		e.Use(middleware.BodyLimit(ParamsRestAPI.Limits.MaxBodyLength))

		// "api/v1" had a custom error handler
		e.HTTPErrorHandler = func(err error, c echo.Context) {
			var statusCode int
			var message string

			var e *echo.HTTPError
			if errors.As(err, &e) {
				statusCode = e.Code
				message = fmt.Sprintf("%s, error: %s", e.Message, err)
			} else {
				statusCode = http.StatusInternalServerError
				message = fmt.Sprintf("internal server error. error: %s", err)
			}

			_ = c.JSON(statusCode, restapi.HTTPErrorResponseEnvelope{Error: restapi.HTTPErrorResponse{Code: strconv.Itoa(statusCode), Message: message}})
		}

		return e
	}); err != nil {
		return err
	}

	return nil
}

func run() error {

	// create a background worker that handles the API
	if err := CoreComponent.Daemon().BackgroundWorker("API", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting API server ...")

		swagger := server.CreateEchoSwagger(deps.Echo, deps.AppInfo.Version, ParamsRestAPI.SwaggerEnabled)

		//nolint:contextcheck //false positive
		_ = server.NewDatabaseServer(
			swagger,
			deps.AppInfo,
			deps.Database,
			deps.Database.UTXOManager(),
			deps.NetworkIDName,
			deps.Bech32HRP,
			ParamsRestAPI.Limits.MaxResults,
		)

		deps.Echo.Server.BaseContext = func(l net.Listener) context.Context {
			// set BaseContext to be the same as the worker,
			// so that requests being processed don't hang the shutdown procedure
			return ctx
		}

		go func() {
			CoreComponent.LogInfof("You can now access the API using: http://%s", ParamsRestAPI.BindAddress)
			if err := deps.Echo.Start(ParamsRestAPI.BindAddress); err != nil && !errors.Is(err, http.ErrServerClosed) {
				CoreComponent.LogErrorfAndExit("Stopped REST-API server due to an error (%s)", err)
			}
		}()

		CoreComponent.LogInfo("Starting API server ... done")
		<-ctx.Done()
		CoreComponent.LogInfo("Stopping API server ...")

		shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCtxCancel()

		//nolint:contextcheck // false positive
		if err := deps.Echo.Shutdown(shutdownCtx); err != nil {
			CoreComponent.LogWarn(err)
		}

		CoreComponent.LogInfo("Stopping API server... done")
	}, daemon.PriorityStopDatabaseAPI); err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}

	return nil
}
