/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snippets

import (
	vault "github.com/hashicorp/vault/api"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs/hashicorp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
	"github.com/pkg/errors"
)

type MixedStorageProvider struct {
	kvs     kvs.KVS
	manager *identitydb.Manager
}

func NewMixedStorageProvider(client *vault.Client, prefix string, manager *identitydb.Manager) (*MixedStorageProvider, error) {
	kvs, err := hashicorp.NewWithClient(client, prefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating hashicorp.NewWithClient")
	}
	return &MixedStorageProvider{kvs: kvs, manager: manager}, nil
}

func (s *MixedStorageProvider) WalletDB(tmsID token.TMSID) (identity.WalletDB, error) {
	return s.manager.WalletDBByTMSId(tmsID)
}

func (s *MixedStorageProvider) IdentityDB(tmsID token.TMSID) (identity.IdentityDB, error) {
	return kvs.NewIdentityDB(s.kvs, tmsID), nil
}

func (s *MixedStorageProvider) Keystore() (identity.Keystore, error) {
	return s.kvs, nil
}
