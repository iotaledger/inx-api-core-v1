package server

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/hive.go/serializer"
	"github.com/iotaledger/inx-api-core-v1/pkg/milestone"
	"github.com/iotaledger/inx-api-core-v1/pkg/restapi"
	"github.com/iotaledger/inx-api-core-v1/pkg/utxo"
	iotago "github.com/iotaledger/iota.go/v2"
)

func newOutputResponse(output *utxo.Output, ledgerIndex milestone.Index) (*OutputResponse, error) {
	var rawOutput iotago.Output
	switch output.OutputType() {
	case iotago.OutputSigLockedSingleOutput:
		rawOutput = &iotago.SigLockedSingleOutput{
			Address: output.Address(),
			Amount:  output.Amount(),
		}
	case iotago.OutputSigLockedDustAllowanceOutput:
		rawOutput = &iotago.SigLockedDustAllowanceOutput{
			Address: output.Address(),
			Amount:  output.Amount(),
		}
	default:
		return nil, errors.WithMessagef(echo.ErrInternalServerError, "unsupported output type: %d", output.OutputType())
	}

	rawOutputJSON, err := rawOutput.MarshalJSON()
	if err != nil {
		return nil, errors.WithMessagef(echo.ErrInternalServerError, "marshaling output failed: %s, error: %s", output.OutputID().ToHex(), err)
	}

	rawRawOutputJSON := json.RawMessage(rawOutputJSON)

	return &OutputResponse{
		MessageID:     output.MessageID().ToHex(),
		TransactionID: hex.EncodeToString(output.OutputID()[:iotago.TransactionIDLength]),
		Spent:         false,
		OutputIndex:   binary.LittleEndian.Uint16(output.OutputID()[iotago.TransactionIDLength : iotago.TransactionIDLength+serializer.UInt16ByteSize]),
		RawOutput:     &rawRawOutputJSON,
		LedgerIndex:   ledgerIndex,
	}, nil
}

func newSpentResponse(spent *utxo.Spent, ledgerIndex milestone.Index) (*OutputResponse, error) {
	response, err := newOutputResponse(spent.Output(), ledgerIndex)
	if err != nil {
		return nil, err
	}
	response.Spent = true
	response.MilestoneIndexSpent = spent.ConfirmationIndex()
	response.TransactionIDSpent = hex.EncodeToString(spent.TargetTransactionID()[:])

	return response, nil
}

func (s *DatabaseServer) outputByID(c echo.Context) (*OutputResponse, error) {
	outputID, err := restapi.ParseOutputIDParam(c)
	if err != nil {
		return nil, err
	}

	ledgerIndex := s.UTXOManager.ReadLedgerIndex()

	output, err := s.UTXOManager.ReadOutputByOutputID(outputID)
	if err != nil {
		if errors.Is(err, kvstore.ErrKeyNotFound) {
			return nil, errors.WithMessagef(echo.ErrNotFound, "output not found: %s", outputID.ToHex())
		}

		return nil, errors.WithMessagef(echo.ErrInternalServerError, "reading output failed: %s, error: %s", outputID.ToHex(), err)
	}

	isUnspent, err := s.UTXOManager.IsOutputUnspent(output)
	if err != nil {
		return nil, errors.WithMessagef(echo.ErrInternalServerError, "reading spent status failed: %s, error: %s", outputID.ToHex(), err)
	}

	if isUnspent {
		return newOutputResponse(output, ledgerIndex)
	}

	spent, err := s.UTXOManager.ReadSpentForOutput(output)
	if err != nil {
		if errors.Is(err, kvstore.ErrKeyNotFound) {
			return nil, errors.WithMessagef(echo.ErrNotFound, "output not found: %s", outputID.ToHex())
		}

		return nil, errors.WithMessagef(echo.ErrInternalServerError, "reading output failed: %s, error: %s", outputID.ToHex(), err)
	}

	return newSpentResponse(spent, ledgerIndex)
}

//nolint:interfacer // false positive
func (s *DatabaseServer) ed25519Balance(address *iotago.Ed25519Address) (*addressBalanceResponse, error) {
	balance, dustAllowed, ledgerIndex, err := s.UTXOManager.AddressBalance(address)
	if err != nil {
		return nil, errors.WithMessagef(echo.ErrInternalServerError, "reading address balance failed: %s, error: %s", address, err)
	}

	return &addressBalanceResponse{
		AddressType: address.Type(),
		Address:     address.String(),
		Balance:     balance,
		DustAllowed: dustAllowed,
		LedgerIndex: ledgerIndex,
	}, nil
}

func (s *DatabaseServer) balanceByBech32Address(c echo.Context) (*addressBalanceResponse, error) {
	bech32Address, err := restapi.ParseBech32AddressParam(c, s.Bech32HRP)
	if err != nil {
		return nil, err
	}

	switch address := bech32Address.(type) {
	case *iotago.Ed25519Address:
		return s.ed25519Balance(address)
	default:
		return nil, errors.WithMessagef(restapi.ErrInvalidParameter, "invalid address: %s, error: unknown address type", address.String())
	}
}

