/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracker/metrics"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type acceptView struct {
	tx *Transaction
}

func NewAcceptView(tx *Transaction) *acceptView {
	return &acceptView{tx: tx}
}

func (s *acceptView) Call(context view.Context) (interface{}, error) {
	agent := metrics.Get(context)
	agent.EmitKey(0, "ttx", "start", "acceptView", s.tx.ID())
	defer agent.EmitKey(0, "ttx", "end", "acceptView", s.tx.ID())

	var err error
	rawRequest, err := s.tx.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal token request")
	}
	// Check the envelope exists
	env := s.tx.Payload.Envelope
	if env == nil {
		return nil, errors.Errorf("expected fabric envelope")
	}

	// Store transient
	agent.EmitKey(0, "ttx", "start", "acceptViewStoreTransient", s.tx.ID())
	err = s.tx.storeTransient()
	if err != nil {
		return nil, errors.Wrapf(err, "failed storing transient")
	}
	agent.EmitKey(0, "ttx", "end", "acceptViewStoreTransient", s.tx.ID())

	// Store envelope
	if err := StoreEnvelope(context, s.tx); err != nil {
		return nil, errors.Wrapf(err, "failed storing envelope %s", s.tx.ID())
	}

	// Store transaction in the token transaction database
	if err := StoreTransactionRecords(context, s.tx); err != nil {
		return nil, errors.Wrapf(err, "failed storing transaction records %s", s.tx.ID())
	}

	// Send back an acknowledgement
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("send back ack")
	}

	logger.Debugf("signing ack response: %s", hash.Hashable(rawRequest))
	signer, err := view2.GetSigService(context).GetSigner(view2.GetIdentityProvider(context).DefaultIdentity())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get signer for default identity")
	}
	sigma, err := signer.Sign(rawRequest)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to sign ack response")
	}

	// Ack for distribution
	// Send the signature back
	session := context.Session()
	logger.Debugf("ack response: [%s] from [%s]", hash.Hashable(sigma), view2.GetIdentityProvider(context).DefaultIdentity())
	if err := session.Send(sigma); err != nil {
		return nil, errors.WithMessage(err, "failed sending ack")
	}

	agent.EmitKey(0, "ttx", "sent", "txAck", s.tx.ID())

	return s.tx, nil
}
