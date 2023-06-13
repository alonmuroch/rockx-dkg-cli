package node

import (
	"github.com/RockX-SG/frost-dkg-demo/internal/logger"
	"github.com/RockX-SG/frost-dkg-demo/internal/node/kyber"
	"github.com/drand/kyber/share/dkg"
)

type Board struct {
	broadcastF func(msg *KyberMessage) error
	logger     *logger.Logger

	DealC          chan dkg.DealBundle
	ResponseC      chan dkg.ResponseBundle
	JustificationC chan dkg.JustificationBundle
}

func NewBoard(
	logger *logger.Logger,
	broadcastF func(msg *KyberMessage) error,
) *Board {
	return &Board{
		broadcastF: broadcastF,
		logger:     logger,

		DealC:          make(chan dkg.DealBundle),
		ResponseC:      make(chan dkg.ResponseBundle),
		JustificationC: make(chan dkg.JustificationBundle),
	}
}

func (b *Board) PushDeals(bundle *dkg.DealBundle) {
	byts, err := kyber.EncodeDealBundle(bundle)
	if err != nil {
		b.logger.Error(err.Error())
		return
	}

	msg := &KyberMessage{
		Type: KyberDealBundleMessageType,
		Data: byts,
	}

	if err := b.broadcastF(msg); err != nil {
		b.logger.Error(err.Error())
		return
	}
}

func (b *Board) IncomingDeal() <-chan dkg.DealBundle {
	return b.DealC
}

func (b *Board) PushResponses(bundle *dkg.ResponseBundle) {
	byts, err := kyber.EncodeResponseBundle(bundle)
	if err != nil {
		b.logger.Error(err.Error())
		return
	}

	msg := &KyberMessage{
		Type: KyberResponseBundleMessageType,
		Data: byts,
	}

	if err := b.broadcastF(msg); err != nil {
		b.logger.Error(err.Error())
		return
	}
}

func (b *Board) IncomingResponse() <-chan dkg.ResponseBundle {
	return b.ResponseC
}

func (b *Board) PushJustifications(bundle *dkg.JustificationBundle) {
	byts, err := kyber.EncodeJustificationBundle(bundle)
	if err != nil {
		b.logger.Error(err.Error())
		return
	}

	msg := &KyberMessage{
		Type: KyberJustificationBundleMessageType,
		Data: byts,
	}

	if err := b.broadcastF(msg); err != nil {
		b.logger.Error(err.Error())
		return
	}
}

func (b *Board) IncomingJustification() <-chan dkg.JustificationBundle {
	return b.JustificationC
}
