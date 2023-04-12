package coreapi

import (
	"github.com/iotaledger/hive.go/app"
)

// ParametersRestAPI contains the definition of the parameters used by the chrysalis API HTTP server.
type ParametersRestAPI struct {
	// BindAddress defines the bind address on which the chrysalis API HTTP server listens.
	BindAddress string `default:"localhost:9094" usage:"the bind address on which the chrysalis API HTTP server listens"`

	// AdvertiseAddress defines the address of the chrysalis API HTTP server which is advertised to the INX Server (optional).
	AdvertiseAddress string `default:"" usage:"the address of the chrysalis API HTTP server which is advertised to the INX Server (optional)"`

	Limits struct {
		// the maximum number of characters that the body of an API call may contain
		MaxBodyLength string `default:"1M" usage:"the maximum number of characters that the body of an API call may contain"`
		// the maximum number of results that may be returned by an endpoint
		MaxResults int `default:"1000" usage:"the maximum number of results that may be returned by an endpoint"`
	}

	// SwaggerEnabled defines whether to provide swagger API documentation under endpoint "/swagger"
	SwaggerEnabled bool `default:"false" usage:"whether to provide swagger API documentation under endpoint \"/swagger\""`

	// DebugRequestLoggerEnabled defines whether the debug logging for requests should be enabled
	DebugRequestLoggerEnabled bool `default:"false" usage:"whether the debug logging for requests should be enabled"`
}

// ParametersProtocol contains the definition of the parameters used by the protocol.
type ParametersProtocol struct {
	// NetworkID defines the network ID this app operates on
	NetworkID string `default:"chrysalis-mainnet" name:"networkID" usage:"the network ID on which this app operates on"`
	// Bech32HRP defines the HRP which should be used for Bech32 addresses
	Bech32HRP string `default:"iota" name:"bech32HRP" usage:"the HRP which should be used for Bech32 addresses"`
}

var ParamsRestAPI = &ParametersRestAPI{}
var ParamsProtocol = &ParametersProtocol{}

var params = &app.ComponentParams{
	Params: map[string]any{
		"restAPI":  ParamsRestAPI,
		"protocol": ParamsProtocol,
	},
	Masked: nil,
}
