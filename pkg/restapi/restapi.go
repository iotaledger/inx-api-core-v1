package restapi

import (
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/iotaledger/inx-api-core-v1/pkg/hornet"
	"github.com/iotaledger/inx-api-core-v1/pkg/milestone"
	"github.com/iotaledger/inx-api-core-v1/pkg/utxo"
	iotago "github.com/iotaledger/iota.go/v2"
)

const (
	// ParameterMessageID is used to identify a message by its ID.
	ParameterMessageID = "messageID"

	// ParameterTransactionID is used to identify a transaction by its ID.
	ParameterTransactionID = "transactionID"

	// ParameterOutputID is used to identify an output by its ID.
	ParameterOutputID = "outputID"

	// ParameterAddress is used to identify an address.
	ParameterAddress = "address"

	// ParameterMilestoneIndex is used to identify a milestone.
	ParameterMilestoneIndex = "milestoneIndex"

	// QueryParameterPageSize is used to define the page size for the results.
	QueryParameterPageSize = "pageSize"
)

var (
	// ErrInvalidParameter defines the invalid parameter error.
	ErrInvalidParameter = echo.NewHTTPError(http.StatusBadRequest, "invalid parameter")
)

// JSONResponse wraps the result into a "data" field and sends the JSON response with status code.
func JSONResponse(c echo.Context, statusCode int, result interface{}) error {
	return c.JSON(statusCode, &HTTPOkResponseEnvelope{Data: result})
}

// HTTPErrorResponse defines the error struct for the HTTPErrorResponseEnvelope.
type HTTPErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HTTPErrorResponseEnvelope defines the error response schema for node API responses.
type HTTPErrorResponseEnvelope struct {
	Error HTTPErrorResponse `json:"error"`
}

// HTTPOkResponseEnvelope defines the ok response schema for node API responses.
type HTTPOkResponseEnvelope struct {
	// The response is encapsulated in the Data field.
	Data interface{} `json:"data"`
}

func ParseMessageIDParam(c echo.Context) (hornet.MessageID, error) {
	messageIDHex := strings.ToLower(c.Param(ParameterMessageID))

	messageID, err := hornet.MessageIDFromHex(messageIDHex)
	if err != nil {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid message ID: %s, error: %s", messageIDHex, err)
	}

	return messageID, nil
}

func ParseTransactionIDParam(c echo.Context) (*iotago.TransactionID, error) {
	transactionIDHex := strings.ToLower(c.Param(ParameterTransactionID))

	transactionIDBytes, err := hex.DecodeString(transactionIDHex)
	if err != nil {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid transaction ID: %s, error: %s", transactionIDHex, err)
	}

	if len(transactionIDBytes) != iotago.TransactionIDLength {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid transaction ID: %s, invalid length: %d", transactionIDHex, len(transactionIDBytes))
	}

	var transactionID iotago.TransactionID
	copy(transactionID[:], transactionIDBytes)

	return &transactionID, nil
}

func ParseOutputIDParam(c echo.Context) (*iotago.UTXOInputID, error) {
	outputIDParam := strings.ToLower(c.Param(ParameterOutputID))

	outputIDBytes, err := hex.DecodeString(outputIDParam)
	if err != nil {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid output ID: %s, error: %s", outputIDParam, err)
	}

	if len(outputIDBytes) != utxo.OutputIDLength {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid output ID: %s, error: %s", outputIDParam, err)
	}

	var outputID iotago.UTXOInputID
	copy(outputID[:], outputIDBytes)

	return &outputID, nil
}

func ParseBech32AddressParam(c echo.Context, prefix iotago.NetworkPrefix) (iotago.Address, error) {
	addressParam := strings.ToLower(c.Param(ParameterAddress))

	hrp, bech32Address, err := iotago.ParseBech32(addressParam)
	if err != nil {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid address: %s, error: %s", addressParam, err)
	}

	if hrp != prefix {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid bech32 address, expected prefix: %s", prefix)
	}

	return bech32Address, nil
}

func ParseEd25519AddressParam(c echo.Context) (*iotago.Ed25519Address, error) {
	addressParam := strings.ToLower(c.Param(ParameterAddress))

	addressBytes, err := hex.DecodeString(addressParam)
	if err != nil {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid address: %s, error: %s", addressParam, err)
	}

	if len(addressBytes) != (iotago.Ed25519AddressBytesLength) {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid address length: %s", addressParam)
	}

	var address iotago.Ed25519Address
	copy(address[:], addressBytes)

	return &address, nil
}

func ParseMilestoneIndexParam(c echo.Context) (milestone.Index, error) {
	milestoneIndex := strings.ToLower(c.Param(ParameterMilestoneIndex))
	if milestoneIndex == "" {
		return 0, errors.WithMessagef(ErrInvalidParameter, "parameter \"%s\" not specified", ParameterMilestoneIndex)
	}

	msIndex, err := strconv.ParseUint(milestoneIndex, 10, 32)
	if err != nil {
		return 0, errors.WithMessagef(ErrInvalidParameter, "invalid milestone index: %s, error: %s", milestoneIndex, err)
	}

	return milestone.Index(msIndex), nil
}
