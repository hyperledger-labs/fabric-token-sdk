/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"go.opentelemetry.io/otel/trace"
)

type transactionDB interface {
	GetTokenRequest(ctx context.Context, txID string) ([]byte, error)
	SetStatus(ctx context.Context, txID string, status driver.TxStatus, message string) error
}

type TokenManagementServiceProvider interface {
	GetManagementService(opts ...token.ServiceOption) (*token.ManagementService, error)
}

type FinalityListener struct {
	logger      logging.Logger
	tmsProvider TokenManagementServiceProvider
	tmsID       token.TMSID
	ttxDB       transactionDB
	tokens      *tokens.Service
	tracer      trace.Tracer
	retryRunner common.RetryRunner
}

func NewFinalityListener(logger logging.Logger, tmsProvider TokenManagementServiceProvider, tmsID token.TMSID, ttxDB transactionDB, tokens *tokens.Service, tracer trace.Tracer) *FinalityListener {
	return &FinalityListener{
		logger:      logger,
		tmsProvider: tmsProvider,
		tmsID:       tmsID,
		ttxDB:       ttxDB,
		tokens:      tokens,
		tracer:      tracer,
		retryRunner: common.NewRetryRunner(common.Infinitely, time.Second, true),
	}
}

func (t *FinalityListener) OnStatus(ctx context.Context, txID string, status int, message string, tokenRequestHash []byte) {
	newCtx, span := t.tracer.Start(ctx, "on_status")
	defer span.End()
	if err := t.retryRunner.Run(func() error { return t.runOnStatus(newCtx, txID, status, message, tokenRequestHash) }); err != nil {
		t.logger.Errorf("Listener failed")
	}
}

func (t *FinalityListener) runOnStatus(ctx context.Context, txID string, status int, message string, tokenRequestHash []byte) error {
	t.logger.DebugfContext(ctx, "tx status changed for tx [%s]: [%s]", txID, status)
	var txStatus driver.TxStatus
	switch status {
	case network.Valid:
		txStatus = driver.Confirmed
		t.logger.DebugfContext(ctx, "get token request for [%s]", txID)

		tr, msgToSign := t.tokens.GetCachedTokenRequest(txID)
		if tr == nil {
			// load it
			tokenRequestRaw, err := t.ttxDB.GetTokenRequest(ctx, txID)
			if err != nil {
				t.logger.ErrorfContext(ctx, "failed retrieving token request [%s]: [%s]", txID, err)
				return fmt.Errorf("failed retrieving token request [%s]: [%w]", txID, err)
			}
			t.logger.DebugfContext(ctx, "Read token request")
			tms, err := t.tmsProvider.GetManagementService(token.WithTMSID(t.tmsID))
			if err != nil {
				return fmt.Errorf("failed retrieving token request [%s]: [%w]", txID, err)
			}
			tr, err = tms.NewFullRequestFromBytes(tokenRequestRaw)
			if err != nil {
				return fmt.Errorf("failed retrieving token request [%s]: [%w]", txID, err)
			}
			msgToSign, err = tr.MarshalToSign()
			if err != nil {
				return fmt.Errorf("failed retrieving token request [%s]: [%w]", txID, err)
			}
		}
		t.logger.DebugfContext(ctx, "Check token request")
		if err := t.checkTokenRequest(txID, msgToSign, tokenRequestHash); err != nil {
			t.logger.ErrorfContext(ctx, "tx [%s], %s", txID, err)
			txStatus = driver.Deleted
			message = err.Error()
		} else {
			t.logger.DebugfContext(ctx, "append token request for [%s]", txID)
			if err := t.tokens.Append(ctx, t.tmsID, token.RequestAnchor(txID), tr); err != nil {
				// at this stage though, we don't fail here because the commit pipeline is processing the tokens still
				t.logger.ErrorfContext(ctx, "failed to append token request to token db [%s]: [%s]", txID, err)
				return fmt.Errorf("failed to append token request to token db [%s]: [%w]", txID, err)
			}
			t.logger.DebugfContext(ctx, "append token request for [%s], done", txID)
		}
	case network.Invalid:
		txStatus = driver.Deleted
	}
	if err := t.ttxDB.SetStatus(ctx, txID, txStatus, message); err != nil {
		t.logger.ErrorfContext(ctx, "<message> [%s]: [%s]", txID, err)
		return fmt.Errorf("<message> [%s]: [%w]", txID, err)
	}
	t.logger.DebugfContext(ctx, "tx status changed for tx [%s]: [%s] done", txID, status)
	return nil
}

func (t *FinalityListener) checkTokenRequest(txID string, trToSign []byte, reference []byte) error {
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
