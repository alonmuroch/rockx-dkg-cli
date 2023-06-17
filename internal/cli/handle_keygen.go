package cli

import (
	"encoding/hex"
	"fmt"
	"github.com/RockX-SG/frost-dkg-demo/internal/node"
	"github.com/pkg/errors"
	"strconv"
	"strings"

	"github.com/RockX-SG/frost-dkg-demo/internal/messenger"
	"github.com/bloxapp/ssv-spec/types"
	"github.com/urfave/cli/v2"
)

func (h *CliHandler) HandleKeygen(c *cli.Context) error {
	keygenRequest := &KeygenRequest{}
	if err := keygenRequest.parseKeygenRequest(c); err != nil {
		return fmt.Errorf("HandleKeygen: failed to parse keygen request: %w", err)
	}

	requestID := getRandRequestID()
	requestIDInHex := hex.EncodeToString(requestID[:])

	fmt.Println("operators", keygenRequest.allOperators())
	messengerClient := messenger.NewMessengerClient(messenger.MessengerAddrFromEnv())
	if err := messengerClient.CreateTopic(requestIDInHex, keygenRequest.allOperators()); err != nil {
		return fmt.Errorf("HandleKeygen: failed to create a new topic on messenger service: %w", err)
	}

	initMsg, err := keygenRequest.initMsgForKeygen(requestID)
	if err != nil {
		return fmt.Errorf("HandleKeygen: failed to generate init message for keygen: %w", err)
	}

	fmt.Printf("Sending keygen init request for session ID: %s\n", requestIDInHex)
	return messengerClient.BroadcastDKGMessage(initMsg)

	//for operatorID, nodeAddr := range keygenRequest.Operators {
	//	if err := h.sendInitMsg(operatorID, nodeAddr, initMsgBytes); err != nil {
	//		return fmt.Errorf("HandleKeygen: failed to send init message to operatorID %d: %w", operatorID, err)
	//	}
	//}
	//
	//fmt.Printf("keygen init request sent with ID: %s\n", requestIDInHex)
	//return nil
}

//func (h *CliHandler) sendInitMsg(operatorID uint64, addr string, data []byte) error {
//	url := fmt.Sprintf("%s/consume", addr)
//	resp, err := h.client.Post(url, "application/json", bytes.NewBuffer(data))
//	if err != nil {
//		return err
//	}
//	defer resp.Body.Close()
//	if resp.StatusCode != http.StatusOK {
//		return fmt.Errorf("request to operator %d to consume init message failed with status %s", operatorID, resp.Status)
//	}
//	return nil
//}

type KeygenRequest struct {
	Operators            map[uint64]string `json:"operators"`
	Threshold            uint64            `json:"threshold"`
	WithdrawalCredential string            `json:"withdrawal_credentials"`
	ForkVersion          string            `json:"fork_version"`
}

func (request *KeygenRequest) allOperators() []uint64 {
	operators := []uint64{}
	for operatorID := range request.Operators {
		operators = append(operators, operatorID)
	}
	return operators
}

func (request *KeygenRequest) parseKeygenRequest(c *cli.Context) error {
	request.Operators = make(map[uint64]string)
	request.Threshold = c.Uint64("threshold")
	request.WithdrawalCredential = c.String("withdrawal-credentials")
	request.ForkVersion = c.String("fork-version")

	operatorkv := c.StringSlice("operator")
	for _, op := range operatorkv {
		op = strings.Trim(op, " ")
		pair := strings.Split(op, "=")
		if len(pair) != 2 {
			return fmt.Errorf("operator %s is not in the form of key=value", op)
		}
		opID, err := strconv.Atoi(pair[0])
		if err != nil {
			return err
		}
		request.Operators[uint64(opID)] = pair[1]
	}
	return nil
}

func (request *KeygenRequest) initMsgForKeygen(requestID [24]byte) (*node.SignedTransport, error) {
	withdrawalCred, _ := hex.DecodeString(request.WithdrawalCredential)
	forkVersion := types.NetworkFromString(request.ForkVersion).ForkVersion()

	//init := testingutils.InitMessageData(
	//	request.allOperators(),
	//	uint16(request.Threshold),
	//	withdrawalCred,
	//	forkVersion,
	//)
	//initBytes, _ := init.Encode()

	init := &node.Init{
		Operators:             request.allOperators(),
		T:                     request.Threshold,
		WithdrawalCredentials: withdrawalCred,
		Fork:                  forkVersion,
	}
	byts, err := init.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrap(err, "could not encode init msg")
	}

	return &node.SignedTransport{
		Message: &node.Transport{
			Type:       node.InitMessageType,
			Identifier: requestID,
			Data:       byts,
		},
	}, nil
	//signedByts, err := signedInit.MarshalSSZ()
	//if err != nil {
	//	return nil, errors.Wrap(err, "could not encode signed init msg")
	//}

	// TODO: TBD who signs this init msg
	//ks := testingutils.TestingKeygenKeySet()
	//signedInitMsg := testingutils.SignDKGMsg(ks.DKGOperators[1].SK, 1, &dkg.Message{
	//	MsgType:    dkg.InitMsgType,
	//	Identifier: requestID,
	//	Data:       byts,
	//})
	//signedInitMsgBytes, _ := signedInitMsg.Encode()

	//msg := &types.SSVMessage{
	//	MsgType: types.DKGMsgType,
	//	Data:    signedByts,
	//}
	//return signedInit.MarshalSSZ()
}
