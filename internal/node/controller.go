package node

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/RockX-SG/frost-dkg-demo/internal/logger"
	"github.com/bloxapp/ssv-spec/dkg"
	"github.com/bloxapp/ssv-spec/types"
)

type Config struct {
	SSVOperator *dkg.Operator
	Storage     dkg.Storage

	Logger *logger.Logger
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
			InitMsg:   initMsg,
			Operators: operators,

			config: c.config,
		}
		c.Instances[hex.EncodeToString(msg.Message.Identifier[:])] = i
		i.Start()

		c.config.Logger.Printf("Started DKG instanc with ID: %x", msg.Message.Identifier)
	default:
		return errors.New(fmt.Sprintf("unknown message type: %d", msg.Message.Type))
	}
	return nil
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
