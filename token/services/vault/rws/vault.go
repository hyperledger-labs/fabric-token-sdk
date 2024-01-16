/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rws

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/cache/secondcache"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/certification"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/query"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type Vault struct {
	vault                driver.Vault
	queryEngine          *query.Engine
	certificationStorage certification.Storage
}

func NewVault(sp view.ServiceProvider, tmsID token2.TMSID, vault driver.Vault) (*Vault, error) {
	storageProvider, err := certification.GetStorageProvider(sp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get storage provider")
	}
	storage, err := storageProvider.NewStorage(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create new storage")
	}
	return &Vault{
		vault:                vault,
		queryEngine:          query.NewEngine(vault, tmsID.Namespace, secondcache.New(20000)),
		certificationStorage: storage,
	}, nil
}

func (v *Vault) QueryEngine() vault.QueryEngine {
	return v.queryEngine
}

func (v *Vault) CertificationStorage() certification.Storage {
	return v.certificationStorage
}

func (v *Vault) DeleteTokens(ns string, ids ...*token.ID) error {
	return v.vault.DeleteTokens(ns, ids...)
}
