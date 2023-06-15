package node

import (
	"bytes"
	"errors"
	kyber2 "github.com/RockX-SG/frost-dkg-demo/internal/node/kyber"
	"github.com/bloxapp/ssv-spec/dkg"
	"github.com/bloxapp/ssv-spec/types"
	"github.com/drand/kyber"
	dkg2 "github.com/drand/kyber/share/dkg"
	"github.com/drand/kyber/util/random"
	"github.com/herumi/bls-eth-go-binary/bls"
)

type Instance struct {
	InitMsg    *Init
	Identifier [24]byte
	Operators  map[uint64]*dkg.Operator

	eciesSK          kyber.Scalar
	exchangeMessages map[uint64]*Exchange
	outputMessages   map[uint64]*Output

	dkgProtocol *dkg2.Protocol
	board       *Board

	config *Config

	Share    *bls.SecretKey
	Finished bool
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
		return i.processExchangeMsg(msg)
	case KyberMessageType:
		return i.processKyberMsg(msg)
	case OutputMessageType:
		return i.processOutputMsg(msg)
	default:
		return errors.New("unknown type")
	}
}

func (i *Instance) processOutputMsg(msg *SignedTransport) error {
	exchMsg := &Output{}
	if err := exchMsg.UnmarshalSSZ(msg.Message.Data); err != nil {
		return err
	}

	if i.outputMessages[msg.Signer] != nil {
		return errors.New("duplicate output msg")
	}
	i.outputMessages[msg.Signer] = exchMsg

	i.config.Logger.Infof("received output msg")

	// all exchange messages received
	if len(i.outputMessages) == len(i.Operators) {
		i.config.Logger.Infof("output msg quorum")

		// validate validator PK outputs
		validatorPK := i.outputMessages[uint64(i.config.SSVOperator.Operator.OperatorID)].ValidatorPK
		for _, m := range i.outputMessages {
			if !bytes.Equal(validatorPK, m.ValidatorPK) {
				return errors.New("out validator PK inconsistency")
			}
		}

		// get output
		o, err := i.generateKeyGenOutput()
		if err != nil {
			return err
		}

		// store
		if err := i.config.Storage.SaveKeyGenOutput(o); err != nil {
			i.config.Logger.Errorf("could not save output: %s", err.Error())
		}

		// mark finished
		i.config.Logger.Infof("FINISHED!")
		i.Finished = true
	}
	return nil
}

func (i *Instance) generateKeyGenOutput() (*dkg.KeyGenOutput, error) {
	pubKeys := make(map[types.OperatorID]*bls.PublicKey)
	for id, msg := range i.outputMessages {
		pk := &bls.PublicKey{}
		if err := pk.Deserialize(msg.SharePK); err != nil {
			return nil, err
		}
		pubKeys[types.OperatorID(id)] = pk
	}

	return &dkg.KeyGenOutput{
		Share:           i.Share,
		OperatorPubKeys: pubKeys,
		ValidatorPK:     i.outputMessages[uint64(i.config.SSVOperator.Operator.OperatorID)].ValidatorPK,
		Threshold:       i.InitMsg.T,
	}, nil
}

func (i *Instance) processKyberMsg(msg *SignedTransport) error {
	kyberMsg := &KyberMessage{}
	if err := kyberMsg.UnmarshalSSZ(msg.Message.Data); err != nil {
		return err
	}

	switch kyberMsg.Type {
	case KyberDealBundleMessageType:
		b, err := kyber2.DecodeDealBundle(kyberMsg.Data, i.config.GetG1Suite())
		if err != nil {
			return err
		}

		i.config.Logger.Infof("received deal bundle from %d", msg.Signer)

		i.board.DealC <- *b
	case KyberResponseBundleMessageType:
		b, err := kyber2.DecodeResponseBundle(kyberMsg.Data)
		if err != nil {
			return err
		}

		i.config.Logger.Infof("received response bundle from %d", msg.Signer)

		i.board.ResponseC <- *b
	case KyberJustificationBundleMessageType:
		b, err := kyber2.DecodeJustificationBundle(kyberMsg.Data, i.config.GetG1Suite())
		if err != nil {
			return err
		}

		i.config.Logger.Infof("received justification bundle from %d", msg.Signer)

		i.board.JustificationC <- *b
	default:
		return errors.New("unknown kyber message type")
	}
	return nil
}

