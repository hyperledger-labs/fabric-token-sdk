/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rws

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/cache/secondcache"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/certification"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/query"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Vault struct {
	vault                driver.Vault
	queryEngine          *query.Engine
	certificationStorage certification.Storage
}

func NewVault(cs certification.Storage, tmsID token2.TMSID, vault driver.Vault) (*Vault, error) {
	return &Vault{
		vault:                vault,
		queryEngine:          query.NewEngine(vault, tmsID.Namespace, secondcache.New(20000)),
		certificationStorage: cs,
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
