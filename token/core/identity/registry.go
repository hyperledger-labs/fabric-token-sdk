/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"fmt"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

type KVS interface {
	Exists(id string) bool
	Put(id string, state interface{}) error
	Get(id string, state interface{}) error
}

type WalletEntry struct {
	Prefix string
	Wallet driver.Wallet
}

type WalletsRegistry struct {
	Channel          string
	Namespace        string
	IdentityProvider driver.IdentityProvider
	IdentityRole     driver.IdentityRole
	KVS              KVS

	WLock   sync.RWMutex
	Wallets map[string]*WalletEntry
}

func NewWalletsRegistry(channel string, namespace string, identityProvider driver.IdentityProvider, identityRole driver.IdentityRole, KVS KVS) *WalletsRegistry {
	return &WalletsRegistry{
		Channel:          channel,
		Namespace:        namespace,
		IdentityProvider: identityProvider,
		IdentityRole:     identityRole,
		KVS:              KVS,
		Wallets:          map[string]*WalletEntry{},
	}
}

func (r *WalletsRegistry) Lookup(id interface{}) (driver.Wallet, driver.IdentityInfo, string, error) {
	identity, walletID, err := r.IdentityProvider.LookupIdentifier(r.IdentityRole, id)
	if err != nil {
		return nil, nil, "", errors.WithMessagef(err, "failed to lookup wallet [%s]", id)
	}
	wID := r.walletID(walletID)
	walletEntry, ok := r.Wallets[wID]
	if ok {
		return walletEntry.Wallet, nil, wID, nil
	}
	logger.Debugf("no wallet found for [%s] at [%s]", identity, wID)
	if len(identity) != 0 {
		identityWID, err := r.GetWallet(identity)
		logger.Debugf("wallet for identity [%s] -> [%s:%s]", identity, identityWID, err)
		if err == nil && len(identityWID) != 0 {
			w, ok := r.Wallets[identityWID]
			if ok {
				logger.Debugf("found wallet [%s:%s:%s:%s]", identity, walletID, w.Wallet.ID(), identityWID)
				return w.Wallet, nil, identityWID, nil
			}
		} /*else {
			// brute force search as last resort
			logger.Debugf("brute force search of the wallet id for [%s]", identity)
			for _, w := range r.Wallets {
				if w.Wallet.Contains(identity) {
					logger.Debugf("found wallet [%s:%s:%s]", identity, walletID, w.Wallet.ID())
					return w.Wallet, nil, wID, nil
				}
			}

			logger.Errorf("failed to lookup wallet for identity [%s]: [%s]", identity, err)
		}*/
	}

	idInfo, err := r.IdentityProvider.GetIdentityInfo(r.IdentityRole, walletID)
	if err != nil {
		return nil, nil, wID, errors.WithMessagef(err, "failed to get wwallet info for [%s]", walletID)
	}

	return nil, idInfo, wID, nil
}

func (r *WalletsRegistry) walletID(id string) string {
	return r.Channel + r.Namespace + id
}

func (r *WalletsRegistry) Register(id string, w driver.Wallet) {
	r.Wallets[id] = &WalletEntry{
		Prefix: fmt.Sprintf("%s:%s:%s", r.Channel, r.Namespace, id),
		Wallet: w,
	}
}

func (r *WalletsRegistry) ExistsRecipientIdentity(identity view.Identity, wID string) bool {
	return r.KVS.Exists(r.Wallets[wID].Prefix + identity.Hash())
}

func (r *WalletsRegistry) PutRecipientIdentity(identity view.Identity, wID string) error {
	logger.Debugf("put recipient identity [%s]->[%s]", identity, wID)
	idHash := identity.Hash()
	if err := r.KVS.Put(idHash, wID); err != nil {
		return err
	}
	if err := r.KVS.Put(r.Wallets[wID].Prefix+idHash, wID); err != nil {
		return err
	}
	return nil
}

func (r *WalletsRegistry) GetWallet(identity view.Identity) (string, error) {
	var wID string
	if err := r.KVS.Get(identity.Hash(), &wID); err != nil {
		return "", err
	}
	logger.Debugf("wallet [%s] is bound to identity [%s]", wID, identity)
	return wID, nil
}

func (r *WalletsRegistry) Lock() {
	r.WLock.Lock()
}

func (r *WalletsRegistry) Unlock() {
	r.WLock.Unlock()
}