func (i *Instance) processExchangeMsg(msg *SignedTransport) error {
	exchMsg := &Exchange{}
	if err := exchMsg.UnmarshalSSZ(msg.Message.Data); err != nil {
		return err
	}

	if i.exchangeMessages[msg.Signer] != nil {
		return errors.New("duplicate exchange msg")
	}
	i.exchangeMessages[msg.Signer] = exchMsg

	i.config.Logger.Infof("received exchange message from %d", msg.Signer)

	// all exchange messages received
	if len(i.exchangeMessages) == len(i.Operators) {
		// new Kyber board
		board := i.getKyberBoard()

		// generate nodes
		nodes := make([]dkg2.Node, 0)
		for id, e := range i.exchangeMessages {
			p := i.config.GetG1Suite().Point()
			if err := p.UnmarshalBinary(e.PK); err != nil {
				return err
			}

			nodes = append(nodes, dkg2.Node{
				Index:  dkg2.Index(id),
				Public: p,
			})
		}

		// New protocol
		p, err := kyber2.NewDKGProtocol(&kyber2.Config{
			Identifier: i.Identifier[:],
			Secret:     i.eciesSK,
			Nodes:      nodes,
			Suite:      i.config.PairingSuite,
			T:          int(i.InitMsg.T),
			Board:      board,

			Logger: i.config.Logger,
		})
		if err != nil {
			return err
		}
		i.dkgProtocol = p

		go func(p *dkg2.Protocol, postF func(res *dkg2.OptionResult)) {
			res := <-p.WaitEnd()
			postF(&res)

		}(i.dkgProtocol, i.postDKGSession)

		i.config.Logger.Infof("All exchange messages received, starting DKG session")
	}

	return nil
}

func (i *Instance) getKyberBoard() *Board {
	if i.board == nil {
		i.board = NewBoard(
			i.config.Logger,
			func(msg *KyberMessage) error {
				i.config.Logger.Infof("broadcasting kyber message")

				byts, err := msg.MarshalSSZ()
				if err != nil {
					return err
				}

				return i.Broadcast(&Transport{
					Type:       KyberMessageType,
					Identifier: i.Identifier,
					Data:       byts,
				})
			},
		)
	}
	return i.board
}

func (i *Instance) validateTransportMessage(msg *SignedTransport) error {
	if operator, ok := i.Operators[msg.Signer]; ok {
		return VerifyRSA(operator.EncryptionPubKey, msg.Message, msg.Signature)
	} else {
		return errors.New("unknown signer")
	}
}

func (i *Instance) postDKGSession(res *dkg2.OptionResult) {
	i.config.Logger.Infof("<<<< ---- Post DKG ---- >>>>")
	if res.Error != nil {
		i.config.Logger.Errorf("post DKG error: %s", res.Error.Error())
		return
	}

	shareRes := res.Result
	id := uint64(shareRes.Key.Share.I)

	// verify share ID
	if id != uint64(i.config.SSVOperator.Operator.OperatorID) {
		i.config.Logger.Errorf("dkg id result doesn't match instance ID")
		return
	}

	// extract share
	share, err := ResultToShareSecretKey(shareRes)
	if err != nil {
		i.config.Logger.Errorf("could not get share from result: %s", err.Error())
		return
	}
	i.Share = share

	// encrypt share
	encryptedShare, err := Encrypt(i.config.SSVOperator.Operator.EncryptionPubKey, share.Serialize())
	if err != nil {
		i.config.Logger.Errorf("could not encrypt share: %s", err.Error())
		return
	}

	// extract validator PK
	validatorPK, err := ResultsToValidatorPK(shareRes.Key.Commitments(), i.config.GetG1Suite())
	if err != nil {
		i.config.Logger.Errorf("could not get validator PK from result: %s", err.Error())
		return
	}

	// sign partial deposit data
	root, _, err := GenerateETHDepositData(
		validatorPK.Serialize(),
		i.InitMsg.WithdrawalCredentials,
		i.InitMsg.Fork,
	)
	partialDepositSig := share.SignByte(root)

	// prepare output
	output := Output{
		EncryptedShare:              encryptedShare,
		SharePK:                     share.GetPublicKey().Serialize(),
		ValidatorPK:                 validatorPK.Serialize(),
		DepositDataPartialSignature: partialDepositSig.Serialize(),
	}
	byts, err := output.MarshalSSZ()
	if err != nil {
		i.config.Logger.Errorf("could not marshal output struct: %s", err.Error())
		return
	}

	if err := i.Broadcast(&Transport{
		Type:       OutputMessageType,
		Identifier: i.Identifier,
		Data:       byts,
	}); err != nil {
		i.config.Logger.Errorf("could not broadcast output msg: %s", err.Error())
		return
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
