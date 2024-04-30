/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/certification"
	vaultdb "github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/db"
	"github.com/pkg/errors"
)

type VaultProvider struct {
	tokenDBProvider tokens.DBProvider
	ttxDBProvider   ttx.DBProvider
	tokenProvider   ttx.TokensProvider
	storageProvider certification.StorageProvider
	fabricNSP       *fabric.NetworkServiceProvider
	orionNSP        *orion.NetworkServiceProvider

	vaultCacheLock sync.RWMutex
	vaultCache     map[string]vault.TokenVault
}

func NewVaultProvider(tokenDBProvider tokens.DBProvider, ttxDBProvider ttx.DBProvider, tokenProvider ttx.TokensProvider, storageProvider certification.StorageProvider, fabricNSP *fabric.NetworkServiceProvider, orionNSP *orion.NetworkServiceProvider) *VaultProvider {
	return &VaultProvider{
		tokenDBProvider: tokenDBProvider,
		ttxDBProvider:   ttxDBProvider,
		tokenProvider:   tokenProvider,
		storageProvider: storageProvider,
		fabricNSP:       fabricNSP,
		orionNSP:        orionNSP,
		vaultCache:      make(map[string]vault.TokenVault),
	}
}

func (v *VaultProvider) Vault(network string, channel string, namespace string) (vault.TokenVault, error) {
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
		return nil, errors.WithMessagef(err, "failed to get token db")
	}
	tokens, err := v.tokenProvider.Tokens(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get token store")
	}

	// Create new vault
	if fns, err := v.fabricNetworkService(network); err == nil {
		ch, err := fns.Channel(channel)
		if err != nil {
			return nil, err
		}
		tmsID := token.TMSID{
			Network:   network,
			Channel:   ch.Name(),
			Namespace: namespace,
		}
		storage, err := v.storageProvider.NewStorage(tmsID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create new storage")
		}
		res, err = vaultdb.NewVault(tmsID, storage, ttxDB, tokenDB, fabric2.NewVault(ch, tokens))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create new vault")
		}
	} else {
		ons, err := v.orionNetworkService(network)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot find network [%s]", network)
		}
		tmsID := token.TMSID{
			Network:   network,
			Channel:   "",
			Namespace: namespace,
		}
		storage, err := v.storageProvider.NewStorage(tmsID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create new storage")
		}
		res, err = vaultdb.NewVault(tmsID, storage, ttxDB, tokenDB, orion2.NewVault(ons, tokens))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create new vault")
		}
	}
	// update cache
	v.vaultCache[k] = res

	return res, nil
}

func (p *VaultProvider) fabricNetworkService(id string) (*fabric.NetworkService, error) {
	if p.fabricNSP == nil {
		return nil, errors.New("fabric nsp not found")
	}
	return p.fabricNSP.FabricNetworkService(id)
}

func (p *VaultProvider) orionNetworkService(id string) (*orion.NetworkService, error) {
	if p.orionNSP == nil {
		return nil, errors.New("orion nsp not found")
	}
	return p.orionNSP.NetworkService(id)
}
