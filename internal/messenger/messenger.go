package messenger

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/RockX-SG/frost-dkg-demo/internal/node"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/RockX-SG/frost-dkg-demo/internal/logger"
	"github.com/RockX-SG/frost-dkg-demo/internal/workers"
	"github.com/bloxapp/ssv-spec/dkg"
	"github.com/bloxapp/ssv-spec/types"
)

var (
	DefaultTopic = "default"
)

type Messenger struct {
	Topics map[string]*Topic
	Data   map[string]*DataStore

	Incoming chan *Message

	logger *logger.Logger
}

func (m *Messenger) WithLogger(logger *logger.Logger) {
	m.logger = logger
}

type Topic struct {
	Name        string
	Subscribers map[string]*Subscriber
}

type Subscriber struct {
	Name         string            `json:"name"`
	SrvAddr      string            `json:"srv_addr"`
	SubscribesTo map[string]*Topic `json:"-"`
	Outgoing     chan *Message     `json:"-"`
	RetryData    map[string]int    `json:"-"`
}

type Message struct {
	Topic string
	Data  []byte
}

type DataStore struct {
	SessionOutputs map[uint64]*node.Output
	InitMsg        *node.Init
	SessionID      [24]byte

	DKGOutputs  map[types.OperatorID]*dkg.SignedOutput
	BlameOutput *dkg.BlameOutput
}

func (m *Messenger) Publish(topicName string, data []byte) error {
	tp, exist := m.Topics[topicName]
	if !exist {
		m.logger.Errorf("Publish: topic %s already exists", topicName)
		return &ErrTopicNotFound{TopicName: topicName}
	}

	m.Incoming <- &Message{Topic: tp.Name, Data: data}
	return nil
}

func (m *Messenger) ProcessIncomingMessageWorker(ctx *context.Context) {
	for msg := range m.Incoming {
		tp, exist := m.Topics[msg.Topic]
		if !exist {
			var err = &ErrTopicNotFound{TopicName: msg.Topic}
			m.logger.Errorf("ProcessIncomingMessageWorker: %w", err)
			continue
		}

		transportMsg := &node.SignedTransport{}
		if err := transportMsg.UnmarshalSSZ(msg.Data); err != nil {
			m.logger.Errorf("ProcessIncomingMessageWorker: %w", err)
			continue
		}

		m.logger.Debugf(
			"received message from %d for msgType %d",
			transportMsg.Signer,
			transportMsg.Message.Type,
		)

		// start a new session
		if transportMsg.Message.Type == node.InitMessageType {
			msgID := hex.EncodeToString(transportMsg.Message.Identifier[:])
			if m.Data[msgID] != nil {
				m.logger.Errorf("session ID already exists")
				continue
			}

			init := &node.Init{}
			if err := init.UnmarshalSSZ(transportMsg.Message.Data); err != nil {
				m.logger.Errorf("could not decoded init data")
				continue
			}

			m.Data[msgID] = &DataStore{
				InitMsg:        init,
				SessionID:      transportMsg.Message.Identifier,
				SessionOutputs: map[uint64]*node.Output{},
			}

			m.logger.Infof("Starting new session %s", msgID)
		}

		// If output message, store for later
		if transportMsg.Message.Type == node.OutputMessageType {
			msgID := hex.EncodeToString(transportMsg.Message.Identifier[:])
			if m.Data[msgID] == nil {
				m.logger.Errorf("session ID doesn't exist")
				continue
			}

			// TODO verify signer part of session

			output := &node.Output{}
			if err := output.UnmarshalSSZ(transportMsg.Message.Data); err != nil {
				m.logger.Errorf("could not decoded output data")
				continue
			}

			// TODO - node.VerifyRSA()

			m.Data[msgID].SessionOutputs[transportMsg.Signer] = output

			if len(m.Data[msgID].SessionOutputs) == len(m.Data[msgID].InitMsg.Operators) {
				// finished
				m.logger.Infof("Session %s FINISHED!", msgID)
			}
		}

		for _, subscriber := range tp.Subscribers {
			// commented out so nodes send themselves
			//operatorID := strconv.Itoa(int(transportMsg.Signer))
			//if operatorID == subscriber.Name {
			//	continue
			//}
			subscriber.Outgoing <- msg
		}
	}
}

const (
	maxRetriesAllowed = 10
)

func (s *Subscriber) ProcessOutgoingMessageWorker(ctx *context.Context) {

	log := (*ctx).Value(workers.Ctxlog("logger"))
	if log == nil {
		panic("logger not found in context")
	}
	logger := log.(*logger.Logger)
	logger.Infof("ProcessOutgoingMessageWorker: logger loaded successfully")

	for msg := range s.Outgoing {

		h := sha256.Sum256(msg.Data)
		k := base64.RawStdEncoding.EncodeToString(h[:])

		numRetries, ok := s.RetryData[k]
		if ok {
			if numRetries >= maxRetriesAllowed {
				continue
			} else {
				s.RetryData[k]++
			}
		} else {
			s.RetryData[k] = 0
		}

		if numRetries > 0 {
			time.Sleep(2 * (time.Second))
		}

		_, exist := s.SubscribesTo[msg.Topic]
		if !exist {
			var err = &ErrTopicNotFound{TopicName: msg.Topic}
			logger.Errorf("ProcessOutgoingMessageWorker: %w", err)
			continue
		}

		// TODO: replace this client
		resp, err := http.Post(fmt.Sprintf("%s/consume", s.SrvAddr), "application/json", bytes.NewBuffer(msg.Data))
		if err != nil {
			logger.Errorf("ProcessOutgoingMessageWorker: %w", err)
			continue
		}

		respbody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			s.Outgoing <- msg

			err := fmt.Errorf("failed to publish message to the subscriber %s %v", s.Name, string(respbody))
			logger.Errorf("ProcessOutgoingMessageWorker: %w", err)
		} else {
			logger.Infof("ProcessOutgoingMessageWorker: message sent to %s successfully", s.Name)
		}
		resp.Body.Close()
	}
}

func MessengerAddrFromEnv() string {
	messengerAddr := os.Getenv("MESSENGER_SRV_ADDR")
	if messengerAddr == "" {
		messengerAddr = "http://0.0.0.0:3000"
	}
	return messengerAddr
}
