package cli

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/RockX-SG/frost-dkg-demo/internal/messenger"
	"github.com/bloxapp/ssv-spec/dkg"
	"github.com/bloxapp/ssv-spec/types"
	"github.com/bloxapp/ssv-spec/types/testingutils"
	"github.com/urfave/cli/v2"
)

func (h *CliHandler) HandleKeygen(c *cli.Context) error {
	keygenRequest, err := parseKeygenRequest(c)
	if err != nil {
		return err
	}

	requestID := getRandRequestID()
	requestIDInHex := hex.EncodeToString(requestID[:])

	messengerClient := messenger.NewMessengerClient(messenger.MessengerAddrFromEnv())
	if err := messengerClient.CreateTopic(requestIDInHex, keygenRequest.allOperators()); err != nil {
		return err
	}

	initMsgBytes, err := keygenRequest.initMsgForKeygen(requestID)
	if err != nil {
		return err
	}

	for operatorID, nodeAddr := range keygenRequest.Operators {
		if err := sendInitMsg(operatorID, nodeAddr, initMsgBytes); err != nil {
			return err
		}
	}

	fmt.Printf("keygen init request sent with ID: %s\n", requestIDInHex)
	return nil
}

func sendInitMsg(operatorID types.OperatorID, addr string, data []byte) error {
	url := fmt.Sprintf("%s/consume", addr)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send init message to the operator %d", operatorID)
	}
	return nil
}

func parseKeygenRequest(c *cli.Context) (*KeygenRequest, error) {
	keygenRequest := KeygenRequest{
		Operators:            make(map[types.OperatorID]string),
		Threshold:            c.Int("threshold"),
		WithdrawalCredential: c.String("withdrawal-credentials"),
		ForkVersion:          c.String("fork-version"),
	}
	operatorkv := c.StringSlice("operator")
	for _, op := range operatorkv {
		op = strings.Trim(op, " ")
		pair := strings.Split(op, "=")
		if len(pair) != 2 {
			return nil, fmt.Errorf("operator %s is not in the form of key=value", op)
		}
		opID, err := strconv.Atoi(pair[0])
		if err != nil {
			return nil, err
		}
		keygenRequest.Operators[types.OperatorID(opID)] = pair[1]
	}

	return &keygenRequest, nil
}

type KeygenRequest struct {
	Operators            map[types.OperatorID]string `json:"operators"`
	Threshold            int                         `json:"threshold"`
	WithdrawalCredential string                      `json:"withdrawal_credentials"`
	ForkVersion          string                      `json:"fork_version"`
}

func (request *KeygenRequest) allOperators() []types.OperatorID {
	operators := []types.OperatorID{}
	for operatorID, _ := range request.Operators {
		operators = append(operators, operatorID)
	}
	return operators
}

func (request *KeygenRequest) initMsgForKeygen(requestID dkg.RequestID) ([]byte, error) {
	withdrawalCred, _ := hex.DecodeString(request.WithdrawalCredential)
	forkVersion := types.NetworkFromString(request.ForkVersion).ForkVersion()

	init := testingutils.InitMessageData(
		request.allOperators(),
		uint16(request.Threshold),
		withdrawalCred,
		forkVersion,
	)
	initBytes, _ := init.Encode()

	// TODO: TBD who signs this init msg
	ks := testingutils.TestingKeygenKeySet()
	signedInitMsg := testingutils.SignDKGMsg(ks.DKGOperators[1].SK, 1, &dkg.Message{
		MsgType:    dkg.InitMsgType,
		Identifier: requestID,
		Data:       initBytes,
	})
	signedInitMsgBytes, _ := signedInitMsg.Encode()

	msg := &types.SSVMessage{
		MsgType: types.DKGMsgType,
		Data:    signedInitMsgBytes,
	}
	return msg.Encode()
}
