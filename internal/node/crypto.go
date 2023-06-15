package node

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"github.com/drand/kyber"
	"github.com/drand/kyber/share"
	"github.com/drand/kyber/share/dkg"
	ssz "github.com/ferranbt/fastssz"
	"github.com/herumi/bls-eth-go-binary/bls"
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

// Encrypt with secret key (base64) the bytes, return the encrypted key string
func Encrypt(pk *rsa.PublicKey, plainText []byte) ([]byte, error) {
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, pk, plainText)
	if err != nil {
		return nil, err
	}
	return encrypted, nil
}

func VerifyRSA(pk *rsa.PublicKey, msg ssz.Marshaler, signature []byte) error {
	byts, err := msg.MarshalSSZ()
	if err != nil {
		return err
	}

	r := sha256.Sum256(byts)
	return rsa.VerifyPSS(pk, crypto.SHA256, r[:], signature, nil)
}

func ResultToShareSecretKey(result *dkg.Result) (*bls.SecretKey, error) {
	share := result.Key.PriShare()
	bytsSk, err := share.V.MarshalBinary()
	if err != nil {
		return nil, err
	}
	sk := &bls.SecretKey{}
	if err := sk.Deserialize(bytsSk); err != nil {
		return nil, err
	}
	return sk, nil
}

func ResultsToValidatorPK(commitments []kyber.Point, suite dkg.Suite) (*bls.PublicKey, error) {
	exp := share.NewPubPoly(suite, suite.Point().Base(), commitments)
	bytsPK, err := exp.Eval(0).V.MarshalBinary()
	if err != nil {
		return nil, err
	}
	pk := &bls.PublicKey{}
	if err := pk.Deserialize(bytsPK); err != nil {
		return nil, err
	}
	return pk, nil
}
