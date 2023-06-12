package node

import "github.com/bloxapp/ssv-spec/dkg"

type Config struct {
	SSVOperator *dkg.Operator
}

type Controller struct {
	config *Config
}

func NewController(c *Config) *Controller {
	return &Controller{config: c}
}

func (c *Controller) Process(msg *SignedTransport) error {
	return nil
}
