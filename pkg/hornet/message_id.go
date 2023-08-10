package hornet

import (
	"encoding/hex"
	"fmt"

	iotago "github.com/iotaledger/iota.go/v2"
)

// MessageID is the ID of a Message.
type MessageID []byte

// MessageIDs is a slice of MessageID.
type MessageIDs []MessageID

// ToHex converts the MessageID to its hex representation.
func (m MessageID) ToHex() string {
	return hex.EncodeToString(m)
}

// ToMapKey converts the MessageID to a string that can be used as a map key.
func (m MessageID) ToMapKey() string {
	return string(m)
}

// MessageIDFromHex creates a MessageID from a hex string representation.
func MessageIDFromHex(hexString string) (MessageID, error) {

	b, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, err
	}

	if len(b) != iotago.MessageIDLength {
		return nil, fmt.Errorf("unknown messageID length (%d)", len(b))
	}

	return MessageID(b), nil
}

// MessageIDFromMapKey creates a MessageID from a map key representation.
func MessageIDFromMapKey(mapKey string) MessageID {
	if len(mapKey) != iotago.MessageIDLength {
		panic(fmt.Sprintf("unknown messageID length (%d)", len(mapKey)))
	}

	return MessageID(mapKey)
}

// MessageIDFromSlice creates a MessageID from a byte slice.
func MessageIDFromSlice(b []byte) MessageID {

	if len(b) != iotago.MessageIDLength {
		panic(fmt.Sprintf("unknown messageID length (%d)", len(b)))
	}

	return MessageID(b)
}

// ToHex converts the MessageIDs to their hex string representation.
func (m MessageIDs) ToHex() []string {
	results := make([]string, len(m))
	for i, msgID := range m {
		results[i] = msgID.ToHex()
	}

	return results
}
