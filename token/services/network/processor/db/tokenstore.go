/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/processor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.network.processor")

type TokenStore struct {
	notifier events.Publisher
	tokenDB  *tokendb.DB
	tmsID    token.TMSID
}

func NewTokenStore(notifier events.Publisher, tokenDB *tokendb.DB, tmsID token.TMSID) (*TokenStore, error) {
	return &TokenStore{
		notifier: notifier,
		tokenDB:  tokenDB,
		tmsID:    tmsID,
	}, nil
}

func (t *TokenStore) DeleteToken(ns string, txID string, index uint64, rws processor.RWSet, deletedBy string) error {
	tok, owners, err := t.tokenDB.OwnersOf(ns, txID, index)
	if err != nil {
		return errors.WithMessagef(err, "failed to get owners for token [%s:%s:%d]", ns, txID, index)
	}
	err = t.tokenDB.Delete(ns, txID, index, deletedBy)
	if err != nil {
		if tok == nil {
			logger.Debugf("nothing further to delete for [%s:%d]", txID, index)
			return nil
		}
		return errors.WithMessagef(err, "failed to delete token [%s:%s:%d]", ns, txID, index)
	}
	for _, id := range owners {
		logger.Debugf("post new delete-token event")
		t.Notify(processor.DeleteToken, t.tmsID, id, tok.Type, txID, index)
	}
	return nil
}

func (t *TokenStore) StoreToken(ns string, txID string, index uint64, tok *token2.Token, rws processor.RWSet, tokenOnLedger []byte, tokenOnLedgerMetadata []byte, ids []string) error {
	err := t.tokenDB.StoreOwnerToken(
		tokendb.TokenRecord{
			TxID:           txID,
			Index:          index,
			Namespace:      ns,
			IssuerRaw:      nil,
			OwnerRaw:       tok.Owner.Raw,
			Ledger:         tokenOnLedger,
			LedgerMetadata: tokenOnLedgerMetadata,
			Quantity:       tok.Quantity,
			Type:           tok.Type,
			Amount:         0,
		},
		ids,
	)
	if err != nil {
		return err
	}

	for _, id := range ids {
		if len(id) == 0 {
			continue
		}
		t.Notify(processor.AddToken, t.tmsID, id, tok.Type, txID, index)
	}

	return nil
}

func (t *TokenStore) StoreIssuedHistoryToken(ns string, txID string, index uint64, tok *token2.Token, rws processor.RWSet, tokenOnLedger []byte, tokenOnLedgerMetadata []byte, issuer view.Identity, precision uint64) error {
	return t.tokenDB.StoreIssuedToken(
		tokendb.TokenRecord{
			TxID:           txID,
			Index:          index,
			Namespace:      ns,
			IssuerRaw:      issuer,
			OwnerRaw:       tok.Owner.Raw,
			Ledger:         tokenOnLedger,
			LedgerMetadata: tokenOnLedgerMetadata,
			Quantity:       tok.Quantity,
			Type:           tok.Type,
			Amount:         0,
		},
	)
}

func (t *TokenStore) StoreAuditToken(ns string, txID string, index uint64, tok *token2.Token, rws processor.RWSet, tokenOnLedger []byte, tokenOnLedgerMetadata []byte) error {
	return t.tokenDB.StoreAuditToken(
		tokendb.TokenRecord{
			TxID:           txID,
			Index:          index,
			Namespace:      ns,
			IssuerRaw:      nil,
			OwnerRaw:       tok.Owner.Raw,
			Ledger:         tokenOnLedger,
			LedgerMetadata: tokenOnLedgerMetadata,
			Quantity:       tok.Quantity,
			Type:           tok.Type,
			Amount:         0,
		},
	)
}

func (t *TokenStore) StorePublicParams(ns string, val []byte) error {
	return t.tokenDB.StorePublicParams(val)
}

func (t *TokenStore) Notify(topic string, tmsID token.TMSID, walletID, tokenType, txID string, index uint64) {
	if t.notifier == nil {
		logger.Warnf("cannot notify others!")
		return
	}

	e := processor.NewTokenProcessorEvent(topic, &processor.TokenMessage{
		TMSID:     tmsID,
		WalletID:  walletID,
		TokenType: tokenType,
		TxID:      txID,
		Index:     index,
	})

	logger.Debugf("Publish new event %v", e)
	t.notifier.Publish(e)
}
