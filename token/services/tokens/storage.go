/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
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
	return &DBStorage{notifier: notifier, tokenDB: tokenDB, tmsID: tmsID}, nil
}

func (d *DBStorage) NewTransaction() (*transaction, error) {
	tx, err := d.tokenDB.NewTransaction()
	if err != nil {
		return nil, err
	}
	return NewTransaction(d.notifier, tx, d.tmsID)
}

func (d *DBStorage) StorePublicParams(raw []byte) error {
	return d.tokenDB.StorePublicParams(raw)
}

type transaction struct {
	notifier events.Publisher
	tx       *tokendb.Transaction
	tmsID    token.TMSID
}

func NewTransaction(notifier events.Publisher, tx *tokendb.Transaction, tmsID token.TMSID) (*transaction, error) {
	return &transaction{
		notifier: notifier,
		tx:       tx,
		tmsID:    tmsID,
	}, nil
}

func (t *transaction) DeleteToken(txID string, index uint64, deletedBy string) error {
	tok, owners, err := t.tx.GetToken(txID, index, true)
	if err != nil {
		return errors.WithMessagef(err, "failed to get token [%s:%d]", txID, index)
	}
	err = t.tx.Delete(txID, index, deletedBy)
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
	for _, owner := range owners {
		logger.Debugf("post new delete-token event [%s:%s:%s]", txID, index, owner)
		t.Notify(DeleteToken, t.tmsID, owner, tok.Type, txID, index)
	}
	return nil
}

func (t *transaction) DeleteTokens(deletedBy string, ids []*token2.ID) error {
	for _, id := range ids {
		if err := t.DeleteToken(id.TxId, id.Index, deletedBy); err != nil {
			return err
		}
	}
	return nil
}

func (t *transaction) AppendToken(
	txID string,
	index uint64,
	tok *token2.Token,
	tokenOnLedger []byte,
	tokenOnLedgerMetadata []byte,
	ids []string,
	issuer view.Identity,
	precision uint64,
	flags Flags,
) error {
	q, err := token2.ToQuantity(tok.Quantity, precision)
	if err != nil {
		return errors.Wrapf(err, "cannot covert [%s] with precision [%d]", tok.Quantity, precision)
	}
	id, err := identity.UnmarshalTypedIdentity(tok.Owner.Raw)
	if err != nil {
		logger.Errorf("could not unmarshal identity when storing token: %s", err.Error())
		return errors.Wrap(err, "could not unmarshal identity when storing token")
	}

	err = t.tx.StoreToken(
		tokendb.TokenRecord{
			TxID:           txID,
			Index:          index,
			IssuerRaw:      issuer,
			OwnerRaw:       tok.Owner.Raw,
			OwnerType:      id.Type,
			OwnerIdentity:  id.Identity,
			Ledger:         tokenOnLedger,
			LedgerMetadata: tokenOnLedgerMetadata,
			Quantity:       tok.Quantity,
			Type:           tok.Type,
			Amount:         q.ToBigInt().Uint64(),
			Owner:          flags.Mine,
			Auditor:        flags.Auditor,
			Issuer:         flags.Issuer,
		},
		ids,
	)
	if err != nil {
		return errors.Wrapf(err, "cannot store token in db")
	}

	for _, id := range ids {
		if len(id) == 0 {
			continue
		}
		t.Notify(AddToken, t.tmsID, id, tok.Type, txID, index)
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

func (t *transaction) TransactionExists(id string) (bool, error) {
	return t.tx.TransactionExists(id)
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
