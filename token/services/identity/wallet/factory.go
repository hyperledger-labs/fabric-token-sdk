/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wallet

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenVault interface {
	IsPending(ctx context.Context, id *token.ID) (bool, error)
	UnspentTokensIteratorBy(ctx context.Context, id string, tokenType token.Type) (driver.UnspentTokensIterator, error)
	ListHistoryIssuedTokens(ctx context.Context) (*token.IssuedTokens, error)
	Balance(ctx context.Context, id string, tokenType token.Type) (uint64, error)
}

type WalletsConfiguration interface {
	CacheSizeForOwnerID(id string) int
}

type Factory struct {
	Logger               logging.Logger
	IdentityProvider     driver.IdentityProvider
	TokenVault           TokenVault
	WalletsConfiguration WalletsConfiguration
	Deserializer         driver.Deserializer
	MetricsProvider      metrics.Provider
}

func NewFactory(
	logger logging.Logger,
	identityProvider driver.IdentityProvider,
	tokenVault TokenVault,
	walletsConfiguration WalletsConfiguration,
	deserializer driver.Deserializer,
	metricsProvider metrics.Provider,
) *Factory {
	return &Factory{
		Logger:               logger,
		IdentityProvider:     identityProvider,
		TokenVault:           tokenVault,
		WalletsConfiguration: walletsConfiguration,
		Deserializer:         deserializer,
		MetricsProvider:      metricsProvider,
	}
}

func (w *Factory) NewWallet(ctx context.Context, id string, role identity.RoleType, wr Registry, info identity.Info) (driver.Wallet, error) {
	switch role {
	case identity.OwnerRole:
		if info.Anonymous() {
			newWallet, err := NewAnonymousOwnerWallet(
				w.Logger,
				w.IdentityProvider,
				w.TokenVault,
				w.Deserializer,
				wr,
				id,
				info,
				w.WalletsConfiguration.CacheSizeForOwnerID(id),
				w.MetricsProvider,
			)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed to create new owner wallet [%s]", id)
			}
			w.Logger.Debugf("created owner wallet [%s] for identity [%s:%s:%v]", id, info.ID(), info.EnrollmentID(), info.Remote())
			return newWallet, nil
		}

		// non-anonymous
		newWallet, err := NewLongTermOwnerWallet(w.IdentityProvider, w.TokenVault, id, info)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to create owner wallet [%s]", id)
		}
		return newWallet, nil
	case identity.IssuerRole:
		idInfoIdentity, _, err := info.Get(ctx)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to get issuer wallet identity for [%s]", id)
		}
		newWallet := NewIssuerWallet(w.Logger, w.IdentityProvider, w.TokenVault, id, idInfoIdentity)
		if err := wr.BindIdentity(ctx, idInfoIdentity, info.EnrollmentID(), id, nil); err != nil {
			return nil, errors.WithMessagef(err, "programming error, failed to register recipient identity [%s]", id)
		}
		w.Logger.Debugf("created issuer wallet [%s]", id)
		return newWallet, nil
	case identity.AuditorRole:
		w.Logger.Debugf("no wallet found, create it [%s]", id)
		idInfoIdentity, _, err := info.Get(ctx)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to get auditor wallet identity for [%s]", id)
		}
		newWallet := NewAuditorWallet(w.IdentityProvider, id, idInfoIdentity)
		if err := wr.BindIdentity(ctx, idInfoIdentity, info.EnrollmentID(), id, nil); err != nil {
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
