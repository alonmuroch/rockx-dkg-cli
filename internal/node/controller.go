package node

import (
	"crypto/rsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/bloxapp/ssv-spec/dkg"
	"github.com/bloxapp/ssv-spec/types"
	"github.com/drand/kyber"
	"github.com/drand/kyber/pairing"
	dkg2 "github.com/drand/kyber/share/dkg"
	"github.com/sirupsen/logrus"
	"time"
)

// OperatorOwner represents the owner of an operator
type OperatorOwner struct {
	Operator     *dkg.Operator
	EncryptionSK *rsa.PrivateKey
}

type Config struct {
	SSVOperator *OperatorOwner
	Storage     dkg.Storage
	Network     Network

	PairingSuite pairing.Suite

	Logger *logrus.Entry
}

func (config *Config) GetG1Suite() dkg2.Suite {
	return config.PairingSuite.G1().(dkg2.Suite)
}

func (config *Config) GetScalar() kyber.Scalar {
	return config.PairingSuite.G1().Scalar()
}

func (config *Config) GetPoint() kyber.Point {
	return config.PairingSuite.G1().Point()
}

type Controller struct {
	// Instances maps identifier (in string) to instance
	Instances map[string]*Instance
	config    *Config
}

func NewController(c *Config) *Controller {
	return &Controller{
		Instances: map[string]*Instance{},
		config:    c,
	}
}

func (c *Controller) Process(msg *SignedTransport) error {
	switch msg.Message.Type {
	case InitMessageType:
		initMsg := &Init{}
		if err := initMsg.UnmarshalSSZ(msg.Message.Data); err != nil {
			return err
		}
		operators, err := c.getOperators(initMsg.Operators)
		if err != nil {
			return err
		}

		i := &Instance{
			InitMsg:    initMsg,
			Identifier: msg.Message.Identifier,
			Operators:  operators,

			exchangeMessages: map[uint64]*Exchange{},
			outputMessages:   map[uint64]*Output{},

			config: c.config,
		}
		c.Instances[c.IdString(msg.Message.Identifier)] = i
		go func() {
			// sleep to let init message propagate
			time.Sleep(time.Second)
			i.Start()
		}()

		c.config.Logger.Printf("Started DKG instanc with ID: %x", msg.Message.Identifier)
	default:
		i := c.getInstance(msg.Message.Identifier)
		if i == nil {
			return errors.New(fmt.Sprintf("instance not found for id: %x", msg.Message.Identifier))
		}

		return i.Process(msg)
	}
	return nil
}

func (c *Controller) IdString(id [24]byte) string {
	return hex.EncodeToString(id[:])
}

func (c *Controller) getInstance(id [24]byte) *Instance {
	return c.Instances[c.IdString(id)]
}

func (c *Controller) getOperators(operators []uint64) (map[uint64]*dkg.Operator, error) {
	ret := make(map[uint64]*dkg.Operator, 0)
	for _, id := range operators {
		found, operator, err := c.config.Storage.GetDKGOperator(types.OperatorID(id))
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.New("operator not found")
		}
		ret[id] = operator
	}
	return ret, nil
}
