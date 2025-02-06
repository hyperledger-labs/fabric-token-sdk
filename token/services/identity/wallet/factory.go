/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wallet

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenVault interface {
	IsPending(id *token.ID) (bool, error)
	UnspentTokensIteratorBy(ctx context.Context, id string, tokenType token.Type) (driver.UnspentTokensIterator, error)
	ListHistoryIssuedTokens() (*token.IssuedTokens, error)
	Balance(id string, tokenType token.Type) (uint64, error)
}

type WalletsConfiguration interface {
	CacheSizeForOwnerID(id string) int
}

type Factory struct {
	Logger               logging.Logger
	IdentityProvider     driver.IdentityProvider
	TokenVault           TokenVault
	walletsConfiguration WalletsConfiguration
	Deserializer         driver.Deserializer
}

func NewFactory(
	logger logging.Logger,
	identityProvider driver.IdentityProvider,
	tokenVault TokenVault,
	walletsConfiguration WalletsConfiguration,
	deserializer driver.Deserializer,
) *Factory {
	return &Factory{
		Logger:               logger,
		IdentityProvider:     identityProvider,
		TokenVault:           tokenVault,
		walletsConfiguration: walletsConfiguration,
		Deserializer:         deserializer,
	}
}

func (w *Factory) NewWallet(id string, role identity.RoleType, walletRegistry Registry, identityInfo identity.Info) (driver.Wallet, error) {
	switch role {
	case identity.OwnerRole:
		if identityInfo.Anonymous() {
			newWallet, err := NewAnonymousOwnerWallet(
				w.Logger,
				w.IdentityProvider,
				w.TokenVault,
				w.Deserializer,
				walletRegistry,
				id,
				identityInfo,
				w.walletsConfiguration.CacheSizeForOwnerID(id),
			)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed to create new owner wallet [%s]", id)
			}
			w.Logger.Debugf("created owner wallet [%s] for identity [%s:%s:%v]", id, identityInfo.ID(), identityInfo.EnrollmentID(), identityInfo.Remote())
			return newWallet, nil
		}

		// non-anonymous
		newWallet, err := NewLongTermOwnerWallet(w.IdentityProvider, w.TokenVault, id, identityInfo)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to create owner wallet [%s]", id)
		}
		return newWallet, nil
	case identity.IssuerRole:
		idInfoIdentity, _, err := identityInfo.Get()
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to get issuer wallet identity for [%s]", id)
		}
		newWallet := NewIssuerWallet(w.Logger, w.IdentityProvider, w.TokenVault, id, idInfoIdentity)
		if err := walletRegistry.BindIdentity(idInfoIdentity, identityInfo.EnrollmentID(), id, nil); err != nil {
			return nil, errors.WithMessagef(err, "programming error, failed to register recipient identity [%s]", id)
		}
		w.Logger.Debugf("created issuer wallet [%s]", id)
		return newWallet, nil
	case identity.AuditorRole:
		w.Logger.Debugf("no wallet found, create it [%s]", id)
		idInfoIdentity, _, err := identityInfo.Get()
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to get auditor wallet identity for [%s]", id)
		}
		newWallet := NewAuditorWallet(w.IdentityProvider, id, idInfoIdentity)
		if err := walletRegistry.BindIdentity(idInfoIdentity, identityInfo.EnrollmentID(), id, nil); err != nil {
			return nil, errors.WithMessagef(err, "programming error, failed to register recipient identity [%s]", id)
		}
		w.Logger.Debugf("created auditor wallet [%s]", id)
		return newWallet, nil
	case identity.CertifierRole:
		return nil, errors.Errorf("certifiers are not supported by this driver")
	default:
		return nil, errors.Errorf("role [%d] not supported", role)
	}
}
