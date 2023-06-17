package cli

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/RockX-SG/frost-dkg-demo/internal/node"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/RockX-SG/frost-dkg-demo/internal/messenger"
	"github.com/urfave/cli/v2"
)

func (h *CliHandler) HandleResharing(c *cli.Context) error {
	resharingRequest := &ResharingRequest{}
	if err := resharingRequest.parseResharingRequest(c); err != nil {
		return fmt.Errorf("HandleResharing: failed to parse resharing request: %w", err)
	}

	requestID := getRandRequestID()
	requestIDInHex := hex.EncodeToString(requestID[:])

	operators := resharingRequest.newOperators()
	operatorsOld := resharingRequest.oldOperators()
	alloperators := append(operators, operatorsOld...)

	messengerClient := messenger.NewMessengerClient(messenger.MessengerAddrFromEnv())
	if err := messengerClient.CreateTopic(requestIDInHex, alloperators); err != nil {
		return fmt.Errorf("HandleResharing: failed to createa new topic on messenger service: %w", err)
	}

	initMsgBytes, err := resharingRequest.initMsgForResharing(requestID)
	if err != nil {
		return fmt.Errorf("HandleResharing: failed to generate init message for keygen: %w", err)
	}

	for _, operatorID := range alloperators {
		addr := resharingRequest.nodeAddress(operatorID)
		if err := h.sendReshareMsg(operatorID, addr, initMsgBytes); err != nil {
			return err
		}
	}

	fmt.Printf("resharing init request sent with ID: %s\n", requestIDInHex)
	return nil
}

func (h *CliHandler) sendReshareMsg(operatorID uint64, addr string, data []byte) error {
	url := fmt.Sprintf("%s/consume", addr)
	resp, err := h.client.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send reshare message with code %d to operator %d", resp.StatusCode, operatorID)
	}
	return nil
}

type ResharingRequest struct {
	Operators    map[uint64]string `json:"operators"`
	Threshold    int               `json:"threshold"`
	ValidatorPK  string            `json:"validator_pk"`
	OperatorsOld map[uint64]string `json:"operators_old"`
}

func (request *ResharingRequest) parseResharingRequest(c *cli.Context) error {
	request.Operators = make(map[uint64]string)
	request.OperatorsOld = make(map[uint64]string)
	request.Threshold = c.Int("threshold")
	request.ValidatorPK = c.String("validator-pk")

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

	oldoperatorkv := c.StringSlice("old-operator")
	for _, op := range oldoperatorkv {
		op = strings.Trim(op, " ")
		pair := strings.Split(op, "=")
		if len(pair) != 2 {
			return fmt.Errorf("operator %s is not in the form of key=value", op)
		}
		opID, err := strconv.Atoi(pair[0])
		if err != nil {
			return err
		}
		request.OperatorsOld[uint64(opID)] = pair[1]
	}
	return nil
}

func (request *ResharingRequest) nodeAddress(operatorID uint64) string {
	var nodeAddr string
	_, ok := request.Operators[operatorID]
	if ok {
		nodeAddr = request.Operators[operatorID]
	} else {
		nodeAddr = request.OperatorsOld[operatorID]
	}
	return nodeAddr
}

func (request *ResharingRequest) newOperators() []uint64 {
	operators := []uint64{}
	for operatorID := range request.Operators {
		operators = append(operators, operatorID)
	}
	return operators
}
func (request *ResharingRequest) oldOperators() []uint64 {
	operatorsOld := []uint64{}
	for operatorID := range request.OperatorsOld {
		operatorsOld = append(operatorsOld, operatorID)
	}
	return operatorsOld
}

func (request *ResharingRequest) initMsgForResharing(requestID [24]byte) ([]byte, error) {
	//vk, err := hex.DecodeString(request.ValidatorPK)
	//if err != nil {
	//	return nil, err
	//}

	init := &node.Init{
		Operators: request.newOperators(),
		T:         uint64(request.Threshold),
	}
	byts, err := init.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrap(err, "could not encode init msg")
	}

	signedInit := node.SignedTransport{
		Message: &node.Transport{
			Type:       node.InitReshareMessageType,
			Identifier: requestID,
			Data:       byts,
		},
	}
	return signedInit.MarshalSSZ()

	//reshare := testingutils.ReshareMessageData(
	//	request.newOperators(),
	//	uint16(request.Threshold),
	//	vk,
	//	request.oldOperators(),
	//)
	//reshareBytes, _ := reshare.Encode()
	//
	//// TODO: TBD who signs this init msg
	//ks := testingutils.TestingResharingKeySet()
	//reshareMsg := testingutils.SignDKGMsg(ks.DKGOperators[5].SK, 5, &dkg.Message{
	//	MsgType:    dkg.ReshareMsgType,
	//	Identifier: requestID,
	//	Data:       reshareBytes,
	//})
	//byts, _ := reshareMsg.Encode()
	//
	//msg := &types.SSVMessage{
	//	MsgType: types.DKGMsgType,
	//	Data:    byts,
	//}
	//return msg.Encode()
}
