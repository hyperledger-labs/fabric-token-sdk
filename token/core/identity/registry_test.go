/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"testing"

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
	kvstore, err := kvs.NewWithConfig(registry, "memory", "_default", cp)
	assert.NoError(t, err)

	alice := view.Identity("alice")
	meta := "meta"
	wr := NewWalletsRegistry(token.TMSID{Network: "testnetwork", Channel: "testchannel", Namespace: "tns"}, nil, driver.OwnerRole, kvstore)
	wr.RegisterWallet("hello", nil)
	assert.NoError(t, wr.RegisterIdentity(alice, "hello", meta))
	wID, err := wr.GetWallet(alice)
	assert.NoError(t, err)
	assert.Equal(t, "hello", wID)
	var meta2 string
	assert.NoError(t, wr.GetIdentityMetadata(alice, "", &meta2))
	assert.Equal(t, meta, meta2)
}
