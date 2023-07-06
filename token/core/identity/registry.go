/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"fmt"
	"strings"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type KVS interface {
	Exists(id string) bool
	Put(id string, state interface{}) error
	Get(id string, state interface{}) error
	GetByPartialCompositeID(prefix string, attrs []string) (kvs.Iterator, error)
}

type WalletEntry struct {
	ID     string
	Prefix string
	Wallet driver.Wallet `json:"-"`
}

type WalletsRegistry struct {
	ID               token.TMSID
	IdentityProvider driver.IdentityProvider
	IdentityRole     driver.IdentityRole
	KVS              KVS

	sync.RWMutex
	Wallets map[string]*WalletEntry
}

// NewWalletsRegistry returns a new wallets registry for the passed parameters
func NewWalletsRegistry(id token.TMSID, identityProvider driver.IdentityProvider, identityRole driver.IdentityRole, KVS KVS) *WalletsRegistry {
	return &WalletsRegistry{
		ID:               id,
		IdentityProvider: identityProvider,
		IdentityRole:     identityRole,
		KVS:              KVS,
		Wallets:          map[string]*WalletEntry{},
	}
}

// Lookup searches the wallet corresponding to the passed id.
// If a wallet is found, Lookup returns the wallet and its identifier.
// If no wallet is found, Lookup returns the identity info and a potential wallet identifier for the passed id.
// The identity info can be nil meaning that nothing has been found bound to the passed identifier
func (r *WalletsRegistry) Lookup(id interface{}) (driver.Wallet, driver.IdentityInfo, string, error) {
	identity, walletID, err := r.IdentityProvider.LookupIdentifier(r.IdentityRole, id)
	if err != nil {
		fail := true
		// give it a second change
		passedIdentity, ok := id.(view.Identity)
		if ok {
			logger.Debugf("lookup failed, check if there is a wallet for identity [%s]", passedIdentity)
			// is this identity registered
			wID, err := r.GetWallet(passedIdentity)
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
	logger.Debugf("looked-up identifier [%s:%s]", identity, walletIDToString(walletID))
	wID := walletID
	walletEntry, ok := r.Wallets[wID]
	if ok {
		return walletEntry.Wallet, nil, wID, nil
	}

	// give it a second chance
	passedIdentity, ok := id.(view.Identity)
	if ok {
		logger.Debugf("no wallet found, check if there is a wallet for identity [%s]", passedIdentity)
		// is this identity registered
		passedWalletID, err := r.GetWallet(passedIdentity)
		if err == nil && len(passedWalletID) != 0 {
			logger.Debugf("no wallet found, there is a wallet for identity [%s]: [%s]", passedIdentity, passedWalletID)
			// we got a hit
			walletEntry, ok = r.Wallets[passedWalletID]
			if ok {
				return walletEntry.Wallet, nil, passedWalletID, nil
			}
		}
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("no wallet found for [%s] at [%s]", identity, walletIDToString(wID))
	}
	var identityWID string
	if len(identity) != 0 {
		identityWID, err = r.GetWallet(identity)
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("wallet for identity [%s] -> [%s:%s]", identity, identityWID, err)
		}
		if err == nil && len(identityWID) != 0 {
			w, ok := r.Wallets[identityWID]
			if ok {
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("found wallet [%s:%s:%s:%s]", identity, walletID, w.Wallet.ID(), identityWID)
				}
				return w.Wallet, nil, identityWID, nil
			}
		}
	}

	for _, id := range []string{wID, identityWID} {
		if len(id) == 0 {
			continue
		}
		// give it a second chance
		var idInfo driver.IdentityInfo
		idInfo, err = r.IdentityProvider.GetIdentityInfo(r.IdentityRole, id)
		if err == nil {
			logger.Debugf("identity info found at [%s]", walletIDToString(id))
			return nil, idInfo, id, nil
		} else {
			logger.Debugf("identity info not found at [%s]", walletIDToString(id))
		}
	}
	return nil, nil, "", errors.Errorf("failed to get wallet info for [%s:%s]", walletIDToString(walletID), walletIDToString(identityWID))
}

// RegisterWallet binds the passed wallet to the passed id
func (r *WalletsRegistry) RegisterWallet(id string, w driver.Wallet) error {
	entry := &WalletEntry{
		ID:     id,
		Prefix: fmt.Sprintf("%s-%s-%s-%s", r.ID.Network, r.ID.Channel, r.ID.Namespace, id),
		Wallet: w,
	}
	if err := r.KVS.Put(kvs.CreateCompositeKeyOrPanic("wallets", []string{r.ID.Network, r.ID.Channel, r.ID.Namespace, id}), entry); err != nil {
		return errors.WithMessagef(err, "failed to store wallet entry [%s]", id)
	}
	r.Wallets[id] = entry
	return nil
}

// RegisterIdentity binds the passed identity to the passed wallet identifier.
// Additional metadata can be bound to the identity.
func (r *WalletsRegistry) RegisterIdentity(identity view.Identity, wID string, meta any) error {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("put recipient identity [%s]->[%s]", identity, wID)
	}
	idHash := identity.Hash()
	if err := r.KVS.Put(idHash, wID); err != nil {
		return errors.WithMessagef(err, "failed to store identity's wallet [%s]", identity)
	}
	if meta != nil {
		if err := r.KVS.Put("meta"+idHash, meta); err != nil {
			return errors.WithMessagef(err, "failed to store identity's metadata [%s]", identity)
		}
	}
	if err := r.KVS.Put(r.Wallets[wID].Prefix+idHash, wID); err != nil {
		return errors.WithMessagef(err, "failed to store identity's wallet reference[%s]", identity)
	}
	return nil
}

// GetIdentityMetadata loads metadata bound to the passed identity into the passed meta argument
func (r *WalletsRegistry) GetIdentityMetadata(identity view.Identity, wID string, meta any) error {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get recipient identity metadata [%s]->[%s]", identity, wID)
	}
	idHash := identity.Hash()
	if err := r.KVS.Get("meta"+idHash, meta); err != nil {
		return errors.WithMessagef(err, "failed to retrieve identity's metadata [%s]", identity)
	}
	return nil
}

// GetWallet returns the wallet identifier bound to the passed identity
func (r *WalletsRegistry) GetWallet(identity view.Identity) (string, error) {
	var wID string
	if err := r.KVS.Get(identity.Hash(), &wID); err != nil {
		return "", err
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("wallet [%s] is bound to identity [%s]", wID, identity)
	}
	return wID, nil
}

// ContainsIdentity returns true if the passed identity belongs to the passed wallet,
// false otherwise
func (r *WalletsRegistry) ContainsIdentity(identity view.Identity, wID string) bool {
	return r.KVS.Exists(r.Wallets[wID].Prefix + identity.Hash())
}

// WalletIDs returns the list of owner wallet identifiers
func (r *WalletsRegistry) WalletIDs() ([]string, error) {
	walletIDs, err := r.IdentityProvider.WalletIDs(r.IdentityRole)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get wallet identifiers from identity provider")
	}
	it, err := r.KVS.GetByPartialCompositeID("wallets", []string{r.ID.Network, r.ID.Channel, r.ID.Namespace})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get wallets iterator")
	}
	duplicates := map[string]bool{}
	for _, id := range walletIDs {
		duplicates[id] = true
	}

	for it.HasNext() {
		entry := &WalletEntry{}
		_, err := it.Next(entry)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get next wallets from iterator")
		}
		_, found := duplicates[entry.ID]
		if !found {
			walletIDs = append(walletIDs, entry.ID)
			duplicates[entry.ID] = true
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
