/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/id"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/sig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
)

// AcceptView is used to accept tokens without the need to generate any signature.
// This is a view executed by a responder.
type AcceptView struct {
	tx      *Transaction
	options *EndorsementsOpts
}

// NewAcceptView returns a new instance of AcceptView given in input a transaction.
// Usually, AcceptView is preceded by an invocation of `tx.ReceiveTransaction(context)`
// necessary if the initiator have invoked the CollectEndorsementsView
func NewAcceptView(tx *Transaction, opts ...EndorsementsOpt) *AcceptView {
	options, err := CompileCollectEndorsementsOpts(opts...)
	if err != nil {
		panic(err)
	}
	return &AcceptView{tx: tx, options: options}
}

// Call accepts the tokens created by the transaction this view has been created with.
func (s *AcceptView) Call(context view.Context) (interface{}, error) {
	// Store transaction in the token transaction database
	if err := StoreTransactionRecords(context, s.tx); err != nil {
		return nil, errors.Wrapf(err, "failed storing transaction records %s", s.tx.ID())
	}

	// ack
	if err := s.ack(context); err != nil {
		return nil, errors.Wrapf(err, "failed acknowledging transaction %s", s.tx.ID())
	}

	// cache the token request into the tokens db
	t, err := tokens.GetService(context, s.tx.TMSID())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tokens db for [%s]", s.tx.TMSID())
	}

	if err := t.CacheRequest(context.Context(), s.tx.TMSID(), s.tx.TokenRequest); err != nil {
		logger.WarnfContext(context.Context(), "failed to cache token request [%s], this might cause delay, investigate when possible: [%s]", s.tx.TokenRequest.Anchor, err)
	}

	// metrics
	labels := []string{
		"network", s.tx.Network(),
		"channel", s.tx.Channel(),
		"namespace", s.tx.Namespace(),
	}
	GetMetrics(context).AcceptedTransactions.With(labels...).Add(1)

	return s.tx, nil
}

func (s *AcceptView) ack(context view.Context) error {
	// Send back an acknowledgement
	txRaw := s.tx.FromRaw
	idProvider, err := id.GetProvider(context)
	if err != nil {
		return errors.Wrapf(err, "failed to get identity provider")
	}
	defaultIdentity := idProvider.DefaultIdentity()

	logger.DebugfContext(context.Context(), "signing ack response [%s] with identity [%s]", utils.Hashable(txRaw), defaultIdentity)
	sigService, err := sig.GetService(context)
	if err != nil {
		return errors.Wrapf(err, "failed to get sig service")
	}
	signer, err := sigService.GetSigner(defaultIdentity)
	if err != nil {
		return errors.WithMessagef(err, "failed to get signer for default identity")
	}
	logger.DebugfContext(context.Context(), "Sign ack for distribution")
	sigma, err := signer.Sign(txRaw)
	if err != nil {
		return errors.WithMessagef(err, "failed to sign ack response")
	}

	// Ack for distribution
	// Send the signature back
	session := context.Session()
	logger.DebugfContext(context.Context(), "ack response: [%s] from [%s]", utils.Hashable(sigma), defaultIdentity)
	if err := session.SendWithContext(context.Context(), sigma); err != nil {
		return errors.WithMessagef(err, "failed sending ack")
	}
	return nil
}
