/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"encoding/base64"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/id"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/sig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	session2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/json/session"
)

type AcceptView struct {
	tx      *Transaction
	options *EndorsementsOpts
}

func NewAcceptView(tx *Transaction, opts ...EndorsementsOpt) *AcceptView {
	options, err := CompileCollectEndorsementsOpts(opts...)
	if err != nil {
		panic(err)
	}
	return &AcceptView{tx: tx, options: options}
}

func (s *AcceptView) Call(context view.Context) (interface{}, error) {
	if err := s.respondToSignatureRequests(context); err != nil {
		return nil, err
	}

	// Store transaction in the token transaction database
	if err := StoreTransactionRecords(context, s.tx); err != nil {
		return nil, errors.Wrapf(err, "failed storing transaction records %s", s.tx.ID())
	}

	txRaw := s.tx.FromRaw
	// Send back an acknowledgement
	idProvider, err := id.GetProvider(context)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity provider")
	}
	defaultIdentity := idProvider.DefaultIdentity()

	logger.DebugfContext(context.Context(), "signing ack response [%s] with identity [%s]", utils.Hashable(txRaw), defaultIdentity)
	sigService, err := sig.GetService(context)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sig service")
	}
	signer, err := sigService.GetSigner(defaultIdentity)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get signer for default identity")
	}
	logger.DebugfContext(context.Context(), "Sign ack for distribution")
	sigma, err := signer.Sign(txRaw)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to sign ack response")
	}

	// Ack for distribution
	// Send the signature back
	session := context.Session()
	logger.DebugfContext(context.Context(), "ack response: [%s] from [%s]", utils.Hashable(sigma), defaultIdentity)
	if err := session.SendWithContext(context.Context(), sigma); err != nil {
		return nil, errors.WithMessagef(err, "failed sending ack")
	}

	// cache the token request into the tokens db
	t, err := tokens.GetService(context, s.tx.TMSID())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tokens db for [%s]", s.tx.TMSID())
	}

	if err := t.CacheRequest(context.Context(), s.tx.TMSID(), s.tx.TokenRequest); err != nil {
		logger.WarnfContext(context.Context(), "failed to cache token request [%s], this might cause delay, investigate when possible: [%s]", s.tx.TokenRequest.Anchor, err)
	}

	labels := []string{
		"network", s.tx.Network(),
		"channel", s.tx.Channel(),
		"namespace", s.tx.Namespace(),
	}
	GetMetrics(context).AcceptedTransactions.With(labels...).Add(1)

	return s.tx, nil
}

func (s *AcceptView) respondToSignatureRequests(context view.Context) error {
	requestsToBeSigned, err := requestsToBeSigned(context.Context(), s.tx.TokenRequest)
	if err != nil {
		return errors.Wrapf(err, "failed collecting requests of signature")
	}
	logger.DebugfContext(context.Context(), "respond to signature requests [%s][%d]", s.tx.ID(), len(requestsToBeSigned))

	session := context.Session()
	for i := range requestsToBeSigned {
		logger.DebugfContext(context.Context(), "Sign request no %d", i)
		signatureRequest := &SignatureRequest{}

		if i == 0 {
			logger.DebugfContext(context.Context(), "First request is fetched from KVS")
			k, err := kvs.CreateCompositeKey("signatureRequest", []string{s.tx.ID()})
			if err != nil {
				return errors.Wrap(err, "failed to generate key to store signature request")
			}
			var srStr string
			if kvss, err := context.GetService(&kvs.KVS{}); err != nil {
				return errors.Wrap(err, "failed to get KVS from context")
			} else if err := kvss.(*kvs.KVS).Get(context.Context(), k, &srStr); err != nil {
				return errors.Wrap(err, "failed to store signature request")
			}
			srRaw, err := base64.StdEncoding.DecodeString(srStr)
			if err != nil {
				return errors.Wrap(err, "failed to decode signature request")
			}
			if err := Unmarshal(srRaw, signatureRequest); err != nil {
				return errors.Wrap(err, "failed unmarshalling signature request")
			}
		} else {
			logger.DebugfContext(context.Context(), "Receiving signature request...")
			jsonSession := session2.JSON(context)
			err := jsonSession.ReceiveWithTimeout(signatureRequest, time.Minute)
			if err != nil {
				return errors.Wrap(err, "failed reading signature request")
			}
		}
		logger.DebugfContext(context.Context(), "Fetched request from session")
		tms, err := token.GetManagementService(context, token.WithTMS(s.tx.Network(), s.tx.Channel(), s.tx.Namespace()))
		if err != nil {
			return errors.Wrapf(err, "failed getting TMS for [%s:%s:%s]", s.tx.Network(), s.tx.Channel(), s.tx.Namespace())
		}

		if !tms.SigService().IsMe(context.Context(), signatureRequest.Signer) {
			return errors.Errorf("identity [%s] is not me", signatureRequest.Signer.UniqueID())
		}
		signer, err := s.tx.TokenService().SigService().GetSigner(context.Context(), signatureRequest.Signer)
		if err != nil {
			return errors.Wrapf(err, "cannot find signer for [%s]", signatureRequest.Signer.UniqueID())
		}
		logger.DebugfContext(context.Context(), "Sign message")
		sigma, err := signer.Sign(signatureRequest.MessageToSign())
		if err != nil {
			return errors.Wrapf(err, "failed signing request")
		}
		logger.DebugfContext(context.Context(), "Send back signature...")

		err = session.SendWithContext(context.Context(), sigma)
		if err != nil {
			return errors.Wrapf(err, "failed sending signature back")
		}
	}

	if len(requestsToBeSigned) > 0 {
		logger.DebugfContext(context.Context(), "wait the transaction to be sent back [%s]", s.tx.ID())
		// expect again to receive a transaction
		tx, err := ReceiveTransaction(context)
		if err != nil {
			return errors.Wrapf(err, "expected to receive a transaction")
		}
		// TODO: check that the token requests match
		s.tx = tx
		logger.DebugfContext(context.Context(), "wait the transaction to be sent back [%s], received", s.tx.ID())
	} else {
		logger.DebugfContext(context.Context(), "no need to wait the transaction to be sent back [%s]", s.tx.ID())
	}

	return nil
}
