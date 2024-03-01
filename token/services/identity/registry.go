/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"fmt"
	"strings"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	db "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type WalletsRegistry struct {
	ID               token.TMSID
	IdentityProvider driver.IdentityProvider
	IdentityRole     driver.IdentityRole
	Storage          db.WalletDB

	sync.RWMutex
	Wallets map[string]driver.Wallet
}

// NewWalletsRegistry returns a new wallets registry for the passed parameters
func NewWalletsRegistry(identityProvider driver.IdentityProvider, identityRole driver.IdentityRole, storage db.WalletDB) *WalletsRegistry {
	return &WalletsRegistry{
		IdentityProvider: identityProvider,
		IdentityRole:     identityRole,
		Storage:          storage,
		Wallets:          map[string]driver.Wallet{},
	}
}

// Lookup searches the wallet corresponding to the passed id.
// If a wallet is found, Lookup returns the wallet and its identifier.
// If no wallet is found, Lookup returns the identity info and a potential wallet identifier for the passed id.
// The identity info can be nil meaning that nothing has been found bound to the passed identifier
func (r *WalletsRegistry) Lookup(id interface{}) (driver.Wallet, driver.IdentityInfo, string, error) {
	var walletIdentifiers []string

	identity, walletID, err := r.IdentityProvider.LookupIdentifier(r.IdentityRole, id)
	if err != nil {
		fail := true
		// give it a second change
		passedIdentity, ok := id.(view.Identity)
		if ok {
			logger.Debugf("lookup failed, check if there is a wallet for identity [%s]", passedIdentity)
			// is this identity registered
			wID, err := r.GetWalletID(passedIdentity)
			if err == nil && len(wID) != 0 {
				logger.Debugf("lookup failed, there is a wallet for identity [%s]: [%s]", passedIdentity, wID)
				// we got a hit
				walletID = wID
				identity = passedIdentity
				fail = false
			}
		}
		if fail {
			return nil, nil, "", errors.WithMessagef(err, "failed to lookup wallet [%s]", id)
		}
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("looked-up identifier [%s:%s]", identity, walletIDToString(walletID))
	}
	wID := walletID
	walletEntry, ok := r.Wallets[wID]
	if ok {
		return walletEntry, nil, wID, nil
	}
	walletIdentifiers = append(walletIdentifiers, wID)

	// give it a second chance
	passedIdentity, ok := id.(view.Identity)
	if ok {
		logger.Debugf("no wallet found, check if there is a wallet for identity [%s]", passedIdentity)
		// is this identity registered
		passedWalletID, err := r.GetWalletID(passedIdentity)
		if err == nil && len(passedWalletID) != 0 {
			logger.Debugf("no wallet found, there is a wallet for identity [%s]: [%s]", passedIdentity, passedWalletID)
			// we got a hit
			walletEntry, ok = r.Wallets[passedWalletID]
			if ok {
				return walletEntry, nil, passedWalletID, nil
			}
			logger.Debugf("no wallet found, there is a wallet for identity [%s]: [%s] but it has not been recreated yet", passedIdentity, passedWalletID)
		}
		walletIdentifiers = append(walletIdentifiers, passedWalletID)
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("no wallet found for [%s] at [%s]", identity, walletIDToString(wID))
	}
	if len(identity) != 0 {
		identityWID, err := r.GetWalletID(identity)
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("wallet for identity [%s] -> [%s:%s]", identity, identityWID, err)
		}
		if err == nil && len(identityWID) != 0 {
			w, ok := r.Wallets[identityWID]
			if ok {
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("found wallet [%s:%s:%s:%s]", identity, walletID, w.ID(), identityWID)
				}
				return w, nil, identityWID, nil
			}
		}
		walletIdentifiers = append(walletIdentifiers, identityWID)
	}

	for _, id := range walletIdentifiers {
		if len(id) == 0 {
			continue
		}
		// give it a second chance
		var idInfo driver.IdentityInfo
		idInfo, err = r.IdentityProvider.GetIdentityInfo(r.IdentityRole, id)
		if err == nil {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("identity info found at [%s]", walletIDToString(id))
			}
			return nil, idInfo, id, nil
		} else {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("identity info not found at [%s]", walletIDToString(id))
			}
		}
	}
	return nil, nil, "", errors.Errorf("failed to get wallet info for [%s:%v]", walletIDToString(walletID), walletIdentifiers)
}

// RegisterWallet binds the passed wallet to the passed id
func (r *WalletsRegistry) RegisterWallet(id string, w driver.Wallet) error {
	if err := r.Storage.StoreWalletID(id); err != nil {
		return errors.WithMessagef(err, "failed to store wallet entry [%s]", id)
	}
	r.Wallets[id] = w
	return nil
}

// RegisterIdentity binds the passed identity to the passed wallet identifier.
// Additional metadata can be bound to the identity.
func (r *WalletsRegistry) RegisterIdentity(identity view.Identity, wID string, meta any) error {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("put recipient identity [%s]->[%s]", identity, wID)
	}
	return r.Storage.StoreIdentity(identity, wID, meta)
}

// GetIdentityMetadata loads metadata bound to the passed identity into the passed meta argument
func (r *WalletsRegistry) GetIdentityMetadata(identity view.Identity, wID string, meta any) error {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get recipient identity metadata [%s]->[%s]", identity, wID)
	}
	if err := r.Storage.LoadMeta(identity, meta); err != nil {
		return errors.WithMessagef(err, "failed to retrieve identity's metadata [%s]", identity)
	}
	return nil
}

// GetWalletID returns the wallet identifier bound to the passed identity
func (r *WalletsRegistry) GetWalletID(identity view.Identity) (string, error) {
	wID, err := r.Storage.GetWalletID(identity)
	if err != nil {
		return "", nil
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("wallet [%s] is bound to identity [%s]", wID, identity)
	}
	return wID, nil
}

// ContainsIdentity returns true if the passed identity belongs to the passed wallet,
// false otherwise
func (r *WalletsRegistry) ContainsIdentity(identity view.Identity, wID string) bool {
	return r.Storage.IdentityExists(identity, wID)
}

// WalletIDs returns the list of wallet identifiers
func (r *WalletsRegistry) WalletIDs() ([]string, error) {
	walletIDs, err := r.IdentityProvider.WalletIDs(r.IdentityRole)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get wallet identifiers from identity provider")
	}
	duplicates := map[string]bool{}
	for _, id := range walletIDs {
		duplicates[id] = true
	}

	ids, err := r.Storage.GetWalletIDs()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get wallets iterator")
	}
	for _, wID := range ids {
		_, found := duplicates[wID]
		if !found {
			walletIDs = append(walletIDs, wID)
			duplicates[wID] = true
		}
	}
	return walletIDs, nil
}

func walletIDToString(w string) string {
	if len(w) <= 20 {
		return strings.ToValidUTF8(w, "X")
	}

	return fmt.Sprintf("%s~%s", strings.ToValidUTF8(w[:20], "X"), hash.Hashable(w).String())
}
