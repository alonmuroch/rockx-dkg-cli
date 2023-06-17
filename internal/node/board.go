package node

import (
	"github.com/RockX-SG/frost-dkg-demo/internal/node/kyber"
	"github.com/drand/kyber/share/dkg"
	"github.com/sirupsen/logrus"
)

type Board struct {
	broadcastF func(msg *KyberMessage) error
	logger     *logrus.Entry

	DealC          chan dkg.DealBundle
	ResponseC      chan dkg.ResponseBundle
	JustificationC chan dkg.JustificationBundle
}

func NewBoard(
	logger *logrus.Entry,
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
	b.logger.Infof("pushing deal bundle")

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
	b.logger.Infof("pushing response bundle")

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
	b.logger.Infof("pushing justification bundle")

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
