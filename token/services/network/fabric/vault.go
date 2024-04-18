/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"runtime/debug"

	"github.com/hashicorp/go-uuid"
	driver3 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenStore interface {
	DeleteToken(deletedBy string, ids ...*token.ID) error
}

type Vault struct {
	ch         *fabric.Channel
	tokenStore TokenStore
}

func NewVault(ch *fabric.Channel, tokenStore TokenStore) *Vault {
	return &Vault{
		ch:         ch,
		tokenStore: tokenStore,
	}
}

func (v *Vault) DeleteTokens(ids ...*token.ID) error {
	// prepare a rws with deletes
	id, err := uuid.GenerateUUID()
	if err != nil {
		return errors.Wrapf(err, "failed to generated uuid")
	}
	txID := "delete_" + id
	rws, err := v.ch.Vault().NewRWSet(txID)
	if err != nil {
		return err
	}
	defer rws.Done()
	if err := v.tokenStore.DeleteToken(string(debug.Stack()), ids...); err != nil {
		return errors.Wrapf(err, "failed to delete tokens")
	}
	rws.Done()

	if err := v.ch.Vault().CommitTX(txID, 0, 0); err != nil {
		return errors.WithMessagef(err, "failed to commit rws with token delitions")
	}

	return nil
}

func (v *Vault) TransactionStatus(txID string) (driver2.ValidationCode, string, error) {
	vc, message, err := v.ch.Vault().Status(txID)
	return vc, message, err
}

type Executor struct {
	qe driver3.QueryExecutor
}

func (e *Executor) Done() {
	e.qe.Done()
}

func (e *Executor) GetState(namespace string, key string) ([]byte, error) {
	return e.qe.GetState(namespace, key)
}
