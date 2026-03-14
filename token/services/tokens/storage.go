/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokendb"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	AddToken    = "store-token"
	DeleteToken = "delete-token"
)

// Flags represents the ownership and auditing roles associated with a token.
type Flags struct {
	// Mine is true if the token belongs to one of my wallets.
	Mine bool
	// Auditor is true if I am an auditor for this token.
	Auditor bool
	// Issuer is true if I issued this token.
	Issuer bool
}

// DBStorage provides a high-level wrapper over TokenDB for managing token persistence.
// It handles transaction orchestration and event notification.
type DBStorage struct {
	// notifier is used to publish events when tokens are added or deleted.
	notifier events.Publisher
	// tokenDB is the underlying persistent store for tokens.
	tokenDB *tokendb.StoreService
	// tmsID is the identifier for the TMS this storage belongs to.
	tmsID token.TMSID
}

// NewDBStorage creates a new DBStorage instance.
func NewDBStorage(notifier events.Publisher, tokenDB *tokendb.StoreService, tmsID token.TMSID) (*DBStorage, error) {
	return &DBStorage{
		notifier: notifier,
		tokenDB:  tokenDB,
		tmsID:    tmsID,
	}, nil
}

// NewTransaction starts a new transaction for local storage operations.
func (d *DBStorage) NewTransaction() (*transaction, error) {
	tx, err := d.tokenDB.NewTransaction()
	if err != nil {
		return nil, err
	}

	return NewTransaction(d.notifier, tx, d.tmsID)
}

// TransactionExists checks if a transaction with the given ID has already been recorded in local storage.
func (d *DBStorage) TransactionExists(ctx context.Context, id string) (bool, error) {
	return d.tokenDB.TransactionExists(ctx, id)
}

// StorePublicParams persists the public parameters associated with the TMS.
func (d *DBStorage) StorePublicParams(ctx context.Context, raw []byte) error {
	return d.tokenDB.StorePublicParams(ctx, raw)
}

// TokenToAppend contains the detailed information required to store a new token in the database.
type TokenToAppend struct {
	// txID is the transaction ID that created this token.
	txID string
	// index is the output index of the token within the transaction.
	index uint64
	// tok is the de-obfuscated token information.
	tok *token2.Token
	// tokenOnLedgerFormat is the format of the token as it appears on the ledger.
	tokenOnLedgerFormat token2.Format
	// tokenOnLedger is the raw byte representation of the token output on the ledger.
	tokenOnLedger []byte
	// tokenOnLedgerMetadata is the metadata associated with the token output on the ledger.
	tokenOnLedgerMetadata []byte
	// ownerType is the type of the token owner's identity.
	ownerType string
	// ownerIdentity is the de-obfuscated identity of the owner.
	ownerIdentity token.Identity
	// ownerWalletID is the local wallet identifier if the owner is mine.
	ownerWalletID string
	// owners is the list of unique identifiers for all recipients/owners.
	owners []string
	// issuer is the identity of the token issuer.
	issuer token.Identity
	// precision is the number of decimal places for quantity calculations.
	precision uint64
	// flags indicates my relationship with this token.
	flags Flags
}

// transaction encapsulates a single atomic update to the token database.
type transaction struct {
	// notifier is used to publish events upon successful deletion or addition.
	notifier events.Publisher
	// tx is the underlying database transaction.
	tx *tokendb.Transaction
	// tmsID is the TMS identifier for the transaction.
	tmsID token.TMSID
}

// NewTransaction creates a new transaction wrapper.
func NewTransaction(notifier events.Publisher, tx *tokendb.Transaction, tmsID token.TMSID) (*transaction, error) {
	return &transaction{
		notifier: notifier,
		tx:       tx,
		tmsID:    tmsID,
	}, nil
}

// DeleteToken removes a single token from the database and notifies listeners.
func (t *transaction) DeleteToken(ctx context.Context, tokenID token2.ID, deletedBy string) error {
	tok, owners, err := t.tx.GetToken(ctx, tokenID, true)
	if err != nil {
		return errors.WithMessagef(err, "failed to get token [%s]", tokenID)
	}

	err = t.tx.Delete(ctx, tokenID, deletedBy)
	if err != nil {
		if tok == nil {
			logger.DebugfContext(ctx, "nothing further to delete for [%s]", tokenID)

			return nil
		}

		return errors.WithMessagef(err, "failed to delete token [%s]", tokenID)
	}
	if tok == nil {
		logger.DebugfContext(ctx, "nothing further to delete for [%s]", tokenID)

		return nil
	}
	logger.DebugfContext(ctx, "Notify owners")
	for _, owner := range owners {
		logger.DebugfContext(ctx, "post new delete-token event [%s:%s]", tokenID, owner)
		t.Notify(ctx, DeleteToken, t.tmsID, owner, tok.Type, tokenID.TxId, tokenID.Index)
	}

	return nil
}

