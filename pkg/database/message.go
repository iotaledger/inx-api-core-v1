package database

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/hive.go/objectstorage"
	"github.com/iotaledger/hive.go/serializer"
	"github.com/iotaledger/inx-api-core-v1/pkg/hornet"
	iotago "github.com/iotaledger/iota.go/v2"
)

type Message struct {
	objectstorage.StorableObjectFlags

	// Key
	messageID hornet.MessageID

	// Value
	data        []byte
	messageOnce sync.Once
	message     *iotago.Message
}

func (msg *Message) Data() []byte {
	return msg.data
}

func (msg *Message) Message() *iotago.Message {
	msg.messageOnce.Do(func() {
		iotaMsg := &iotago.Message{}
		if _, err := iotaMsg.Deserialize(msg.data, serializer.DeSeriModeNoValidation); err != nil {
			panic(fmt.Sprintf("failed to deserialize message: %v, error: %s", msg.messageID.ToHex(), err))
		}

		msg.message = iotaMsg
	})

	return msg.message
}

func (msg *Message) Transaction() *iotago.Transaction {
	switch payload := msg.Message().Payload.(type) {
	case *iotago.Transaction:
		return payload
	default:
		return nil
	}
}

func (msg *Message) TransactionEssence() *iotago.TransactionEssence {
	if transaction := msg.Transaction(); transaction != nil {
		switch essence := transaction.Essence.(type) {
		case *iotago.TransactionEssence:
			return essence
		default:
			return nil
		}
	}

	return nil
}

func (msg *Message) Milestone() *iotago.Milestone {
	switch payload := msg.Message().Payload.(type) {
	case *iotago.Milestone:
		return payload
	default:
		return nil
	}
}

func messageFactory(key []byte, data []byte) *Message {
	return &Message{
		messageID: hornet.MessageIDFromSlice(key[:iotago.MessageIDLength]),
		data:      data,
	}
}

// MessageOrNil returns a message object.
func (db *Database) MessageOrNil(messageID hornet.MessageID) *Message {
	key := messageID

	data, err := db.messagesStore.Get(key)
	if err != nil {
		if !errors.Is(err, kvstore.ErrKeyNotFound) {
			panic(fmt.Errorf("failed to get value from database: %w", err))
		}

		return nil
	}

	return messageFactory(key, data)
}
