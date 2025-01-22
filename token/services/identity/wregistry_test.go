/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity_test

import (
	"testing"

	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	mem "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/kvs"
	"github.com/stretchr/testify/assert"
)

func TestGetWallet(t *testing.T) {
	cp := &mock.ConfigProvider{}
	cp.IsSetReturns(false)
	kvsStorage, err := kvs.NewWithConfig(&mem.Driver{}, "_default", cp)
	assert.NoError(t, err)

	alice := driver.Identity("alice")
	meta := "meta"
	wr := identity.NewWalletRegistry(
		nil,
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

func (f *fakeRole) ID() driver.IdentityRoleType {
	return 0
}

func (f *fakeRole) MapToIdentity(v driver.WalletLookupID) (driver.Identity, string, error) {
	// TODO implement me
	panic("implement me")
}

func (f *fakeRole) GetIdentityInfo(id string) (driver.IdentityInfo, error) {
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
