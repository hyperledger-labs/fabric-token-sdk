/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenVault interface {
	IsPending(id *token.ID) (bool, error)
	UnspentTokensIteratorBy(ctx context.Context, id, tokenType string) (driver.UnspentTokensIterator, error)
	ListHistoryIssuedTokens() (*token.IssuedTokens, error)
	Balance(id, tokenType string) (uint64, error)
}

type WalletsConfiguration interface {
	CacheSizeForOwnerID(id string) int
}

type WalletFactory struct {
	Logger               logging.Logger
	IdentityProvider     driver.IdentityProvider
	TokenVault           TokenVault
	walletsConfiguration WalletsConfiguration
	Deserializer         driver.Deserializer
}

func NewWalletFactory(
	logger logging.Logger,
	identityProvider driver.IdentityProvider,
	tokenVault TokenVault,
	walletsConfiguration WalletsConfiguration,
	deserializer driver.Deserializer,
) *WalletFactory {
	return &WalletFactory{
		Logger:               logger,
		IdentityProvider:     identityProvider,
		TokenVault:           tokenVault,
		walletsConfiguration: walletsConfiguration,
		Deserializer:         deserializer,
	}
}

func (w *WalletFactory) NewWallet(role driver.IdentityRole, walletRegistry common.WalletRegistry, info driver.IdentityInfo, id string) (driver.Wallet, error) {
	switch role {
	case driver.OwnerRole:
		newWallet, err := common.NewAnonymousOwnerWallet(
			w.Logger,
			w.IdentityProvider,
			w.TokenVault,
			w.Deserializer,
			walletRegistry,
			id,
			info,
			w.walletsConfiguration.CacheSizeForOwnerID(id),
		)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to create new owner wallet [%s]", id)
		}
		w.Logger.Debugf("created owner wallet [%s]", id)
		return newWallet, nil
	case driver.IssuerRole:
		idInfoIdentity, _, err := info.Get()
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to get issuer wallet identity for [%s]", id)
		}
		newWallet := common.NewIssuerWallet(w.Logger, w.IdentityProvider, w.TokenVault, id, idInfoIdentity)
		if err := walletRegistry.BindIdentity(idInfoIdentity, info.EnrollmentID(), id, nil); err != nil {
			return nil, errors.WithMessagef(err, "programming error, failed to register recipient identity [%s]", id)
		}
		w.Logger.Debugf("created issuer wallet [%s]", id)
		return newWallet, nil
	case driver.AuditorRole:
		w.Logger.Debugf("no wallet found, create it [%s]", id)
		idInfoIdentity, _, err := info.Get()
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to get auditor wallet identity for [%s]", id)
		}
		newWallet := common.NewAuditorWallet(w.IdentityProvider, id, idInfoIdentity)
		if err := walletRegistry.BindIdentity(idInfoIdentity, info.EnrollmentID(), id, nil); err != nil {
			return nil, errors.WithMessagef(err, "programming error, failed to register recipient identity [%s]", id)
		}
		w.Logger.Debugf("created auditor wallet [%s]", id)
		return newWallet, nil
	case driver.CertifierRole:
		return nil, errors.Errorf("certifiers are not supported by this driver")
	default:
		return nil, errors.Errorf("role [%d] not supported", role)
	}
}
