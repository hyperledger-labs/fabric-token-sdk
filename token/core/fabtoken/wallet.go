/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenVault interface {
	PublicParams() ([]byte, error)
	UnspentTokensIteratorBy(id, tokenType string) (driver.UnspentTokensIterator, error)
	ListHistoryIssuedTokens() (*token.IssuedTokens, error)
}

type WalletFactory struct {
	identityProvider driver.IdentityProvider
	tokenVault       TokenVault
}

func NewWalletFactory(identityProvider driver.IdentityProvider, tokenVault TokenVault) *WalletFactory {
	return &WalletFactory{identityProvider: identityProvider, tokenVault: tokenVault}
}

func (w *WalletFactory) NewWallet(role driver.IdentityRole, walletRegistry common.WalletRegistry, info driver.IdentityInfo, id string) (driver.Wallet, error) {
	idInfoIdentity, _, err := info.Get()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get owner wallet identity for [%s]", id)
	}

	var newWallet driver.Wallet
	switch role {
	case driver.OwnerRole:
		newWallet = newOwnerWallet(w.identityProvider, w.tokenVault, idInfoIdentity, id, info)
	case driver.IssuerRole:
		newWallet = common.NewIssuerWallet(logger, w.identityProvider, w.tokenVault, id, idInfoIdentity)
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
	logger.Debugf("created auditor wallet [%s]", id)
	return newWallet, nil
}

type ownerWallet struct {
	IdentityProvider driver.IdentityProvider
	TokenVault       TokenVault
	id               string
	identityInfo     driver.IdentityInfo
	identity         view.Identity
}

func newOwnerWallet(IdentityProvider driver.IdentityProvider, TokenVault TokenVault, identity view.Identity, id string, identityInfo driver.IdentityInfo) *ownerWallet {
	return &ownerWallet{
		IdentityProvider: IdentityProvider,
		TokenVault:       TokenVault,
		id:               id,
		identityInfo:     identityInfo,
		identity:         identity,
	}
}

func (w *ownerWallet) ID() string {
	return w.id
}

func (w *ownerWallet) Contains(identity view.Identity) bool {
	return w.identity.Equal(identity)
}

func (w *ownerWallet) ContainsToken(token *token.UnspentToken) bool {
	return w.Contains(token.Owner.Raw)
}

func (w *ownerWallet) GetRecipientIdentity() (view.Identity, error) {
	return w.identity, nil
}

func (w *ownerWallet) GetAuditInfo(id view.Identity) ([]byte, error) {
	return w.IdentityProvider.GetAuditInfo(id)
}

func (w *ownerWallet) GetTokenMetadata(id view.Identity) ([]byte, error) {
	return nil, nil
}

func (w *ownerWallet) GetTokenMetadataAuditInfo(id view.Identity) ([]byte, error) {
	return nil, nil
}

func (w *ownerWallet) GetSigner(identity view.Identity) (driver.Signer, error) {
	if !w.identity.Equal(identity) {
		return nil, errors.Errorf("identity does not belong to this wallet [%s]", identity.String())
	}

	si, err := w.IdentityProvider.GetSigner(w.identity)
	if err != nil {
		return nil, err
	}
	return si, err
}

func (w *ownerWallet) ListTokens(opts *driver.ListTokensOptions) (*token.UnspentTokens, error) {
	logger.Debugf("wallet: list tokens, type [%s]", opts.TokenType)
	it, err := w.TokenVault.UnspentTokensIteratorBy(w.id, opts.TokenType)
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}
	defer it.Close()

	unspentTokens := &token.UnspentTokens{}
	for {
		t, err := it.Next()
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to get next unspent token")

		}
		if t == nil {
			break
		}

		logger.Debugf("wallet: adding token of type [%s], quantity [%s]", t.Type, t.Quantity)
		unspentTokens.Tokens = append(unspentTokens.Tokens, t)
	}

	logger.Debugf("wallet: list tokens done, found [%d] unspent tokens", len(unspentTokens.Tokens))

	return unspentTokens, nil
}

func (w *ownerWallet) ListTokensIterator(opts *driver.ListTokensOptions) (driver.UnspentTokensIterator, error) {
	logger.Debugf("wallet: list tokens, type [%s]", opts.TokenType)
	it, err := w.TokenVault.UnspentTokensIteratorBy(w.id, opts.TokenType)
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}
	return it, nil
}

func (w *ownerWallet) EnrollmentID() string {
	return w.identityInfo.EnrollmentID()
}

func (w *ownerWallet) RegisterRecipient(data *driver.RecipientData) error {
	// TODO: if identity is equal to the one this wallet is bound to, then we are good. Otherwise return an error
	return nil
}

func (w *ownerWallet) Remote() bool {
	return w.identityInfo.Remote()
}