// DeleteTokens removes multiple tokens from the database.
func (t *transaction) DeleteTokens(ctx context.Context, deletedBy string, ids []*token2.ID) error {
	for _, id := range ids {
		if err := t.DeleteToken(ctx, *id, deletedBy); err != nil {
			return err
		}
	}

	return nil
}

// AppendToken records a new token in the database and notifies listeners.
func (t *transaction) AppendToken(ctx context.Context, tta TokenToAppend) error {
	q, err := token2.ToQuantity(tta.tok.Quantity, tta.precision)
	if err != nil {
		return errors.Wrapf(err, "cannot covert [%s] with precision [%d]", tta.tok.Quantity, tta.precision)
	}

	err = t.tx.StoreToken(ctx, tokendb.TokenRecord{
		TxID:           tta.txID,
		Index:          tta.index,
		IssuerRaw:      tta.issuer,
		OwnerRaw:       tta.tok.Owner,
		OwnerType:      tta.ownerType,
		OwnerIdentity:  tta.ownerIdentity,
		OwnerWalletID:  tta.ownerWalletID,
		Ledger:         tta.tokenOnLedger,
		LedgerFormat:   tta.tokenOnLedgerFormat,
		LedgerMetadata: tta.tokenOnLedgerMetadata,
		Quantity:       tta.tok.Quantity,
		Type:           tta.tok.Type,
		Amount:         q.ToBigInt().Uint64(),
		Owner:          tta.flags.Mine,
		Auditor:        tta.flags.Auditor,
		Issuer:         tta.flags.Issuer,
	}, tta.owners)
	if err != nil && !errors.HasCause(err, driver.UniqueKeyViolation) {
		return errors.Wrapf(err, "cannot store token in db")
	}

	logger.DebugfContext(ctx, "Notify owners")
	for _, id := range tta.owners {
		if len(id) == 0 {
			continue
		}
		t.Notify(ctx, AddToken, t.tmsID, id, tta.tok.Type, tta.txID, tta.index)
	}

	return nil
}

// Notify publishes a token-related event to the system's notification bus.
func (t *transaction) Notify(ctx context.Context, topic string, tmsID token.TMSID, walletID string, tokenType token2.Type, txID string, index uint64) {
	if t.notifier == nil {
		logger.WarnfContext(ctx, "cannot notify others!")

		return
	}

	e := NewTokenProcessorEvent(topic, &TokenMessage{
		TMSID:     tmsID,
		WalletID:  walletID,
		TokenType: tokenType,
		TxID:      txID,
		Index:     index,
	})

	logger.DebugfContext(ctx, "publish new event %v", e)
	t.notifier.Publish(e)
}

// Rollback cancels all changes made in the transaction.
func (t *transaction) Rollback() error {
	return t.tx.Rollback()
}

// Commit persists all changes made in the transaction.
func (t *transaction) Commit() error {
	return t.tx.Commit()
}

// SetSpendableFlag updates the spendable status for the given tokens in the database.
func (t *transaction) SetSpendableFlag(ctx context.Context, value bool, ids []*token2.ID) error {
	var err error
	for _, id := range ids {
		err = t.tx.SetSpendable(ctx, *id, value)
		if err != nil {
			return err
		}
	}

	return nil
}

// SetSpendableBySupportedTokenTypes marks all tokens matching the given formats as spendable.
func (t *transaction) SetSpendableBySupportedTokenTypes(ctx context.Context, supportedTokens []token2.Format) error {
	return t.tx.SetSpendableBySupportedTokenFormats(ctx, supportedTokens)
}

// TokenProcessorEvent encapsulates a message published by the Tokens Service.
type TokenProcessorEvent struct {
	topic   string
	message TokenMessage
}

// NewTokenProcessorEvent creates a new event with the given topic and message.
func NewTokenProcessorEvent(topic string, message *TokenMessage) *TokenProcessorEvent {
	return &TokenProcessorEvent{
		topic:   topic,
		message: *message,
	}
}

// TokenMessage contains the details of a token-related event.
type TokenMessage struct {
	// TMSID is the TMS identifier for the event.
	TMSID token.TMSID
	// WalletID is the unique identifier of the wallet affected by the event.
	WalletID string
	// TokenType is the type of the token.
	TokenType token2.Type
	// TxID is the transaction ID.
	TxID string
	// Index is the token index within the transaction.
	Index uint64
}

// Topic returns the event's topic.
func (t *TokenProcessorEvent) Topic() string {
	return t.topic
}

// Message returns the event's payload.
func (t *TokenProcessorEvent) Message() interface{} {
	return t.message
}
