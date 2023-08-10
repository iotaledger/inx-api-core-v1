package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pangpanglabs/echoswagger/v2"

	restapipkg "github.com/iotaledger/inx-api-core-v1/pkg/restapi"
)

const (
	// RouteInfo is the route for getting the node info.
	// GET returns the node info.
	RouteInfo = "/info"

	// RouteMessageData is the route for getting message data by its messageID.
	// GET returns message data (json).
	RouteMessageData = "/messages/:" + restapipkg.ParameterMessageID

	// RouteMessageMetadata is the route for getting message metadata by its messageID.
	// GET returns message metadata (including info about "promotion/reattachment needed").
	RouteMessageMetadata = RouteMessageData + "/metadata"

	// RouteMessageBytes is the route for getting message raw data by it's messageID.
	// GET returns raw message data (bytes).
	RouteMessageBytes = RouteMessageData + "/raw"

	// RouteMessageChildren is the route for getting message IDs of the children of a message, identified by its messageID.
	// GET returns the message IDs of all children.
	RouteMessageChildren = RouteMessageData + "/children"

	// RouteMessages is the route for getting message IDs or creating new messages.
	// GET with query parameter (mandatory) returns all message IDs that fit these filter criteria (query parameters: "index").
	// POST creates a single new message and returns the new message ID.
	RouteMessages = "/messages"

	// RouteTransactionsIncludedMessageData is the route for getting the message that was included in the ledger for a given transaction ID.
	// GET returns message data (json).
	RouteTransactionsIncludedMessageData = "/transactions/:" + restapipkg.ParameterTransactionID + "/included-message"

	// RouteTransactionsIncludedMessageMetadata is the route for getting the message metadata that was included in the ledger for a given transaction ID.
	// GET returns message metadata (including info about "promotion/reattachment needed").
	RouteTransactionsIncludedMessageMetadata = RouteTransactionsIncludedMessageData + "/metadata"

	// RouteTransactionsIncludedMessageBytes is the route for getting the message raw data that was included in the ledger for a given transaction ID.
	// GET returns raw message data (bytes).
	RouteTransactionsIncludedMessageBytes = RouteTransactionsIncludedMessageData + "/raw"

	// RouteTransactionsIncludedMessageChildren is the route for getting the message IDs of the children of the message that was included in the ledger for a given transaction ID.
	// GET returns the message IDs of all children.
	RouteTransactionsIncludedMessageChildren = RouteTransactionsIncludedMessageData + "/children"

	// RouteMilestone is the route for getting a milestone by it's milestoneIndex.
	// GET returns the milestone.
	RouteMilestone = "/milestones/:" + restapipkg.ParameterMilestoneIndex

	// RouteMilestoneUTXOChanges is the route for getting all UTXO changes of a milestone by its milestoneIndex.
	// GET returns the output IDs of all UTXO changes.
	RouteMilestoneUTXOChanges = RouteMilestone + "/utxo-changes"

	// RouteOutput is the route for getting outputs by their outputID (transactionHash + outputIndex).
	// GET returns the output.
	RouteOutput = "/outputs/:" + restapipkg.ParameterOutputID

	// RouteAddressBech32Balance is the route for getting the total balance of all unspent outputs of an address.
	// The address must be encoded in bech32.
	// GET returns the balance of all unspent outputs of this address.
	RouteAddressBech32Balance = "/addresses/:" + restapipkg.ParameterAddress

	// RouteAddressEd25519Balance is the route for getting the total balance of all unspent outputs of an ed25519 address.
	// The ed25519 address must be encoded in hex.
	// GET returns the balance of all unspent outputs of this address.
	RouteAddressEd25519Balance = "/addresses/ed25519/:" + restapipkg.ParameterAddress

	// RouteAddressBech32Outputs is the route for getting all output IDs for an address.
	// The address must be encoded in bech32.
	// GET returns the outputIDs for all outputs of this address (optional query parameters: "include-spent").
	RouteAddressBech32Outputs = "/addresses/:" + restapipkg.ParameterAddress + "/outputs"

	// RouteAddressEd25519Outputs is the route for getting all output IDs for an ed25519 address.
	// The ed25519 address must be encoded in hex.
	// GET returns the outputIDs for all outputs of this address (optional query parameters: "include-spent").
	RouteAddressEd25519Outputs = "/addresses/ed25519/:" + restapipkg.ParameterAddress + "/outputs"

	// RouteAddressBech32History is the route for getting the transaction history of an address.
	// The address must be encoded in bech32.
	// GET returns the tx-history of this address.
	RouteAddressBech32History = "/addresses/:" + restapipkg.ParameterAddress + "/tx-history"

	// RouteAddressEd25519History is the route for getting the transaction history of an ed25519 address.
	// The ed25519 address must be encoded in hex.
	// GET returns the tx-history of this address.
	RouteAddressEd25519History = "/addresses/ed25519/:" + restapipkg.ParameterAddress + "/tx-history"

	// RouteTreasury is the route for getting the current treasury output.
	RouteTreasury = "/treasury"

	// RouteReceipts is the route for getting all stored receipts.
	RouteReceipts = "/receipts"

	// RouteReceiptsMigratedAtIndex is the route for getting all receipts for a given migrated at index.
	RouteReceiptsMigratedAtIndex = "/receipts/:" + restapipkg.ParameterMilestoneIndex
)

