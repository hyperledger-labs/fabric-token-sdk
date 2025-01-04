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
}

func NewDBStorage(notifier events.Publisher, tokenDB *tokendb.DB, tmsID token.TMSID) (*DBStorage, error) {
	return &DBStorage{
		notifier: notifier,
		tokenDB:  tokenDB,
		tmsID:    tmsID,
	}, nil
}

func (d *DBStorage) NewTransaction(ctx context.Context) (*transaction, error) {
	tx, err := d.tokenDB.NewTransaction(ctx)
	if err != nil {
		return nil, err
	}
	return NewTransaction(ctx, d.notifier, tx, d.tmsID)
}

func (d *DBStorage) TransactionExists(ctx context.Context, id string) (bool, error) {
	return d.tokenDB.TransactionExists(ctx, id)
}

func (d *DBStorage) StorePublicParams(raw []byte) error {
	return d.tokenDB.StorePublicParams(raw)
}

type TokenToAppend struct {
	txID                  string
	index                 uint64
	tok                   *token2.Token
	tokenOnLedgerType     token2.TokenType
	tokenOnLedger         []byte
	tokenOnLedgerMetadata []byte
	ownerType             string
	ownerIdentity         token.Identity
	ownerWalletID         string
	owners                []string
	issuer                token.Identity
	precision             uint64
	flags                 Flags
}

type transaction struct {
	ctx      context.Context
	notifier events.Publisher
	tx       *tokendb.Transaction
	tmsID    token.TMSID
}

func NewTransaction(ctx context.Context, notifier events.Publisher, tx *tokendb.Transaction, tmsID token.TMSID) (*transaction, error) {
	return &transaction{
		ctx:      ctx,
		notifier: notifier,
		tx:       tx,
		tmsID:    tmsID,
	}, nil
}

func (t *transaction) DeleteToken(tokenID token2.ID, deletedBy string) error {
	span := trace.SpanFromContext(t.ctx)
	span.AddEvent("get_token")
	tok, owners, err := t.tx.GetToken(tokenID, true)
	if err != nil {
		return errors.WithMessagef(err, "failed to get token [%s]", tokenID)
	}
	span.AddEvent("delete_token")
	err = t.tx.Delete(tokenID, deletedBy)
	if err != nil {
		if tok == nil {
			logger.Debugf("nothing further to delete for [%s]", tokenID)
			return nil
		}
		return errors.WithMessagef(err, "failed to delete token [%s]", tokenID)
	}
	if tok == nil {
		logger.Debugf("nothing further to delete for [%s]", tokenID)
		return nil
	}
	span.AddEvent("notify_owners")
	for _, owner := range owners {
		logger.Debugf("post new delete-token event [%s:%s]", tokenID, owner)
		t.Notify(DeleteToken, t.tmsID, owner, tok.Type, tokenID.TxId, tokenID.Index)
	}
	return nil
}

func (t *transaction) DeleteTokens(deletedBy string, ids []*token2.ID) error {
	for _, id := range ids {
		if err := t.DeleteToken(*id, deletedBy); err != nil {
			return err
		}
	}
	return nil
}

func (t *transaction) AppendToken(tta TokenToAppend) error {
	span := trace.SpanFromContext(t.ctx)
	q, err := token2.ToQuantity(tta.tok.Quantity, tta.precision)
	if err != nil {
		return errors.Wrapf(err, "cannot covert [%s] with precision [%d]", tta.tok.Quantity, tta.precision)
	}

	span.AddEvent("store_token")
	err = t.tx.StoreToken(tokendb.TokenRecord{
		TxID:           tta.txID,
		Index:          tta.index,
		IssuerRaw:      tta.issuer,
		OwnerRaw:       tta.tok.Owner,
		OwnerType:      tta.ownerType,
		OwnerIdentity:  tta.ownerIdentity,
		OwnerWalletID:  tta.ownerWalletID,
		Ledger:         tta.tokenOnLedger,
		LedgerType:     tta.tokenOnLedgerType,
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

func (t *transaction) Notify(topic string, tmsID token.TMSID, walletID string, tokenType token2.TokenType, txID string, index uint64) {
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

func (t *transaction) SetSpendableFlag(value bool, ids []*token2.ID) error {
	var err error
	for _, id := range ids {
		err = t.tx.SetSpendable(*id, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *transaction) SetSpendableBySupportedTokenTypes(supportedTokens []token2.TokenType) error {
	return t.tx.SetSpendableBySupportedTokenTypes(supportedTokens)
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
	TokenType token2.TokenType
	TxID      string
	Index     uint64
}

func (t *TokenProcessorEvent) Topic() string {
	return t.topic
}

func (t *TokenProcessorEvent) Message() interface{} {
	return t.message
}
