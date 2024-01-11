/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	kvs2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/kvs"
)

type StorageProvider struct {
	sp view.ServiceProvider
}

func NewStorageProvider(sp view.ServiceProvider) *StorageProvider {
	return &StorageProvider{sp: sp}
}

func (s *StorageProvider) New(tmsID token.TMSID) (identity.Storage, error) {
	return kvs.NewIdentityStorage(kvs2.GetService(s.sp), tmsID), nil
}
