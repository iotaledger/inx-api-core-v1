package server

import (
	"github.com/labstack/echo/v4"
	"github.com/pangpanglabs/echoswagger/v2"

	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/inx-api-core-v1/pkg/database"
	restapipkg "github.com/iotaledger/inx-api-core-v1/pkg/restapi"
	"github.com/iotaledger/inx-api-core-v1/pkg/utxo"
	"github.com/iotaledger/inx-app/pkg/httpserver"
	iotago "github.com/iotaledger/iota.go/v2"
)

const (
	APIRoute = "/api/core/v1"
)

type DatabaseServer struct {
	AppInfo                 *app.Info
	Database                *database.Database
	UTXOManager             *utxo.Manager
	NetworkIDName           string
	Bech32HRP               iotago.NetworkPrefix
	RestAPILimitsMaxResults int
}

func NewDatabaseServer(swagger echoswagger.ApiRoot, appInfo *app.Info, db *database.Database, utxoManager *utxo.Manager, networkIDName string, bech32HRP iotago.NetworkPrefix, maxResults int) *DatabaseServer {
	s := &DatabaseServer{
		AppInfo:                 appInfo,
		Database:                db,
		UTXOManager:             utxoManager,
		NetworkIDName:           networkIDName,
		Bech32HRP:               bech32HRP,
		RestAPILimitsMaxResults: maxResults,
	}

	s.configureRoutes(swagger.Group("root", APIRoute))

	return s
}

func CreateEchoSwagger(e *echo.Echo, version string, enabled bool) echoswagger.ApiRoot {
	if !enabled {
		return echoswagger.NewNop(e)
	}

	echoSwagger := echoswagger.New(e, "/swagger", &echoswagger.Info{
		Title:       "inx-api-core-v1 API",
		Description: "REST/RPC API for IOTA chrysalis",
		Version:     version,
	})

	echoSwagger.SetExternalDocs("Find out more about inx-api-core-v1", "https://wiki.iota.org/shimmer/inx-api-core-v1/welcome/")
	echoSwagger.SetUI(echoswagger.UISetting{DetachSpec: false, HideTop: false})
	echoSwagger.SetScheme("http", "https")
	echoSwagger.SetRequestContentType(echo.MIMEApplicationJSON)
	echoSwagger.SetResponseContentType(echo.MIMEApplicationJSON)

	return echoSwagger
}

func (s *DatabaseServer) maxResultsFromContext(c echo.Context) int {
	maxPageSize := uint32(s.RestAPILimitsMaxResults)
	if len(c.QueryParam(restapipkg.QueryParameterPageSize)) > 0 {
		pageSizeQueryParam, err := httpserver.ParseUint32QueryParam(c, restapipkg.QueryParameterPageSize, maxPageSize)
		if err != nil {
			return int(maxPageSize)
		}

		if pageSizeQueryParam < maxPageSize {
			// use the smaller page size given by the request context
			maxPageSize = pageSizeQueryParam
		}
	}

	return int(maxPageSize)
}
