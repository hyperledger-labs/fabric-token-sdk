/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package vault

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/api"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/certification"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/query"
)

type Channel interface {
	Name() string
	Vault() *fabric.Vault
}

type Vault struct {
	sp                   view.ServiceProvider
	queryEngine          *query.Engine
	certificationStorage *certification.Storage
}

func NewVault(sp view.ServiceProvider, channel Channel, namespace string) *Vault {
	return &Vault{
		queryEngine:          query.NewEngine(channel, namespace),
		certificationStorage: certification.NewStorage(sp, channel, namespace),
	}
}

func (v *Vault) QueryEngine() api.QueryEngine {
	return v.queryEngine
}

func (v *Vault) CertificationStorage() *certification.Storage {
	return v.certificationStorage
}
