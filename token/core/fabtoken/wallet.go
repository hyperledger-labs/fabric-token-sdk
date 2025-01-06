/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenVault interface {
	PublicParams() ([]byte, error)
	UnspentTokensIteratorBy(ctx context.Context, id string, tokenType token.Type) (driver.UnspentTokensIterator, error)
	ListHistoryIssuedTokens() (*token.IssuedTokens, error)
	Balance(id string, tokenType token.Type) (uint64, error)
}

type WalletFactory struct {
	logger           logging.Logger
	identityProvider driver.IdentityProvider
	tokenVault       TokenVault
}

func NewWalletFactory(logger logging.Logger, identityProvider driver.IdentityProvider, tokenVault TokenVault) *WalletFactory {
	return &WalletFactory{logger: logger, identityProvider: identityProvider, tokenVault: tokenVault}
}

func (w *WalletFactory) NewWallet(role driver.IdentityRole, walletRegistry common.WalletRegistry, info driver.IdentityInfo, id string) (driver.Wallet, error) {
	idInfoIdentity, _, err := info.Get()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get owner wallet identity for [%s]", id)
	}

	var newWallet driver.Wallet
	switch role {
	case driver.OwnerRole:
		newWallet = common.NewLongTermOwnerWallet(w.identityProvider, w.tokenVault, idInfoIdentity, id, info)
	case driver.IssuerRole:
		newWallet = common.NewIssuerWallet(w.logger, w.identityProvider, w.tokenVault, id, idInfoIdentity)
	case driver.AuditorRole:
		newWallet = common.NewAuditorWallet(w.identityProvider, id, idInfoIdentity)
	case driver.CertifierRole:
		return nil, errors.Errorf("certifiers are not supported by this driver")
	default:
		return nil, errors.Errorf("role [%d] not supported", role)
	}
	if err := walletRegistry.BindIdentity(idInfoIdentity, info.EnrollmentID(), id, nil); err != nil {
		return nil, errors.WithMessagef(err, "failed to register recipient identity [%s]", id)
	}
	w.logger.Debugf("created auditor wallet [%s]", id)
	return newWallet, nil
}
