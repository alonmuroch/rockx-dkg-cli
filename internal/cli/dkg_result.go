package cli

import (
	"encoding/hex"
	"strconv"

	"github.com/RockX-SG/frost-dkg-demo/internal/messenger"
	"github.com/bloxapp/ssv-spec/dkg"
	"github.com/bloxapp/ssv-spec/types"
)

type DKGResult struct {
	Output map[types.OperatorID]SignedOutput `json:"output"`
	Blame  *dkg.BlameOutput                  `json:"blame"`
}

type Output struct {
	RequestID            string
	EncryptedShare       string
	SharePubKey          string
	ValidatorPubKey      string
	DepositDataSignature string
}

type SignedOutput struct {
	Data      Output
	Signer    string
	Signature string
}

func formatResults(data *messenger.DataStore) *DKGResult {
	if data.BlameOutput != nil {
		return formatBlameResults(data.BlameOutput)
	}

	output := make(map[types.OperatorID]SignedOutput)
	for operatorID, o := range data.SessionOutputs {
		getHex := hex.EncodeToString
		v := SignedOutput{
			Data: Output{
				RequestID:            getHex(data.SessionID[:]),
				EncryptedShare:       getHex(o.EncryptedShare),
				SharePubKey:          getHex(o.SharePK),
				ValidatorPubKey:      getHex(o.ValidatorPK),
				DepositDataSignature: getHex(o.DepositDataPartialSignature),
			},
			Signer: strconv.Itoa(int(operatorID)),
			//Signature: hex.EncodeToString(o.Signature),
		}
		output[types.OperatorID(operatorID)] = v
	}

	return &DKGResult{Output: output}
}

func formatBlameResults(blameOutput *dkg.BlameOutput) *DKGResult {
	return &DKGResult{Blame: blameOutput}
}
