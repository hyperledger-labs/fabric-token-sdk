/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"fmt"
	"strconv"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	ndriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenStore struct {
	notifier events.Publisher
	tmsID    token.TMSID
	db       driver.TokenDB
}

func NewTokenStore(sp view2.ServiceProvider, tmsID token.TMSID) (*TokenStore, error) {
	walletID := fmt.Sprintf("%s-%s-%s", tmsID.Network, tmsID.Channel, tmsID.Namespace)
	db := ttxdb.Get(sp, tmsID.String(), walletID)
	if db == nil {
		return nil, errors.New("cannot get database")
	}

	notifier, err := events.GetPublisher(sp)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get event publisher")
	}

	return &TokenStore{notifier: notifier, tmsID: tmsID, db: db}, nil
}

func (cts *TokenStore) DeleteFabToken(ns string, txID string, index uint64, rws ndriver.RWSetProcessor, deletedBy string) error {
	logger.Debugf("spend token [%s,%d]", txID, index)
	err := cts.db.Delete(ns, txID, index, deletedBy)
	if err != nil {
		logger.Error(err)
	}
	return err
}

func (cts *TokenStore) StoreFabToken(ns string, txID string, index uint64, tok *token2.Token, rws ndriver.RWSetProcessor, infoRaw []byte, ids []string) error {
	logger.Debugf("transaction [%s], append fabtoken output [%s,%d,%v]", txID, index, view.Identity(tok.Owner.Raw), len(infoRaw))
	amount := uint64(999) // TODO
	tr := driver.TokenRecord{
		Namespace: ns,
		TxID:      txID,
		Index:     index,
		OwnerRaw:  tok.Owner.Raw,
		Type:      tok.Type,
		Quantity:  tok.Quantity,
		Amount:    amount,
		InfoRaw:   string(infoRaw),
		// TODO: TxStatus
	}
	err := cts.db.StoreOwnerToken(tr, ids)
	if err != nil {
		logger.Critical(err)
		return err
	}
	for _, id := range ids {
		// notify others
		logger.Debugf("post new event!")
		cts.Notify(ndriver.AddToken, cts.tmsID, id, tok.Type, txID, index)
	}

	return nil
}

func (cts *TokenStore) StoreIssuedHistoryToken(ns string, txID string, index uint64, tok *token2.Token, rws ndriver.RWSetProcessor, infoRaw []byte, issuer view.Identity, precision uint64) error {
	q, err := token2.ToQuantity(tok.Quantity, precision)
	if err != nil {
		return errors.Wrapf(err, "invalid quantity [%s]", tok.Quantity)
	}
	a, err := strconv.ParseUint(q.Decimal(), 10, int(precision))
	if err != nil {
		return errors.New("can't parse quantity to uint64")
	}

	tr := driver.TokenRecord{
		Namespace: ns,
		TxID:      txID,
		Index:     index,
		OwnerRaw:  tok.Owner.Raw,
		IssuerRaw: issuer,
		Type:      tok.Type,
		Quantity:  tok.Quantity,
		Amount:    a,
		InfoRaw:   string(infoRaw),
		// TODO: TxStatus
	}

	logger.Debugf("transaction [%s], append issued history token [%s,%s][%s,%d]",
		txID, tok.Type, a, index, len(infoRaw),
	)

	err = cts.db.StoreIssuedToken(tr)
	if err != nil {
		logger.Critical(err)
		return err
	}
	return nil
}

func (cts *TokenStore) StoreAuditToken(ns string, txID string, index uint64, tok *token2.Token, rws ndriver.RWSetProcessor, infoRaw []byte) error {
	tr := driver.TokenRecord{
		Namespace: ns,
		TxID:      txID,
		Index:     index,
		OwnerRaw:  tok.Owner.Raw,
		Type:      tok.Type,
		Quantity:  tok.Quantity,
		Amount:    999, // TODO
		InfoRaw:   string(infoRaw),
		// TODO: TxStatus
	}
	err := cts.db.StoreAuditToken(tr)
	if err != nil {
		logger.Critical(err)
		return err
	}
	return nil
}

func (cts *TokenStore) Notify(topic string, tmsID token.TMSID, walletID, tokenType, txID string, index uint64) {
	if cts.notifier == nil {
		logger.Warnf("cannot notify others!")
		return
	}

	e := ndriver.NewTokenProcessorEvent(topic, &ndriver.TokenMessage{
		TMSID:     tmsID,
		WalletID:  walletID,
		TokenType: tokenType,
		TxID:      txID,
		Index:     index,
	})

	logger.Debugf("Publish new event %v", e)
	cts.notifier.Publish(e)
}
