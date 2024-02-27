/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type tokenStore struct {
	notifier events.Publisher
	tokenDB  *tokendb.DB
	tmsID    token.TMSID
}

func NewTokenStore(notifier events.Publisher, tokenDB *tokendb.DB, tmsID token.TMSID) (*tokenStore, error) {
	return &tokenStore{
		notifier: notifier,
		tokenDB:  tokenDB,
		tmsID:    tmsID,
	}, nil
}

func (t *tokenStore) DeleteToken(txID string, index uint64, deletedBy string) error {
	tok, owners, err := t.tokenDB.OwnersOf(txID, index)
	if err != nil {
		return errors.WithMessagef(err, "failed to get owners for token [%s:%d]", txID, index)
	}
	err = t.tokenDB.Delete(txID, index, deletedBy)
	if err != nil {
		if tok == nil {
			logger.Debugf("nothing further to delete for [%s:%d]", txID, index)
			return nil
		}
		return errors.WithMessagef(err, "failed to delete token [%s:%d]", txID, index)
	}
	for _, id := range owners {
		logger.Debugf("post new delete-token event")
		t.Notify(DeleteToken, t.tmsID, id, tok.Type, txID, index)
	}
	return nil
}

func (t *tokenStore) AppendToken(
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
	err = t.tokenDB.StoreToken(
		tokendb.TokenRecord{
			TxID:           txID,
			Index:          index,
			IssuerRaw:      issuer,
			OwnerRaw:       tok.Owner.Raw,
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

func (t *tokenStore) StorePublicParams(val []byte) error {
	return t.tokenDB.StorePublicParams(val)
}

func (t *tokenStore) Notify(topic string, tmsID token.TMSID, walletID, tokenType, txID string, index uint64) {
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
