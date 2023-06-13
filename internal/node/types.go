package node

import (
	"github.com/bloxapp/ssv-spec/dkg"
	"github.com/bloxapp/ssv-spec/types"
)

type Network interface {
	// StreamDKGBlame will stream to any subscriber the blame result of the DKG
	StreamDKGBlame(blame *dkg.BlameOutput) error
	// StreamDKGOutput will stream to any subscriber the result of the DKG
	StreamDKGOutput(output map[types.OperatorID]*dkg.SignedOutput) error
	// BroadcastDKGMessage will broadcast a msg to the dkg network
	BroadcastDKGMessage(msg *SignedTransport) error
}
