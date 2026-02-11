/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"go.opentelemetry.io/otel/trace"
)

type transactionDB interface {
	GetTokenRequest(ctx context.Context, txID string) ([]byte, error)
	SetStatus(ctx context.Context, txID string, status storage.TxStatus, message string) error
}

type Listener struct {
	logger      logging.Logger
	net         dep.Network
	namespace   string
	tmsProvider dep.TokenManagementServiceProvider
	tmsID       token.TMSID
	ttxDB       transactionDB
	tokens      *tokens.Service
	tracer      trace.Tracer
	retryRunner utils.RetryRunner
}

func NewListener(logger logging.Logger, net dep.Network, namespace string, tmsProvider dep.TokenManagementServiceProvider, tmsID token.TMSID, ttxDB transactionDB, tokens *tokens.Service, tracer trace.Tracer) *Listener {
	return &Listener{
		logger:      logger,
		net:         net,
		namespace:   namespace,
		tmsProvider: tmsProvider,
		tmsID:       tmsID,
		ttxDB:       ttxDB,
		tokens:      tokens,
		tracer:      tracer,
		retryRunner: utils.NewRetryRunner(logger, utils.Infinitely, time.Second, true),
	}
}

func (t *Listener) OnStatus(ctx context.Context, txID string, status int, message string, tokenRequestHash []byte) {
	newCtx, span := t.tracer.Start(ctx, "on_status")
	defer span.End()
	if err := t.retryRunner.Run(func() error {
		err := t.runOnStatus(newCtx, txID, status, message, tokenRequestHash)
		if err != nil {
			t.logger.Errorf("finality listener on [%s] failed with error: [%+v], retrying...", txID, err)
		}
		return err
	}); err != nil {
		t.logger.Errorf("finality listener on [%s] failed with error: [%+v], stop.", txID, err)
	}
}

func (t *Listener) runOnStatus(ctx context.Context, txID string, status int, message string, tokenRequestHash []byte) error {
	t.logger.InfofContext(ctx, "tx status changed for tx [%s]: [%s]", txID, status)
	var txStatus storage.TxStatus
	switch status {
	case network.Valid:
		txStatus = storage.Confirmed
		t.logger.InfofContext(ctx, "get token request for [%s]", txID)

		tr, msgToSign := t.tokens.GetCachedTokenRequest(txID)
		if tr == nil {
			// load it
			tokenRequestRaw, err := t.ttxDB.GetTokenRequest(ctx, txID)
			if err != nil {
				t.logger.ErrorfContext(ctx, "failed retrieving token request [%s]: [%s]", txID, err)

				return errors.Errorf("failed retrieving token request [%s]: [%w]", txID, err)
			}
			t.logger.InfofContext(ctx, "Read token request")
			tms, err := t.tmsProvider.TokenManagementService(token.WithTMSID(t.tmsID))
			if err != nil {
				return errors.Errorf("failed retrieving token request [%s]: [%w]", txID, err)
			}
			tr, err = tms.NewFullRequestFromBytes(tokenRequestRaw)
			if err != nil {
				return errors.Errorf("failed retrieving token request [%s]: [%w]", txID, err)
			}
			msgToSign, err = tr.MarshalToSign()
			if err != nil {
				return errors.Errorf("failed retrieving token request [%s]: [%w]", txID, err)
			}
		}
		t.logger.InfofContext(ctx, "Check token request")
		if err := t.checkTokenRequest(txID, msgToSign, tokenRequestHash); err != nil {
			t.logger.ErrorfContext(ctx, "tx [%s], %s", txID, err)
			txStatus = storage.Deleted
			message = err.Error()
		} else {
			t.logger.InfofContext(ctx, "append token request for [%s]", txID)
			if err := t.tokens.Append(ctx, t.tmsID, token.RequestAnchor(txID), tr); err != nil {
				// at this stage though, we don't fail here because the commit pipeline is processing the tokens still
				t.logger.ErrorfContext(ctx, "failed to append token request to token db [%s]: [%s]", txID, err)

				return errors.Errorf("failed to append token request to token db [%s]: [%w]", txID, err)
			}
			t.logger.InfofContext(ctx, "append token request for [%s], done", txID)
		}
	case network.Invalid:
		txStatus = storage.Deleted
	default:
		t.logger.Infof("listener invoked on [%s] with status [%d], listen again...", txID, status)
		// In this case, we do the following:
		// We return no error to terminate this listener, and we add it again for a second chance.
		if err := t.net.AddFinalityListener(t.namespace, txID, t); err != nil {
			return errors.Wrap(err, "failed to add finality listener")
		}
		return nil
	}

	// update the status, if here, either txStatus is Confirmed or Deleted
	if err := t.ttxDB.SetStatus(ctx, txID, txStatus, message); err != nil {
		t.logger.ErrorfContext(ctx, "<message> [%s]: [%s]", txID, err)

		return errors.Errorf("<message> [%s]: [%w]", txID, err)
	}
	t.logger.InfofContext(ctx, "tx status changed for tx [%s]: [%s] done", txID, status)
	return nil
}

func (t *Listener) checkTokenRequest(txID string, trToSign []byte, reference []byte) error {
	if base64.StdEncoding.EncodeToString(reference) != utils.Hashable(trToSign).String() {
		t.logger.Errorf("tx [%s], tr hashes [%s][%s]", txID, base64.StdEncoding.EncodeToString(reference), utils.Hashable(trToSign))
		// no further processing of the tokens of these transactions
		return errors.Errorf(
			"token requests do not match, tr hashes [%s][%s]",
			base64.StdEncoding.EncodeToString(reference),
			utils.Hashable(trToSign),
		)
	}

	return nil
}
