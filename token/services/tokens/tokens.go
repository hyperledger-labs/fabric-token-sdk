/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"context"
	"runtime/debug"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

var logger = logging.MustGetLogger("token-sdk.tokens")

type MetaData interface {
	SpentTokenID() []*token2.ID
	GetToken(raw []byte) (*token2.Token, token.Identity, []byte, error)
}

type GetTMSProviderFunc = func() *token.ManagementServiceProvider

// Transaction models a token transaction
type Transaction interface {
	ID() string
	Network() string
	Channel() string
	Namespace() string
	Request() *token.Request
}

type Cache interface {
	Get(key string) (*CacheEntry, bool)
	Add(key string, value *CacheEntry)
	Delete(key string)
}

type CacheEntry struct {
	Request  *token.Request
	ToSpend  []*token2.ID
	ToAppend []TokenToAppend
}

// Tokens is the interface for the token service
type Tokens struct {
	TMSProvider TMSProvider
	Storage     *DBStorage

	RequestsCache Cache
}

func (t *Tokens) Append(ctx context.Context, tmsID token.TMSID, txID string, request *token.Request) (err error) {
	span := trace.SpanFromContext(ctx)
	if request == nil {
		logger.Debugf("transaction [%s], no request found, skip it", txID)
		return nil
	}
	if request.Metadata == nil {
		logger.Debugf("transaction [%s], no metadata found, skip it", txID)
		return nil
	}

	span.AddEvent("check_tx_exists")
	exists, err := t.Storage.TransactionExists(ctx, txID)
	if err != nil {
		logger.Errorf("transaction [%s], failed to check existence in db [%s]", txID, err)
		return errors.WithMessagef(err, "transaction [%s], failed to check existence in db", txID)
	}
	if exists {
		logger.Debugf("transaction [%s], exists in db, skipping", txID)
		return nil
	}

	span.AddEvent("get_actions")
	toSpend, toAppend, err := t.getActions(tmsID, txID, request)
	if err != nil {
		return errors.WithMessagef(err, "transaction [%s], failed to extract actions", txID)
	}

	logger.Debugf("transaction [%s] start db transaction", txID)
	span.AddEvent("create_new_tx")
	ts, err := t.Storage.NewTransaction(ctx)
	if err != nil {
		return errors.WithMessagef(err, "transaction [%s], failed to start db transaction", txID)
	}
	defer func() {
		if err == nil {
			return
		}
		span.RecordError(err)
		if err1 := ts.Rollback(); err1 != nil {
			logger.Errorf("error rolling back [%s][%s]", err1, debug.Stack())
		} else {
			logger.Infof("transaction [%s] rolled back", txID)
		}
	}()
	span.AddEvent("append_tokens")
	for _, tta := range toAppend {
		err = ts.AppendToken(tta)
		if err != nil {
			return errors.WithMessagef(err, "transaction [%s], failed to append token", txID)
		}
	}
	span.AddEvent("delete_tokens")
	err = ts.DeleteTokens(txID, toSpend)
	if err != nil {
		return errors.WithMessagef(err, "transaction [%s], failed to delete tokens", txID)
	}
	span.AddEvent("commit")
	if err = ts.Commit(); err != nil {
		return errors.WithMessagef(err, "transaction [%s], failed to commit tokens to database", txID)
	}
	logger.Debugf("transaction [%s], committed tokens [%d:%d] to database", txID, len(toAppend), len(toSpend))

	return nil
}

func (t *Tokens) AppendRaw(ctx context.Context, tmsID token.TMSID, txID string, requestRaw []byte) (err error) {
	logger.Debugf("get tms for [%s]", txID)
	tms, err := t.TMSProvider.GetManagementService(token.WithTMSID(tmsID))
	if err != nil {
		return errors.WithMessagef(err, "failed getting token management service [%s]", tmsID)
	}
	logger.Debugf("get tms for [%s], done", txID)
	tr, err := tms.NewFullRequestFromBytes(requestRaw)
	if err != nil {
		return errors.WithMessagef(err, "failed unmarshal token request [%s]", txID)
	}
	logger.Debugf("append token request for [%s]", txID)
	return t.Append(ctx, tmsID, txID, tr)
}

