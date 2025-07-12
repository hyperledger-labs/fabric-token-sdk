/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbtest

import (
	"context"
	"testing"
	"time"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	assert2 "github.com/stretchr/testify/assert"
	"github.com/test-go/testify/assert"
)

type TestTokenDB interface {
	driver.TokenStore

	StoreToken(ctx context.Context, tr driver.TokenRecord, owners []string) error
	GetAllTokenInfos(ctx context.Context, ids []*token.ID) ([][]byte, error)
}

const (
	TST = token.Type("TST")
	ABC = token.Type("ABC")
)

var TokenNotifierCases = []struct {
	Name string
	Fn   func(*testing.T, TestTokenDB, driver.TokenNotifier)
}{
	{"SubscribeStore", TSubscribeStore},
	{"SubscribeStoreDelete", TSubscribeStoreDelete},
	{"SubscribeStoreNoCommit", TSubscribeStoreNoCommit},
	{"SubscribeRead", TSubscribeRead},
}

type dbEvent struct {
	op   driver2.Operation
	vals map[driver2.ColumnKey]string
}

func collectDBEvents(db driver.TokenNotifier) (*[]dbEvent, error) {
	ch := make(chan dbEvent)
	err := db.Subscribe(func(operation driver2.Operation, m map[driver2.ColumnKey]string) {
		ch <- dbEvent{op: operation, vals: m}
	})
	if err != nil {
		return nil, err
	}
	result := make([]dbEvent, 0, 1)
	go func() {
		for e := range ch {
			result = append(result, e)
		}
	}()
	return &result, nil
}

func TSubscribeStore(t *testing.T, db TestTokenDB, notifier driver.TokenNotifier) {
	result, err := collectDBEvents(notifier)
	assert.Nil(t, err)
	tx, err := db.NewTokenDBTransaction()
	assert.NoError(t, err)
	assert.NoError(t, tx.StoreToken(context.TODO(), driver.TokenRecord{TxID: "tx1", Index: 0}, []string{"alice"}))
	assert.NoError(t, tx.StoreToken(context.TODO(), driver.TokenRecord{TxID: "tx1", Index: 1}, []string{"alice"}))
	assert.NoError(t, tx.Commit())

	assert2.Eventually(t, func() bool { return len(*result) == 2 }, time.Second, 20*time.Millisecond)
}

func TSubscribeStoreDelete(t *testing.T, db TestTokenDB, notifier driver.TokenNotifier) {
	result, err := collectDBEvents(notifier)
	assert.Nil(t, err)
	tx, err := db.NewTokenDBTransaction()
	assert.NoError(t, err)
	assert.NoError(t, tx.StoreToken(context.TODO(), driver.TokenRecord{TxID: "tx1", Index: 0}, []string{"alice"}))
	assert.NoError(t, tx.StoreToken(context.TODO(), driver.TokenRecord{TxID: "tx1", Index: 1}, []string{"alice"}))
	assert.NoError(t, tx.Delete(context.TODO(), token.ID{TxId: "tx1", Index: 1}, "alice"))
	assert.NoError(t, tx.Commit())

	assert2.Eventually(t, func() bool { return len(*result) == 3 }, time.Second, 20*time.Millisecond)
}

func TSubscribeStoreNoCommit(t *testing.T, db TestTokenDB, notifier driver.TokenNotifier) {
	result, err := collectDBEvents(notifier)
	assert.Nil(t, err)
	tx, err := db.NewTokenDBTransaction()
	assert.NoError(t, err)
	assert.NoError(t, tx.StoreToken(context.TODO(), driver.TokenRecord{TxID: "tx1", Index: 0}, []string{"alice"}))
	assert.NoError(t, tx.StoreToken(context.TODO(), driver.TokenRecord{TxID: "tx1", Index: 1}, []string{"alice"}))

	assert2.Eventually(t, func() bool { return len(*result) == 0 }, time.Second, 20*time.Millisecond)
}

func TSubscribeRead(t *testing.T, db TestTokenDB, notifier driver.TokenNotifier) {
	result, err := collectDBEvents(notifier)
	assert.Nil(t, err)
	tx, err := db.NewTokenDBTransaction()
	assert.NoError(t, err)
	// assert.NoError(t, tx.StoreToken(context.TODO(), driver.TokenRecord{TxID: "tx1", Index: 0}, []string{"alice"}))
	_, _, err = tx.GetToken(context.Background(), token.ID{TxId: "tx1"}, true)
	assert.NoError(t, err)
	assert.NoError(t, tx.Commit())

	assert2.Eventually(t, func() bool { return len(*result) == 0 }, time.Second, 20*time.Millisecond)
}