func (s *DatabaseServer) configureRoutes(routeGroup echoswagger.ApiGroup) {

	routeGroup.GET(RouteInfo, func(c echo.Context) error {
		resp, err := s.info()
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteMessageData, func(c echo.Context) error {
		messageID, err := restapipkg.ParseMessageIDParam(c)
		if err != nil {
			return err
		}

		resp, err := s.messageByMessageID(messageID)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteMessageMetadata, func(c echo.Context) error {
		messageID, err := restapipkg.ParseMessageIDParam(c)
		if err != nil {
			return err
		}

		resp, err := s.messageMetadataByMessageID(messageID)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteMessageBytes, func(c echo.Context) error {
		messageID, err := restapipkg.ParseMessageIDParam(c)
		if err != nil {
			return err
		}

		resp, err := s.messageBytesByMessageID(messageID)
		if err != nil {
			return err
		}

		return c.Blob(http.StatusOK, echo.MIMEOctetStream, resp)
	})

	routeGroup.GET(RouteMessageChildren, func(c echo.Context) error {
		messageID, err := restapipkg.ParseMessageIDParam(c)
		if err != nil {
			return err
		}

		resp, err := s.childrenIDsByMessageID(c, messageID)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteMessages, func(c echo.Context) error {
		resp, err := s.messageIDsByIndex(c)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteTransactionsIncludedMessageData, func(c echo.Context) error {
		messageID, err := s.messageIDByTransactionID(c)
		if err != nil {
			return err
		}

		resp, err := s.messageByMessageID(messageID)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteTransactionsIncludedMessageMetadata, func(c echo.Context) error {
		messageID, err := s.messageIDByTransactionID(c)
		if err != nil {
			return err
		}

		resp, err := s.messageMetadataByMessageID(messageID)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteTransactionsIncludedMessageBytes, func(c echo.Context) error {
		messageID, err := s.messageIDByTransactionID(c)
		if err != nil {
			return err
		}

		resp, err := s.messageBytesByMessageID(messageID)
		if err != nil {
			return err
		}

		return c.Blob(http.StatusOK, echo.MIMEOctetStream, resp)
	})

	routeGroup.GET(RouteTransactionsIncludedMessageChildren, func(c echo.Context) error {
		messageID, err := s.messageIDByTransactionID(c)
		if err != nil {
			return err
		}

		resp, err := s.childrenIDsByMessageID(c, messageID)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteMilestone, func(c echo.Context) error {
		resp, err := s.milestoneByIndex(c)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteMilestoneUTXOChanges, func(c echo.Context) error {
		resp, err := s.milestoneUTXOChangesByIndex(c)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteOutput, func(c echo.Context) error {
		resp, err := s.outputByID(c)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteAddressBech32Balance, func(c echo.Context) error {
		resp, err := s.balanceByBech32Address(c)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteAddressEd25519Balance, func(c echo.Context) error {
		resp, err := s.balanceByEd25519Address(c)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteAddressBech32Outputs, func(c echo.Context) error {
		address, err := restapipkg.ParseBech32AddressParam(c, s.Bech32HRP)
		if err != nil {
			return err
		}

		resp, err := s.outputIDsByAddress(c, address)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteAddressEd25519Outputs, func(c echo.Context) error {
		address, err := restapipkg.ParseEd25519AddressParam(c)
		if err != nil {
			return err
		}

		resp, err := s.outputIDsByAddress(c, address)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteAddressBech32History, func(c echo.Context) error {
		address, err := restapipkg.ParseBech32AddressParam(c, s.Bech32HRP)
		if err != nil {
			return err
		}

		return s.transactionHistoryResponseByAddressAndMimeType(c, address)
	})

	routeGroup.GET(RouteAddressEd25519History, func(c echo.Context) error {
		address, err := restapipkg.ParseEd25519AddressParam(c)
		if err != nil {
			return err
		}

		return s.transactionHistoryResponseByAddressAndMimeType(c, address)
	})

	routeGroup.GET(RouteTreasury, func(c echo.Context) error {
		resp, err := s.treasury(c)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteReceipts, func(c echo.Context) error {
		resp, err := s.receipts(c)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})

	routeGroup.GET(RouteReceiptsMigratedAtIndex, func(c echo.Context) error {
		resp, err := s.receiptsByMigratedAtIndex(c)
		if err != nil {
			return err
		}

		return restapipkg.JSONResponse(c, http.StatusOK, resp)
	})
}
