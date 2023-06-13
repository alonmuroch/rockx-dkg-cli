package kyber

import (
	"crypto/rand"
	"github.com/drand/kyber"
	"github.com/drand/kyber/pairing"
	"github.com/drand/kyber/share/dkg"
	"github.com/drand/kyber/sign/bls"
	"time"
)

// NonceLength is the length of the nonce
const NonceLength = 32

type Config struct {
	// Secret session secret key
	Secret kyber.Scalar
	Nodes  []dkg.Node
	Suite  pairing.Suite
	T      int
	Board  dkg.Board
}

func NewDKGProtocol(config *Config) (*dkg.Protocol, error) {
	dkgConfig := &dkg.Config{
		Longterm:  config.Secret,
		Nonce:     GetNonce(),
		Suite:     config.Suite.G1().(dkg.Suite),
		NewNodes:  config.Nodes,
		Threshold: config.T,
		Auth:      bls.NewSchemeOnG2(config.Suite),
	}

	return dkg.NewProtocol(
		dkgConfig,
		config.Board,
		dkg.NewTimePhaser(time.Second*2),
		false,
	)
}

// GetNonce returns a suitable nonce to feed in the DKG config.
func GetNonce() []byte {
	var nonce [NonceLength]byte
	n, err := rand.Read(nonce[:])
	if n != NonceLength {
		panic("could not read enough random bytes for nonce")
	}
	if err != nil {
		panic(err)
	}
	return nonce[:]
}
