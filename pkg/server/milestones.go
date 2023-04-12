package server

import (
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/iotaledger/inx-api-core-v1/pkg/restapi"

	"github.com/iotaledger/hive.go/kvstore"
)

func (s *DatabaseServer) milestoneByIndex(c echo.Context) (*milestoneResponse, error) {

	msIndex, err := restapi.ParseMilestoneIndexParam(c)
	if err != nil {
		return nil, err
	}

	ms := s.Database.MilestoneOrNil(msIndex)
	if ms == nil {
		return nil, errors.WithMessagef(echo.ErrNotFound, "milestone not found: %d", msIndex)
	}

	return &milestoneResponse{
		Index:     uint32(ms.Index),
		MessageID: ms.MessageID.ToHex(),
		Time:      ms.Timestamp.Unix(),
	}, nil
}

func (s *DatabaseServer) milestoneUTXOChangesByIndex(c echo.Context) (*milestoneUTXOChangesResponse, error) {

	msIndex, err := restapi.ParseMilestoneIndexParam(c)
	if err != nil {
		return nil, err
	}

	diff, err := s.UTXOManager.MilestoneDiff(msIndex)
	if err != nil {
		if errors.Is(err, kvstore.ErrKeyNotFound) {
			return nil, errors.WithMessagef(echo.ErrNotFound, "can't load milestone diff for index: %d, error: %s", msIndex, err)
		}

		return nil, errors.WithMessagef(echo.ErrInternalServerError, "can't load milestone diff for index: %d, error: %s", msIndex, err)
	}

	createdOutputs := make([]string, len(diff.Outputs))
	consumedOutputs := make([]string, len(diff.Spents))

	for i, output := range diff.Outputs {
		createdOutputs[i] = output.OutputID().ToHex()
	}

	for i, output := range diff.Spents {
		consumedOutputs[i] = output.OutputID().ToHex()
	}

	return &milestoneUTXOChangesResponse{
		Index:           uint32(msIndex),
		CreatedOutputs:  createdOutputs,
		ConsumedOutputs: consumedOutputs,
	}, nil
}
