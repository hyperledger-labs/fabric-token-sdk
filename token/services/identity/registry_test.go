/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/kvs"

	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs/mock"
	registry2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
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
	wr := identity.NewWalletsRegistry(nil, driver.OwnerRole, kvs2.NewIdentityStorage(kvsStorage, token.TMSID{Network: "testnetwork", Channel: "testchannel", Namespace: "tns"}))
	assert.NoError(t, wr.RegisterWallet("hello", nil))
	assert.NoError(t, wr.RegisterIdentity(alice, "hello", meta))
	wID, err := wr.GetWalletID(alice)
	assert.NoError(t, err)
	assert.Equal(t, "hello", wID)
	var meta2 string
	assert.NoError(t, wr.GetIdentityMetadata(alice, "", &meta2))
	assert.Equal(t, meta, meta2)
}
