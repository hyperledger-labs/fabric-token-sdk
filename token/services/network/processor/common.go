/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package processor

import (
	"fmt"
	"strconv"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.network.processor")

const (
	IDs = "ids"
)

type GetStateOpt int

type RWSet interface {
	SetState(namespace string, key string, value []byte) error
	GetState(namespace string, key string) ([]byte, error)
	GetStateMetadata(namespace, key string) (map[string][]byte, error)
	DeleteState(namespace string, key string) error
	SetStateMetadata(namespace, key string, metadata map[string][]byte) error
}

type TokenStore interface {
	// DeleteFabToken adds to the passed rws the deletion of the passed token
	DeleteFabToken(ns string, txID string, index uint64, rws RWSet, deletedBy string) error
	StoreFabToken(ns string, txID string, index uint64, tok *token2.Token, rws RWSet, infoRaw []byte, ids []string) error
	StoreIssuedHistoryToken(ns string, txID string, index uint64, tok *token2.Token, rws RWSet, infoRaw []byte, issuer view.Identity, precision uint64) error
	StoreAuditToken(ns string, txID string, index uint64, tok *token2.Token, rws RWSet, infoRaw []byte) error
}

type CommonTokenStore struct {
	notifier events.Publisher
	tmsID    token.TMSID
	db       driver.TokenDB
}

func NewCommonTokenStore(sp view2.ServiceProvider, tmsID token.TMSID) (*CommonTokenStore, error) {
	walletID := fmt.Sprintf("%s-%s-%s", tmsID.Network, tmsID.Channel, tmsID.Namespace)
	db := ttxdb.Get(sp, tmsID.String(), walletID)
	if db == nil {
		return nil, errors.New("cannot get database")
	}

	notifier, err := events.GetPublisher(sp)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get event publisher")
	}

	return &CommonTokenStore{notifier: notifier, tmsID: tmsID, db: db}, nil
}

func (cts *CommonTokenStore) DeleteFabToken(ns string, txID string, index uint64, rws RWSet, deletedBy string) error {
	logger.Debugf("spend token [%s,%d]", txID, index)
	err := cts.db.Delete(ns, txID, index, deletedBy)
	if err != nil {
		logger.Error(err)
	}
	return err
}

func (cts *CommonTokenStore) StoreFabToken(ns string, txID string, index uint64, tok *token2.Token, rws RWSet, infoRaw []byte, ids []string) error {
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
		cts.Notify(AddToken, cts.tmsID, id, tok.Type, txID, index)
	}

	return nil
}

func (cts *CommonTokenStore) StoreIssuedHistoryToken(ns string, txID string, index uint64, tok *token2.Token, rws RWSet, infoRaw []byte, issuer view.Identity, precision uint64) error {
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

func (cts *CommonTokenStore) StoreAuditToken(ns string, txID string, index uint64, tok *token2.Token, rws RWSet, infoRaw []byte) error {
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
