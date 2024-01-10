/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rws

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/cache/secondcache"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/certification"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/query"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Vault struct {
	vault                driver.Vault
	queryEngine          *query.Engine
	certificationStorage *certification.Storage
}

func NewVault(sp view.ServiceProvider, channel string, namespace string, vault driver.Vault) *Vault {
	return &Vault{
		vault:                vault,
		queryEngine:          query.NewEngine(vault, namespace, secondcache.New(20000)),
		certificationStorage: certification.NewStorage(sp, channel, namespace),
	}
}

func (v *Vault) QueryEngine() vault.QueryEngine {
	return v.queryEngine
}

func (v *Vault) CertificationStorage() vault.CertificationStorage {
	return v.certificationStorage
}

func (v *Vault) DeleteTokens(ns string, ids ...*token.ID) error {
	return v.vault.DeleteTokens(ns, ids...)
}
