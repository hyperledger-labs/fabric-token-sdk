/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vault

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/cache/secondcache"

	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/certification"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/query"
)

type Vault struct {
	sp                   view.ServiceProvider
	queryEngine          *query.Engine
	certificationStorage *certification.Storage
}

func New(sp view.ServiceProvider, channel string, namespace string, vault driver.Vault) *Vault {
	return &Vault{
		queryEngine: query.NewEngine(
			vault,
			namespace,
			secondcache.New(20000),
		),
		certificationStorage: certification.NewStorage(sp, channel, namespace),
	}
}

func (v *Vault) QueryEngine() driver2.QueryEngine {
	return v.queryEngine
}

func (v *Vault) CertificationStorage() *certification.Storage {
	return v.certificationStorage
}
