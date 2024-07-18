/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"context"

	errors2 "github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

const (
	AddToken    = "store-token"
	DeleteToken = "delete-token"
)

type Flags struct {
	Mine    bool
	Auditor bool
	Issuer  bool
}

type DBStorage struct {
	notifier events.Publisher
	tokenDB  *tokendb.DB
	tmsID    token.TMSID
	ote      OwnerTypeExtractor
}

func NewDBStorage(notifier events.Publisher, ote OwnerTypeExtractor, tokenDB *tokendb.DB, tmsID token.TMSID) (*DBStorage, error) {
	return &DBStorage{notifier: notifier, ote: ote, tokenDB: tokenDB, tmsID: tmsID}, nil
}

func (d *DBStorage) NewTransaction(ctx context.Context) (*transaction, error) {
	tx, err := d.tokenDB.NewTransaction(ctx)
	if err != nil {
		return nil, err
	}
	return &transaction{
		notifier: d.notifier,
		tx:       tx,
		tmsID:    d.tmsID,
		ote:      d.ote,
	}, nil
}

func (d *DBStorage) TransactionExists(ctx context.Context, id string) (bool, error) {
	return d.tokenDB.TransactionExists(ctx, id)
}

func (d *DBStorage) StorePublicParams(raw []byte) error {
	return d.tokenDB.StorePublicParams(raw)
}

type transaction struct {
	notifier events.Publisher
	tx       *tokendb.Transaction
	tmsID    token.TMSID
	ote      OwnerTypeExtractor
}

type OwnerTypeExtractor interface {
	OwnerType(raw []byte) (string, []byte, error)
}

func NewTransaction(notifier events.Publisher, tx *tokendb.Transaction, tmsID token.TMSID) (*transaction, error) {
	return &transaction{
		notifier: notifier,
		tx:       tx,
		tmsID:    tmsID,
	}, nil
}

func (t *transaction) DeleteToken(ctx context.Context, txID string, index uint64, deletedBy string) error {
	span := trace.SpanFromContext(ctx)
	span.AddEvent("get_token")
	tok, owners, err := t.tx.GetToken(ctx, txID, index, true)
	if err != nil {
		return errors.WithMessagef(err, "failed to get token [%s:%d]", txID, index)
	}
	span.AddEvent("delete_token")
	err = t.tx.Delete(ctx, txID, index, deletedBy)
	if err != nil {
		if tok == nil {
			logger.Debugf("nothing further to delete for [%s:%d]", txID, index)
			return nil
		}
		return errors.WithMessagef(err, "failed to delete token [%s:%d]", txID, index)
	}
	if tok == nil {
		logger.Debugf("nothing further to delete for [%s:%d]", txID, index)
		return nil
	}
	span.AddEvent("notify_owners")
	for _, owner := range owners {
		logger.Debugf("post new delete-token event [%s:%s:%s]", txID, index, owner)
		t.Notify(DeleteToken, t.tmsID, owner, tok.Type, txID, index)
	}
	return nil
}

func (t *transaction) DeleteTokens(ctx context.Context, deletedBy string, ids []*token2.ID) error {
	for _, id := range ids {
		if err := t.DeleteToken(ctx, id.TxId, id.Index, deletedBy); err != nil {
			return err
		}
	}
	return nil
}

type TokenToAppend struct {
	txID                  string
	index                 uint64
	tok                   *token2.Token
	tokenOnLedger         []byte
	tokenOnLedgerMetadata []byte
	owners                []string
	issuer                token.Identity
	precision             uint64
	flags                 Flags
}

func (t *transaction) AppendToken(ctx context.Context, tta TokenToAppend) error {
	span := trace.SpanFromContext(ctx)
	q, err := token2.ToQuantity(tta.tok.Quantity, tta.precision)
	if err != nil {
		return errors.Wrapf(err, "cannot covert [%s] with precision [%d]", tta.tok.Quantity, tta.precision)
	}

	typ, id, err := t.ote.OwnerType(tta.tok.Owner.Raw)
	if err != nil {
		logger.Errorf("could not unmarshal identity when storing token: %s", err.Error())
		return errors.Wrap(err, "could not unmarshal identity when storing token")
	}

	span.AddEvent("store_token")
	err = t.tx.StoreToken(ctx, tokendb.TokenRecord{
		TxID:           tta.txID,
		Index:          tta.index,
		IssuerRaw:      tta.issuer,
		OwnerRaw:       tta.tok.Owner.Raw,
		OwnerType:      typ,
		OwnerIdentity:  id,
		Ledger:         tta.tokenOnLedger,
		LedgerMetadata: tta.tokenOnLedgerMetadata,
		Quantity:       tta.tok.Quantity,
		Type:           tta.tok.Type,
		Amount:         q.ToBigInt().Uint64(),
		Owner:          tta.flags.Mine,
		Auditor:        tta.flags.Auditor,
		Issuer:         tta.flags.Issuer,
	}, tta.owners)
	if err != nil && !errors2.HasCause(err, driver.UniqueKeyViolation) {
		return errors.Wrapf(err, "cannot store token in db")
	}

	span.AddEvent("notify_owners")
	for _, id := range tta.owners {
		if len(id) == 0 {
			continue
		}
		t.Notify(AddToken, t.tmsID, id, tta.tok.Type, tta.txID, tta.index)
	}

	return nil
}

func (t *transaction) Notify(topic string, tmsID token.TMSID, walletID, tokenType, txID string, index uint64) {
	if t.notifier == nil {
		logger.Warnf("cannot notify others!")
		return
	}

	e := NewTokenProcessorEvent(topic, &TokenMessage{
		TMSID:     tmsID,
		WalletID:  walletID,
		TokenType: tokenType,
		TxID:      txID,
		Index:     index,
	})

	logger.Debugf("Publish new event %v", e)
	t.notifier.Publish(e)
}

func (t *transaction) Rollback() error {
	return t.tx.Rollback()
}

func (t *transaction) Commit() error {
	return t.tx.Commit()
}

type TokenProcessorEvent struct {
	topic   string
	message TokenMessage
}

func NewTokenProcessorEvent(topic string, message *TokenMessage) *TokenProcessorEvent {
	return &TokenProcessorEvent{
		topic:   topic,
		message: *message,
	}
}

type TokenMessage struct {
	TMSID     token.TMSID
	WalletID  string
	TokenType string
	TxID      string
	Index     uint64
}

func (t *TokenProcessorEvent) Topic() string {
	return t.topic
}

func (t *TokenProcessorEvent) Message() interface{} {
	return t.message
}
