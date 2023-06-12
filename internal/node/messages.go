package node

import "encoding/binary"

func NewIdentifier(address []byte, nonce uint64) [24]byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(nonce))

	ret := [24]byte{}
	copy(ret[:], address[:20])
	copy(ret[20:], b[:])

	return ret
}

type TransportType uint64

const (
	InitMessageType TransportType = iota
	InitReshareMessageType
	ExchangeMessageType
	OutputMessageType
)

type Transport struct {
	Type       TransportType
	Identifier [24]byte `ssz-size:"24"`     // | -- 20 bytes address --- | --- 4 bytes nonce --- |
	Data       []byte   `ssz-max:"8388608"` // 2^23
}

type SignedTransport struct {
	Message   *Transport
	Signer    uint64
	Signature []byte `ssz-max:"2048"`
}

type Init struct {
	// Operators involved in the DKG
	Operators []uint64 `ssz-max:"13"`
	// T is the threshold for signing
	T uint64
	// WithdrawalCredentials for deposit data
	WithdrawalCredentials []byte `ssz-max:"256"` // 2^23
	// Fork ethereum fork for signing
	Fork [4]byte `ssz-size:"4"`
}

type Output struct {
	EncryptedShare              []byte `ssz-max:"2048"`
	SharePK                     []byte `ssz-max:"2048"`
	ValidatorPK                 []byte `ssz-size:"48"`
	DepositDataPartialSignature []byte `ssz-size:"96"`
}
