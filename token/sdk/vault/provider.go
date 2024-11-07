/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vault

import (
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	vault2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/pkg/errors"
)

type DBProvider[T any] interface {
	DBByTMSId(id token.TMSID) (T, error)
}

type Provider struct {
	tokenDBProvider tokens.DBProvider
	ttxDBProvider   ttx.DBProvider
	auditDBProvider auditor.AuditDBProvider

	vaultCacheLock sync.RWMutex
	vaultCache     map[string]driver.TokenVault
}

func NewVaultProvider(tokenDBProvider tokens.DBProvider, ttxDBProvider ttx.DBProvider, auditDBProvider auditor.AuditDBProvider) *Provider {
	return &Provider{
		ttxDBProvider:   ttxDBProvider,
		tokenDBProvider: tokenDBProvider,
		auditDBProvider: auditDBProvider,
		vaultCache:      make(map[string]driver.TokenVault),
	}
}

func (v *Provider) Vault(network string, channel string, namespace string) (driver.TokenVault, error) {
	k := network + channel + namespace
	// Check cache
	v.vaultCacheLock.RLock()
	res, ok := v.vaultCache[k]
	v.vaultCacheLock.RUnlock()
	if ok {
		return res, nil
	}

	// lock
	v.vaultCacheLock.Lock()
	defer v.vaultCacheLock.Unlock()

	// check cache again
	res, ok = v.vaultCache[k]
	if ok {
		return res, nil
	}

	tmsID := token.TMSID{
		Network:   network,
		Channel:   channel,
		Namespace: namespace,
	}
	tokenDB, err := v.tokenDBProvider.DBByTMSId(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get token db")
	}
	ttxDB, err := v.ttxDBProvider.DBByTMSId(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get ttx db")
	}
	auditDB, err := v.auditDBProvider.DBByTMSId(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get audit db")
	}

	// Create new vault
	res, err = vault2.NewVault(tmsID, auditDB, ttxDB, tokenDB)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create new vault")
	}

	// update cache
	v.vaultCache[k] = res

	return res, nil
}
