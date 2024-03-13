/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"

	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs/mock"
	registry2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/kvs"
	"github.com/stretchr/testify/assert"
)

func TestGetWallet(t *testing.T) {
	cp := &mock.ConfigProvider{}
	cp.IsSetReturns(false)
	registry := registry2.New()
	kvsStorage, err := kvs.NewWithConfig(registry, "memory", "_default", cp)
	assert.NoError(t, err)

	alice := view.Identity("alice")
	meta := "meta"
	wr := identity.NewWalletRegistry(&fakeRole{}, kvs2.NewWalletDB(kvsStorage, token.TMSID{Network: "testnetwork", Channel: "testchannel", Namespace: "tns"}))
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

func (f *fakeRole) ID() driver.IdentityRole {
	return 0
}

func (f *fakeRole) MapToID(v interface{}) (view.Identity, string, error) {
	//TODO implement me
	panic("implement me")
}

func (f *fakeRole) GetIdentityInfo(id string) (driver.IdentityInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (f *fakeRole) RegisterIdentity(id string, path string) error {
	//TODO implement me
	panic("implement me")
}

func (f *fakeRole) IdentityIDs() ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (f *fakeRole) Reload(pp driver.PublicParameters) error {
	//TODO implement me
	panic("implement me")
}