func (t *Tokens) CacheRequest(tmsID token.TMSID, request *token.Request) error {
	toSpend, toAppend, err := t.extractActions(tmsID, request.Anchor, request)
	if err != nil {
		return errors.WithMessagef(err, "failed to extract actions for request [%s]", request.ID())
	}
	logger.Debugf("cache request [%s]", request.ID())
	// append to cache
	t.RequestsCache.Add(request.Anchor, &CacheEntry{
		Request:  request,
		ToSpend:  toSpend,
		ToAppend: toAppend,
	})
	return nil
}

func (t *Tokens) GetCachedTokenRequest(txID string) *token.Request {
	res, ok := t.RequestsCache.Get(txID)
	if !ok {
		return nil
	}
	return res.Request
}

// AppendTransaction appends the content of the passed transaction to the token db.
// If the transaction is already in there, nothing more happens.
// The operation is atomic.
func (t *Tokens) AppendTransaction(ctx context.Context, tx Transaction) (err error) {
	return t.Append(ctx, token.TMSID{
		Network:   tx.Network(),
		Channel:   tx.Channel(),
		Namespace: tx.Namespace(),
	}, tx.ID(), tx.Request())
}

// StorePublicParams stores the passed public parameters in the token db
func (t *Tokens) StorePublicParams(raw []byte) error {
	return t.Storage.StorePublicParams(raw)
}

// DeleteToken marks the entries corresponding to the passed token ids as deleted.
// The deletion is attributed to the passed deletedBy argument.
func (t *Tokens) DeleteToken(deletedBy string, ids ...*token2.ID) (err error) {
	return t.Storage.tokenDB.DeleteTokens(deletedBy, ids...)
}

func (t *Tokens) SetSpendableFlag(value bool, ids ...*token2.ID) error {
	tx, err := t.Storage.NewTransaction(context.TODO())
	if err != nil {
		return errors.Wrapf(err, "failed initiating transaction")
	}
	if err := tx.SetSpendableFlag(value, ids); err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			logger.Errorf("failed rolling back transaction that set spendable flag [%s]", err2)
		}
		return errors.Wrapf(err, "failed setting spendable flag")
	}
	return tx.Commit()
}

func (t *Tokens) SetSupportedTokens(types []string) error {
	tx, err := t.Storage.NewTransaction(context.TODO())
	if err != nil {
		return errors.WithMessagef(err, "error creating new transaction")
	}
	if err := tx.SetSupportedTokens(types); err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			logger.Errorf("error rolling back transaction: %v", err2)
		}
		return errors.WithMessagef(err, "error setting supported tokens")
	}
	if err := tx.Commit(); err != nil {
		return errors.WithMessagef(err, "error committing transaction")
	}
	return nil
}

func (t *Tokens) getActions(tmsID token.TMSID, txID string, request *token.Request) ([]*token2.ID, []TokenToAppend, error) {
	// check the cache first
	entry, ok := t.RequestsCache.Get(txID)
	if ok {
		return entry.ToSpend, entry.ToAppend, nil
	}
	// extract
	return t.extractActions(tmsID, txID, request)
}

func (t *Tokens) extractActions(tmsID token.TMSID, txID string, request *token.Request) ([]*token2.ID, []TokenToAppend, error) {
	tms, err := t.TMSProvider.GetManagementService(token.WithTMSID(tmsID))
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed getting token management service [%s]", tmsID)
	}

	logger.Debugf("transaction [%s on (%s)] is known, extract tokens", txID, tms.ID())
	pp := tms.PublicParametersManager().PublicParameters()
	graphHiding := pp.GraphHiding()
	precision := pp.Precision()
	auth := tms.Authorization()
	auditorFlag := auth.AmIAnAuditor()
	if auditorFlag {
		logger.Debugf("transaction [%s], I must be the auditor", txID)
	}
	md, err := request.GetMetadata()
	if err != nil {
		logger.Debugf("transaction [%s], failed to get metadata [%s]", txID, err)
		return nil, nil, errors.WithMessagef(err, "transaction [%s], failed to get request metadata", txID)
	}

	is, os, err := request.InputsAndOutputs()
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to get request's outputs")
	}
	toSpend, toAppend := t.parse(auth, txID, md, is, os, auditorFlag, precision, graphHiding)
	return toSpend, toAppend, nil
}

