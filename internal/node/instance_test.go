package node

import (
	"github.com/RockX-SG/frost-dkg-demo/internal/logger"
	"github.com/bloxapp/ssv-spec/dkg"
	"github.com/bloxapp/ssv-spec/types"
	"github.com/drand/kyber"
	bls "github.com/drand/kyber-bls12381"
	"github.com/drand/kyber/util/random"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var testIdentifier = [24]byte{}

var network = NewTestNetwork()

func instanceConfig(owner *OperatorOwner) *Config {
	storage := &testStorage{}

	return &Config{
		SSVOperator:  owner,
		Storage:      storage,
		Network:      network,
		PairingSuite: bls.NewBLS12381Suite(),
		Logger:       logger.NewSimple().WithField("id", owner.Operator.OperatorID),
	}
}

func instanceForID(owner *OperatorOwner) *Instance {
	operators := []uint64{538, 539, 540, 541}
	config := instanceConfig(owner)
	return &Instance{
		InitMsg: &Init{
			Operators:             operators,
			T:                     3,
			WithdrawalCredentials: make([]byte, 30),
			Fork:                  [4]byte{},
		},
		Identifier: testIdentifier,
		Operators: map[uint64]*dkg.Operator{
			538: OperatorOwners[0].Operator,
			539: OperatorOwners[1].Operator,
			540: OperatorOwners[2].Operator,
			541: OperatorOwners[3].Operator,
		},

		exchangeMessages: map[uint64]*Exchange{},

		config: config,
	}
}

func signMsg(t *testing.T, id uint64, msg *Transport) *SignedTransport {
	for _, o := range OperatorOwners {
		if o.Operator.OperatorID == types.OperatorID(id) {
			sig, err := SignRSA(o.EncryptionSK, msg)
			require.NoError(t, err)

			return &SignedTransport{
				Message:   msg,
				Signer:    uint64(o.Operator.OperatorID),
				Signature: sig,
			}
		}
	}
	panic("operator not found")
}

func TestInstance_Start(t *testing.T) {
	operators := []uint64{538, 539, 540, 541}
	instances := []*Instance{
		instanceForID(OperatorOwners[0]),
		instanceForID(OperatorOwners[1]),
		instanceForID(OperatorOwners[2]),
		instanceForID(OperatorOwners[3]),
	}

	for _, i := range instances {
		i.config.Network.(*testNetwork).registerNode(uint64(i.config.SSVOperator.Operator.OperatorID), i.Process)
	}

	eciesSKs := make(map[uint64]kyber.Scalar)
	for ii, id := range operators {
		eciesSKs[id] = instances[ii].config.GetScalar().Pick(random.New())
		instances[ii].eciesSK = eciesSKs[id]

		pk := instances[ii].config.GetPoint().Mul(eciesSKs[id], nil)
		pkByts, err := pk.MarshalBinary()
		require.NoError(t, err)

		exch := Exchange{
			PK: pkByts,
		}
		exchByts, err := exch.MarshalSSZ()
		require.NoError(t, err)

		signedMsg := signMsg(t, id, &Transport{
			Type:       ExchangeMessageType,
			Identifier: testIdentifier,
			Data:       exchByts,
		})

		require.NoError(t, VerifyRSA(OperatorOwners[ii].Operator.EncryptionPubKey, signedMsg.Message, signedMsg.Signature))
		for _, inst := range instances {
			require.NoError(t, inst.Process(signedMsg))
		}
	}

	for {
		time.Sleep(time.Millisecond * 100)
		if instances[0].Finished {
			return
		}
	}
}
