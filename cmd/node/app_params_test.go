package main

import (
	"encoding/base64"
	"fmt"
	"github.com/bloxapp/ssv-spec/types"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
)

func TestDecrypt(t *testing.T) {
	keyByts, err := ioutil.ReadFile("../../keys/538")
	require.NoError(t, err)
	fmt.Printf("%s\n", string(keyByts))
	byts, err := base64.StdEncoding.DecodeString(string(keyByts))
	require.NoError(t, err)

	_, err = types.PemToPrivateKey(byts)
	require.NoError(t, err)
}
