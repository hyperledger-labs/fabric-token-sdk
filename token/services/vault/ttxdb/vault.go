/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttxdb

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/certification"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("vault.ttxdb")

type Vault struct {
	vault                driver.Vault
	db                   *ttxdb.DB
	queryEngine          vault.QueryEngine
	certificationStorage certification.Storage
}

func NewVault(sp view.ServiceProvider, tmsID token2.TMSID, vault driver.Vault) (*Vault, error) {
	walletID := fmt.Sprintf("%s-%s-%s", tmsID.Network, tmsID.Channel, tmsID.Namespace)
	db := ttxdb.Get(sp, tmsID.String(), walletID)
	if db == nil {
		return nil, errors.New("cannot get database")
	}
	engine := NewEngine(tmsID.Namespace, db) // TODO: no cache?

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
		queryEngine:          engine,
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
	return v.db.DeleteTokens(ns, ids...)
}
