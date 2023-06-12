package node

import "github.com/bloxapp/ssv-spec/dkg"

type Instance struct {
	InitMsg   *Init
	Operators map[uint64]*dkg.Operator

	config *Config
}

func (i *Instance) Start() {

}
