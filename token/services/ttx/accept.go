/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type acceptView struct {
	tx      *Transaction
	options *EndorsementsOpts
}

func NewAcceptView(tx *Transaction, opts ...EndorsementsOpt) *acceptView {
	options, err := CompileCollectEndorsementsOpts(opts...)
	if err != nil {
		panic(err)
	}
	return &acceptView{tx: tx, options: options}
}

func (s *acceptView) Call(context view.Context) (interface{}, error) {
	if err := s.respondToSignatureRequests(context); err != nil {
		return nil, err
	}

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
	err = s.tx.storeTransient()
	if err != nil {
		return nil, errors.Wrapf(err, "failed storing transient")
	}

	// Store envelope
	if !s.options.SkipApproval {
		if err := StoreEnvelope(context, s.tx); err != nil {
			return nil, errors.Wrapf(err, "failed storing envelope %s", s.tx.ID())
		}
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

	labels := []string{
		"network", s.tx.Network(),
		"channel", s.tx.Channel(),
		"namespace", s.tx.Namespace(),
	}
	GetMetrics(context).AcceptedTransactions.With(labels...).Add(1)

	return s.tx, nil
}

func (s *acceptView) respondToSignatureRequests(context view.Context) error {
	requestsToBeSigned, err := requestsToBeSigned(s.tx.TokenRequest)
	if err != nil {
		return errors.Wrapf(err, "failed collecting requests of signature")
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("respond to signature requests [%s][%d]", s.tx.ID(), len(requestsToBeSigned))
	}

	session := context.Session()
	for i := 0; i < len(requestsToBeSigned); i++ {
		signatureRequest := &SignatureRequest{}

		if i == 0 {
			k, err := keys.CreateCompositeKey("signatureRequest", []string{s.tx.ID()})
			if err != nil {
				return errors.Wrap(err, "failed to generate key to store signature request")
			}
			var srRaw []byte
			if err := kvs.GetService(context).Get(k, &srRaw); err != nil {
				return errors.Wrap(err, "failed to to store signature request")
			}
			if err := Unmarshal(srRaw, signatureRequest); err != nil {
				return errors.Wrap(err, "failed unmarshalling signature request")
			}
		} else {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("Receiving signature request...")
			}

			timeout := time.NewTimer(time.Minute)

			sessionChannel := session.Receive()
			var msg *view.Message
			select {
			case msg = <-sessionChannel:
				logger.Debug("message received from %s", session.Info().Caller)
				timeout.Stop()
			case <-timeout.C:
				timeout.Stop()
				return errors.Errorf("Timeout from party %s", session.Info().Caller)
			}
			if msg.Status == view.ERROR {
				return errors.New(string(msg.Payload))
			}

			// TODO: check what is signed...
			err := Unmarshal(msg.Payload, signatureRequest)
			if err != nil {
				return errors.Wrap(err, "failed unmarshalling signature request")
			}
		}
		tms := token.GetManagementService(context, token.WithTMS(s.tx.Network(), s.tx.Channel(), s.tx.Namespace()))
		if tms == nil {
			return errors.Errorf("failed getting TMS for [%s:%s:%s]", s.tx.Network(), s.tx.Channel(), s.tx.Namespace())
		}

		if !tms.WalletManager().IsMe(signatureRequest.Signer) {
			return errors.Errorf("identity [%s] is not me", signatureRequest.Signer.UniqueID())
		}
		signer, err := s.tx.TokenService().SigService().GetSigner(signatureRequest.Signer)
		if err != nil {
			return errors.Wrapf(err, "cannot find signer for [%s]", signatureRequest.Signer.UniqueID())
		}
		sigma, err := signer.Sign(signatureRequest.MessageToSign())
		if err != nil {
			return errors.Wrapf(err, "failed signing request")
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("Send back signature...")
		}
		err = session.Send(sigma)
		if err != nil {
			return errors.Wrapf(err, "failed sending signature back")
		}
	}

	if len(requestsToBeSigned) > 0 {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("wait the transaction to be sent back [%s]", s.tx.ID())
		}
		// expect again to receive a transaction
		tx, err := ReceiveTransaction(context)
		if err != nil {
			return errors.Wrapf(err, "expected to receive a transaction")
		}
		// TODO: check that the token requests match
		s.tx = tx
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("wait the transaction to be sent back [%s], received", s.tx.ID())
		}
	} else {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("no need to wait the transaction to be sent back [%s]", s.tx.ID())
		}
	}

	return nil
}
