/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

// WalletRegistry manages wallets whose long-term identities have a given role.
type WalletRegistry struct {
	Logger  logging.Logger
	Role    identity.Role
	Storage idriver.WalletStoreService

	Wallets map[string]driver.Wallet
}

// NewWalletRegistry returns a new registry for the passed parameters.
// A registry is bound to a given role, and it is persistent.
// Long-term identities are provided by the passed identity provider
func NewWalletRegistry(logger logging.Logger, role identity.Role, storage idriver.WalletStoreService) *WalletRegistry {
	return &WalletRegistry{
		Logger:  logger,
		Role:    role,
		Storage: storage,
		Wallets: map[string]driver.Wallet{},
	}
}

func (r *WalletRegistry) RegisterIdentity(config driver.IdentityConfiguration) error {
	r.Logger.Debugf("register identity [%s:%s]", config.ID, config.URL)
	return r.Role.RegisterIdentity(config)
}

// Lookup searches the wallet corresponding to the passed id.
// If a wallet is found, Lookup returns the wallet and its identifier.
// If no wallet is found, Lookup returns the identity info and a potential wallet identifier for the passed id, if anything is found
func (r *WalletRegistry) Lookup(id driver.WalletLookupID) (driver.Wallet, idriver.IdentityInfo, string, error) {
	r.Logger.Debugf("lookup wallet by [%T]", id)
	var walletIdentifiers []string

	identity, walletID, err := r.Role.MapToIdentity(id)
	if err != nil {
		r.Logger.Errorf("failed to map wallet [%s] to identity [%s], use a fallback strategy", id, err)
		fail := true
		// give it a second change
		passedIdentity, ok := toViewIdentity(id)
		if ok {
			r.Logger.Debugf("lookup failed, check if there is a wallet for identity [%s]", passedIdentity)
			// is this identity registered
			wID, err := r.GetWalletID(passedIdentity)
			if err == nil && len(wID) != 0 {
				r.Logger.Debugf("lookup failed, there is a wallet for identity [%s]: [%s]", passedIdentity, wID)
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
	r.Logger.Debugf("looked-up identifier [%s:%s]", identity, logging.WalletID(walletID))
	wID := walletID
	walletEntry, ok := r.Wallets[wID]
	if ok {
		return walletEntry, nil, wID, nil
	}
	walletIdentifiers = append(walletIdentifiers, wID)

	// give it a second chance
	passedIdentity, ok := toViewIdentity(id)
	if ok {
		r.Logger.Debugf("no wallet found, check if there is a wallet for identity [%s]", passedIdentity)
		// is this identity registered
		passedWalletID, err := r.GetWalletID(passedIdentity)
		if err == nil && len(passedWalletID) != 0 {
			r.Logger.Debugf("no wallet found, there is a wallet for identity [%s]: [%s]", passedIdentity, passedWalletID)
			// we got a hit
			walletEntry, ok = r.Wallets[passedWalletID]
			if ok {
				return walletEntry, nil, passedWalletID, nil
			}
			r.Logger.Debugf("no wallet found, there is a wallet for identity [%s]: [%s] but it has not been recreated yet", passedIdentity, passedWalletID)
		}
		walletIdentifiers = append(walletIdentifiers, passedWalletID)
	}

	r.Logger.Debugf("no wallet found for [%s] at [%s]", passedIdentity, logging.WalletID(wID))
	if len(identity) != 0 {
		identityWID, err := r.GetWalletID(identity)
		r.Logger.Debugf("wallet for identity [%s] -> [%s:%s]", identity, identityWID, err)
		if err == nil && len(identityWID) != 0 {
			w, ok := r.Wallets[identityWID]
			if ok {
				r.Logger.Debugf("found wallet [%s:%s:%s:%s]", identity, walletID, w.ID(), identityWID)
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
		var idInfo idriver.IdentityInfo
		idInfo, err = r.Role.GetIdentityInfo(id)
		if err == nil {
			r.Logger.Debugf("identity info found at [%s]", logging.WalletID(id))
			return nil, idInfo, id, nil
		} else {
			r.Logger.Debugf("identity info not found at [%s]", logging.WalletID(id))
		}
	}
	return nil, nil, "", errors.Errorf("failed to get wallet info for [%s:%v]", toString(walletID), walletIdentifiers)
}

// RegisterWallet binds the passed wallet to the passed id
func (r *WalletRegistry) RegisterWallet(id string, w driver.Wallet) error {
	r.Wallets[id] = w
	return nil
}

// BindIdentity binds the passed identity to the passed wallet identifier.
// Additional metadata can be bound to the identity.
func (r *WalletRegistry) BindIdentity(identity driver.Identity, eID string, wID string, meta any) error {
	r.Logger.Debugf("put recipient identity [%s]->[%s]", identity, wID)
	metaEncoded, err := json.Marshal(meta)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal metadata")
	}
	return r.Storage.StoreIdentity(identity, eID, wID, int(r.Role.ID()), metaEncoded)
}

// ContainsIdentity returns true if the passed identity belongs to the passed wallet,
// false otherwise
func (r *WalletRegistry) ContainsIdentity(identity driver.Identity, wID string) bool {
	return r.Storage.IdentityExists(identity, wID, int(r.Role.ID()))
}

// WalletIDs returns the list of wallet identifiers
func (r *WalletRegistry) WalletIDs() ([]string, error) {
	walletIDs, err := r.Role.IdentityIDs()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get wallet identifiers from identity provider")
	}
	duplicates := map[string]bool{}
	for _, id := range walletIDs {
		duplicates[id] = true
	}

	ids, err := r.Storage.GetWalletIDs(int(r.Role.ID()))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get roles iterator")
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

// GetIdentityMetadata loads metadata bound to the passed identity into the passed meta argument
func (r *WalletRegistry) GetIdentityMetadata(identity driver.Identity, wID string, meta any) error {
	r.Logger.Debugf("get recipient identity metadata [%s]->[%s]", identity, wID)
	raw, err := r.Storage.LoadMeta(identity, wID, int(r.Role.ID()))
	if err != nil {
		return errors.WithMessagef(err, "failed to retrieve identity's metadata [%s]", identity)
	}
	return json.Unmarshal(raw, &meta)
}

// GetWalletID returns the wallet identifier bound to the passed identity
func (r *WalletRegistry) GetWalletID(identity driver.Identity) (string, error) {
	wID, err := r.Storage.GetWalletID(identity, int(r.Role.ID()))
	if err != nil {
		return "", nil
	}
	r.Logger.Debugf("wallet [%s] is bound to identity [%s]", wID, identity)
	return wID, nil
}

func toString(w string) string {
	if len(w) <= 20 {
		return strings.ToValidUTF8(w, "X")
	}

	return fmt.Sprintf("%s~%s", strings.ToValidUTF8(w[:20], "X"), hash.Hashable(w).String())
}

func toViewIdentity(id driver.WalletLookupID) (driver.Identity, bool) {
	switch v := id.(type) {
	case driver.Identity:
		return v, true
	case []byte:
		return v, true
	default:
		return nil, false
	}
}