func (s *DatabaseServer) balanceByEd25519Address(c echo.Context) (*addressBalanceResponse, error) {
	address, err := restapi.ParseEd25519AddressParam(c)
	if err != nil {
		return nil, err
	}

	return s.ed25519Balance(address)
}

func (s *DatabaseServer) outputsResponse(address iotago.Address, includeSpent bool, filterType *iotago.OutputType) (*addressOutputsResponse, error) {
	maxResults := s.RestAPILimitsMaxResults

	opts := []utxo.IterateOption{
		utxo.FilterAddress(address),
	}

	if filterType != nil {
		opts = append(opts, utxo.FilterOutputType(*filterType))
	}

	ledgerIndex := s.UTXOManager.ReadLedgerIndex()

	unspentOutputs, err := s.UTXOManager.UnspentOutputs(append(opts, utxo.MaxResultCount(maxResults))...)
	if err != nil {
		return nil, errors.WithMessagef(echo.ErrInternalServerError, "reading unspent outputs failed: %s, error: %s", address, err)
	}

	outputIDs := make([]string, len(unspentOutputs))
	for i, unspentOutput := range unspentOutputs {
		outputIDs[i] = unspentOutput.OutputID().ToHex()
	}

	if includeSpent && maxResults-len(outputIDs) > 0 {

		spents, err := s.UTXOManager.SpentOutputs(append(opts, utxo.MaxResultCount(maxResults-len(outputIDs)))...)
		if err != nil {
			return nil, errors.WithMessagef(echo.ErrInternalServerError, "reading spent outputs failed: %s, error: %s", address, err)
		}

		outputIDsSpent := make([]string, len(spents))
		for i, spent := range spents {
			outputIDsSpent[i] = spent.OutputID().ToHex()
		}

		//nolint:makezero // false positive
		outputIDs = append(outputIDs, outputIDsSpent...)
	}

	return &addressOutputsResponse{
		AddressType: address.Type(),
		Address:     address.String(),
		MaxResults:  uint32(maxResults),
		Count:       uint32(len(outputIDs)),
		OutputIDs:   outputIDs,
		LedgerIndex: ledgerIndex,
	}, nil
}

func (s *DatabaseServer) outputsIDsByBech32Address(c echo.Context) (*addressOutputsResponse, error) {
	// error is ignored because it returns false in case it can't be parsed
	includeSpent, _ := strconv.ParseBool(strings.ToLower(c.QueryParam("include-spent")))

	typeParam := strings.ToLower(c.QueryParam("type"))
	var filteredType *iotago.OutputType

	if len(typeParam) > 0 {
		outputTypeInt, err := strconv.ParseInt(typeParam, 10, 32)
		if err != nil {
			return nil, errors.WithMessagef(restapi.ErrInvalidParameter, "invalid type: %s, error: unknown output type", typeParam)
		}
		outputType := iotago.OutputType(outputTypeInt)
		if outputType != iotago.OutputSigLockedSingleOutput && outputType != iotago.OutputSigLockedDustAllowanceOutput {
			return nil, errors.WithMessagef(restapi.ErrInvalidParameter, "invalid type: %s, error: unknown output type", typeParam)
		}
		filteredType = &outputType
	}

	bech32Address, err := restapi.ParseBech32AddressParam(c, s.Bech32HRP)
	if err != nil {
		return nil, err
	}

	return s.outputsResponse(bech32Address, includeSpent, filteredType)
}

func (s *DatabaseServer) outputsIDsByEd25519Address(c echo.Context) (*addressOutputsResponse, error) {
	// error is ignored because it returns false in case it can't be parsed
	includeSpent, _ := strconv.ParseBool(strings.ToLower(c.QueryParam("include-spent")))

	var filteredType *iotago.OutputType
	typeParam := strings.ToLower(c.QueryParam("type"))
	if len(typeParam) > 0 {
		outputTypeInt, err := strconv.ParseInt(typeParam, 10, 32)
		if err != nil {
			return nil, errors.WithMessagef(restapi.ErrInvalidParameter, "invalid type: %s, error: unknown output type", typeParam)
		}
		outputType := iotago.OutputType(outputTypeInt)
		if outputType != iotago.OutputSigLockedSingleOutput && outputType != iotago.OutputSigLockedDustAllowanceOutput {
			return nil, errors.WithMessagef(restapi.ErrInvalidParameter, "invalid type: %s, error: unknown output type", typeParam)
		}
		filteredType = &outputType
	}

	address, err := restapi.ParseEd25519AddressParam(c)
	if err != nil {
		return nil, err
	}

	return s.outputsResponse(address, includeSpent, filteredType)
}

func (s *DatabaseServer) treasury(_ echo.Context) (*treasuryResponse, error) {

	treasuryOutput, err := s.UTXOManager.UnspentTreasuryOutput()
	if err != nil {
		return nil, err
	}

	return &treasuryResponse{
		MilestoneID: hex.EncodeToString(treasuryOutput.MilestoneID[:]),
		Amount:      treasuryOutput.Amount,
	}, nil
}
