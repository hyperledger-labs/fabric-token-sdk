/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttxdb

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/certification"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var logger = flogging.MustGetLogger("vault.ttxdb")

type Vault struct {
	db                   *ttxdb.DB
	queryEngine          vault.QueryEngine
	certificationStorage *certification.Storage
}

func NewVault(sp view.ServiceProvider, channel string, namespace string) *Vault {
	db := ttxdb.Get(sp, fmt.Sprintf("%s,%s", channel, namespace), "tok") // TODO: cheating to match CommonTokenStore
	if db == nil {
		// logger.Critical("cannot get database") // TODO
		return nil
	}
	engine := NewEngine(namespace, db) // TODO: no cache?

	return &Vault{
		db:                   db,
		queryEngine:          engine,
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
	return v.db.DeleteTokens(ns, ids...)
}
