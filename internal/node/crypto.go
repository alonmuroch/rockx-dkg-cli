package node

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	ssz "github.com/ferranbt/fastssz"
)

func SignRSA(sk *rsa.PrivateKey, msg ssz.Marshaler) ([]byte, error) {
	byts, err := msg.MarshalSSZ()
	if err != nil {
		return nil, err
	}

	r := sha256.Sum256(byts)
	return sk.Sign(rand.Reader, r[:], &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthAuto,
		Hash:       crypto.SHA256,
	})
}

func VerifyRSA(pk *rsa.PublicKey, msg ssz.Marshaler, signature []byte) error {
	byts, err := msg.MarshalSSZ()
	if err != nil {
		return err
	}

	r := sha256.Sum256(byts)
	return rsa.VerifyPSS(pk, crypto.SHA256, r[:], signature, nil)
}
