/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db_test

import (
	"testing"

	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/wallet/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/stretchr/testify/assert"
)

func TestGetWallet(t *testing.T) {
	cp := &mock.ConfigProvider{}
	cp.IsSetReturns(false)
	kvsStorage, err := kvs2.NewInMemoryKVS()
	assert.NoError(t, err)

	alice := driver.Identity("alice")
	meta := "meta"
	wr := db.NewWalletRegistry(
		&logging.MockLogger{},
		&fakeRole{},
		kvs2.NewWalletDB(kvsStorage, token.TMSID{Network: "testnetwork", Channel: "testchannel", Namespace: "tns"}),
	)
	assert.NoError(t, wr.RegisterWallet("hello", nil))
	assert.NoError(t, wr.BindIdentity(alice, "alice", "hello", meta))
	wID, err := wr.GetWalletID(alice)
	assert.NoError(t, err)
	assert.Equal(t, "hello", wID)
	var meta2 string
	assert.NoError(t, wr.GetIdentityMetadata(alice, "hello", &meta2))
	assert.Equal(t, meta, meta2)
}

type fakeRole struct{}

func (f *fakeRole) ID() idriver.IdentityRoleType {
	return 0
}

func (f *fakeRole) MapToIdentity(v driver.WalletLookupID) (driver.Identity, string, error) {
	// TODO implement me
	panic("implement me")
}

func (f *fakeRole) GetIdentityInfo(id string) (idriver.IdentityInfo, error) {
	// TODO implement me
	panic("implement me")
}

func (f *fakeRole) RegisterIdentity(config driver.IdentityConfiguration) error {
	// TODO implement me
	panic("implement me")
}

func (f *fakeRole) IdentityIDs() ([]string, error) {
	// TODO implement me
	panic("implement me")
}

func (f *fakeRole) Load(pp driver.PublicParameters) error {
	// TODO implement me
	panic("implement me")
}
