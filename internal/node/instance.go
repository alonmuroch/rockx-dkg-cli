package node

import (
	"errors"
	"github.com/bloxapp/ssv-spec/dkg"
	"github.com/drand/kyber"
	"github.com/drand/kyber/util/random"
)

type Instance struct {
	InitMsg    *Init
	Identifier [24]byte
	Operators  map[uint64]*dkg.Operator

	eciesSK          kyber.Scalar
	exchangeMessages map[uint64]*Exchange

	config *Config
}

func (i *Instance) Start() error {
	i.eciesSK = i.config.GetScalar().Pick(random.New())
	pk := i.config.GetPoint().Mul(i.eciesSK, nil)
	pkByts, err := pk.MarshalBinary()
	if err != nil {
		return err
	}

	exch := Exchange{
		PK: pkByts,
	}
	exchByts, err := exch.MarshalSSZ()
	if err != nil {
		return err
	}

	return i.Broadcast(&Transport{
		Type:       ExchangeMessageType,
		Identifier: i.Identifier,
		Data:       exchByts,
	})
}

func (i *Instance) Process(msg *SignedTransport) error {
	if err := i.validateTransportMessage(msg); err != nil {
		return err
	}

	switch msg.Message.Type {
	case ExchangeMessageType:
		initMsg := &Exchange{}
		if err := initMsg.UnmarshalSSZ(msg.Message.Data); err != nil {
			return err
		}

		if i.exchangeMessages[msg.Signer] != nil {
			return errors.New("duplicate exchange msg")
		}
		i.exchangeMessages[msg.Signer] = initMsg

		i.config.Logger.Infof("received exchange message from %d", msg.Signer)
		
		// all exchange messages received
		if len(i.exchangeMessages) == len(i.Operators) {
			i.config.Logger.Infof("All exchange messages received, starting DKG session")
			// start kyber instance
		}

		return nil
	case KyberMessageType:
		initMsg := &KyberMessage{}
		if err := initMsg.UnmarshalSSZ(msg.Message.Data); err != nil {
			return err
		}
		return nil
	default:
		return errors.New("unknown type")
	}
}

func (i *Instance) validateTransportMessage(msg *SignedTransport) error {
	if operator, ok := i.Operators[msg.Signer]; ok {
		return VerifyRSA(operator.EncryptionPubKey, msg.Message, msg.Signature)
	} else {
		return errors.New("unknown signer")
	}
}

func (i *Instance) Broadcast(msg *Transport) error {
	sig, err := SignRSA(i.config.SSVOperator.EncryptionSK, msg)
	if err != nil {
		return err
	}

	signedMsg := &SignedTransport{
		Message:   msg,
		Signer:    uint64(i.config.SSVOperator.Operator.OperatorID),
		Signature: sig,
	}

	return i.config.Network.BroadcastDKGMessage(signedMsg)
}