// parse returns the tokens to store and spend as the result of a transaction
func (t *Tokens) parse(
	auth driver.Authorization,
	txID string,
	md MetaData,
	is *token.InputStream,
	os *token.OutputStream,
	auditorFlag bool,
	precision uint64,
	graphHiding bool,
) (toSpend []*token2.ID, toAppend []TokenToAppend) {
	if graphHiding {
		ids := md.SpentTokenID()
		logger.Debugf("transaction [%s] with graph hiding, delete inputs [%v]", txID, ids)
		toSpend = append(toSpend, ids...)
	}

	logger.Debugf("parse [%d] inputs and [%d] outputs from [%s]", is.Count(), os.Count(), txID)

	// parse the inputs
	for _, input := range is.Inputs() {
		if input.Id == nil {
			logger.Debugf("transaction [%s] found an input that is not mine, skip it", txID)
			continue
		}
		logger.Debugf("transaction [%s] delete input [%s]", txID, input.Id)
		toSpend = append(toSpend, input.Id)
	}

	// parse the outputs
	for _, output := range os.Outputs() {
		// get token in the clear
		tok, issuer, tokenOnLedgerMetadata, err := md.GetToken(output.LedgerOutput)
		if err != nil {
			logger.Errorf("transaction [%s], found a token but failed getting the clear version, skipping it [%s]", txID, err)
			continue
		}
		if tok == nil {
			logger.Warnf("failed getting token in the clear for [%s, %s]", output.ID(txID), string(output.LedgerOutput))
			continue
		}

		if len(output.LedgerOutput) == 0 {
			logger.Debugf("transaction [%s] without graph hiding, delete input [%d]", txID, output.Index)
			toSpend = append(toSpend, &token2.ID{TxId: txID, Index: output.Index})
			continue
		}

		issuerFlag := !issuer.IsNone() && auth.Issued(issuer, tok)
		ownerWalletID, ids, mine := auth.IsMine(tok)
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			if mine {
				logger.Debugf("transaction [%s], found a token and it is mine", txID)
			} else {
				logger.Debugf("transaction [%s], found a token and it is NOT mine", txID)
			}
			if issuerFlag {
				logger.Debugf("transaction [%s], found a token and I have issued it", txID)
			}
			logger.Debugf("store token [%s:%d][%s]", txID, output.Index, hash.Hashable(output.LedgerOutput))
		}
		if !mine && !auditorFlag && !issuerFlag {
			logger.Debugf("transaction [%s], discarding token, not mine, not an auditor, not an issuer", txID)
			continue
		}

		ownerType, ownerIdentity, err := auth.OwnerType(tok.Owner)
		if err != nil {
			logger.Errorf("could not unmarshal identity when storing token: %s", err.Error())
			continue
		}

		tta := TokenToAppend{
			txID:                  txID,
			index:                 output.Index,
			tok:                   tok,
			tokenOnLedger:         output.LedgerOutput,
			tokenOnLedgerType:     output.LedgerOutputType,
			tokenOnLedgerMetadata: tokenOnLedgerMetadata,
			ownerType:             ownerType,
			ownerIdentity:         ownerIdentity,
			ownerWalletID:         ownerWalletID,
			owners:                ids,
			issuer:                issuer,
			precision:             precision,
			flags: Flags{
				Mine:    mine,
				Auditor: auditorFlag,
				Issuer:  issuerFlag,
			},
		}
		toAppend = append(toAppend, tta)

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("done parsing write key [%s]", output.ID(txID))
		}
	}
	return
}
