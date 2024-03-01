/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"runtime/debug"

	"github.com/hashicorp/go-uuid"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenStore interface {
	DeleteToken(deletedBy string, ids ...*token.ID) error
}

type Vault struct {
	ons        *orion.NetworkService
	tokenStore TokenStore
}

func NewVault(ons *orion.NetworkService, tokenStore TokenStore) *Vault {
	return &Vault{ons: ons, tokenStore: tokenStore}
}

func (v *Vault) DeleteTokens(ids ...*token.ID) error {
	// prepare a rws with deletes
	id, err := uuid.GenerateUUID()
	if err != nil {
		return errors.Wrapf(err, "failed to generated uuid")
	}
	txID := "delete_" + id
	rws, err := v.ons.Vault().NewRWSet(txID)
	if err != nil {
		return err
	}
	defer rws.Done()
	if err := v.tokenStore.DeleteToken(string(debug.Stack()), ids...); err != nil {
		return errors.Wrapf(err, "failed to delete tokens")
	}
	rws.Done()

	if err := v.ons.Vault().CommitTX(txID, 0, 0); err != nil {
		return errors.WithMessagef(err, "failed to commit rws with token delitions")
	}

	return nil
}

func (v *Vault) TransactionStatus(txID string) (driver.ValidationCode, error) {
	vc, err := v.ons.Vault().Status(txID)
	return driver.ValidationCode(vc), err
}

type Executor struct {
	qe *orion.QueryExecutor
}

func (e *Executor) Done() {
	e.qe.Done()
}

func (e *Executor) GetState(namespace string, key string) ([]byte, error) {
	return e.qe.GetState(namespace, key)
}
