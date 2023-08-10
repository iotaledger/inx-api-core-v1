package server

import (
	"encoding/hex"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/iotaledger/inx-api-core-v1/pkg/database"
	"github.com/iotaledger/inx-api-core-v1/pkg/hornet"
	"github.com/iotaledger/inx-api-core-v1/pkg/milestone"
	"github.com/iotaledger/inx-api-core-v1/pkg/restapi"
	iotago "github.com/iotaledger/iota.go/v2"
)

func (s *DatabaseServer) messageMetadataByMessageID(messageID hornet.MessageID) (*messageMetadataResponse, error) {

	msgMeta := s.Database.MessageMetadataOrNil(messageID)
	if msgMeta == nil {
		return nil, errors.WithMessagef(echo.ErrNotFound, "message not found: %s", messageID.ToHex())
	}

	var referencedByMilestone *milestone.Index
	referenced, referencedIndex := msgMeta.ReferencedWithIndex()
	if referenced {
		referencedByMilestone = &referencedIndex
	}

	messageMetadataResponse := &messageMetadataResponse{
		MessageID:                  msgMeta.MessageID().ToHex(),
		Parents:                    msgMeta.Parents().ToHex(),
		Solid:                      msgMeta.IsSolid(),
		ReferencedByMilestoneIndex: referencedByMilestone,
	}

	if msgMeta.IsMilestone() {
		messageMetadataResponse.MilestoneIndex = referencedByMilestone
	}

	if referenced {
		inclusionState := "noTransaction"

		conflict := msgMeta.Conflict()

		if conflict != database.ConflictNone {
			inclusionState = "conflicting"
			messageMetadataResponse.ConflictReason = &conflict
		} else if msgMeta.IsIncludedTxInLedger() {
			inclusionState = "included"
		}

		messageMetadataResponse.LedgerInclusionState = &inclusionState
	}
	/*
		else if msgMeta.IsSolid() {
			// in the node API, we calculated the quality of the tip here, but this is unused in "read-only" mode.
		}
	*/

	return messageMetadataResponse, nil
}

func (s *DatabaseServer) messageByMessageID(messageID hornet.MessageID) (*iotago.Message, error) {
	msg := s.Database.MessageOrNil(messageID)
	if msg == nil {
		return nil, errors.WithMessagef(echo.ErrNotFound, "message not found: %s", messageID.ToHex())
	}

	return msg.Message(), nil
}

func (s *DatabaseServer) messageBytesByMessageID(messageID hornet.MessageID) ([]byte, error) {
	msg := s.Database.MessageOrNil(messageID)
	if msg == nil {
		return nil, errors.WithMessagef(echo.ErrNotFound, "message not found: %s", messageID.ToHex())
	}

	return msg.Data(), nil
}

func (s *DatabaseServer) childrenIDsByMessageID(c echo.Context, messageID hornet.MessageID) (*childrenResponse, error) {
	maxResults := s.maxResultsFromContext(c)

	childrenMessageIDs, err := s.Database.ChildrenMessageIDs(messageID, maxResults)
	if err != nil {
		return nil, errors.WithMessage(echo.ErrInternalServerError, err.Error())
	}

	return &childrenResponse{
		MessageID:  messageID.ToHex(),
		MaxResults: uint32(maxResults),
		Count:      uint32(len(childrenMessageIDs)),
		Children:   childrenMessageIDs.ToHex(),
	}, nil
}

func (s *DatabaseServer) messageIDsByIndex(c echo.Context) (*messageIDsByIndexResponse, error) {
	maxResults := s.maxResultsFromContext(c)

	index := c.QueryParam("index")

	if index == "" {
		return nil, errors.WithMessage(restapi.ErrInvalidParameter, "query parameter index empty")
	}

	indexBytes, err := hex.DecodeString(index)
	if err != nil {
		return nil, errors.WithMessage(restapi.ErrInvalidParameter, "query parameter index invalid hex")
	}

	if len(indexBytes) > database.IndexationIndexLength {
		return nil, errors.WithMessage(restapi.ErrInvalidParameter, fmt.Sprintf("query parameter index too long, max. %d bytes but is %d", database.IndexationIndexLength, len(indexBytes)))
	}

	indexMessageIDs, err := s.Database.IndexMessageIDs(indexBytes, maxResults)
	if err != nil {
		return nil, errors.WithMessage(echo.ErrInternalServerError, err.Error())
	}

	return &messageIDsByIndexResponse{
		Index:      index,
		MaxResults: uint32(maxResults),
		Count:      uint32(len(indexMessageIDs)),
		MessageIDs: indexMessageIDs.ToHex(),
	}, nil
}
