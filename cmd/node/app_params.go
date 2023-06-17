package main

import (
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/bloxapp/ssv-spec/types"
)

type AppParams struct {
	OperatorID       types.OperatorID
	HttpAddress      string
	KeystoreFilePath string
	keystorePassword string
}

func (params *AppParams) loadFromEnv() {
	params.loadOperatorID()
	params.loadHttpAddress()
	params.loadKeystoreFilePath()
	params.loadKeystorePassword()
}

func (params *AppParams) print() string {
	return fmt.Sprintf(
		"operatorID=%d http_addr=%s keystore_filepath=%s",
		params.OperatorID,
		params.HttpAddress,
		params.KeystoreFilePath,
	)
}

func (params *AppParams) loadOperatorID() {
	operatorID, err := strconv.ParseUint(os.Getenv("NODE_OPERATOR_ID"), 10, 32)
	if err != nil {
		panic(err)
	}
	params.OperatorID = types.OperatorID(operatorID)
}

func (params *AppParams) loadHttpAddress() {
	nodeAddr := os.Getenv("NODE_ADDR")
	if nodeAddr == "" {
		nodeAddr = "0.0.0.0:8080"
	}
	params.HttpAddress = nodeAddr
}

func (params *AppParams) loadKeystoreFilePath() {
	keystoreFilePath := os.Getenv("KEYSTORE_FILE_PATH")
	if keystoreFilePath == "" {
		keystoreFilePath = "keystore.json"
	}
	params.KeystoreFilePath = keystoreFilePath
}

func (params *AppParams) loadKeystorePassword() {
	params.keystorePassword = os.Getenv("KEYSTORE_PASSWORD")
}

func (params *AppParams) loadDecryptedPrivateKey() (*rsa.PrivateKey, error) {
	keyByts, err := ioutil.ReadFile(params.KeystoreFilePath)
	if err != nil {
		return nil, err
	}
	byts, err := base64.StdEncoding.DecodeString(string(keyByts))

	//key, err := keystore.DecryptKey(keyByts, params.keystorePassword)
	//if err != nil {
	//	return nil, err
	//}

	return types.PemToPrivateKey(byts)
}
