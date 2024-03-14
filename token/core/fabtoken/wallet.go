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

type WalletFactory struct{}

func (w *WalletFactory) NewOwnerWallet(s *common.WalletService, idInfo driver.IdentityInfo, wID string) (driver.OwnerWallet, error) {
	idInfoIdentity, _, err := idInfo.Get()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get owner wallet identity for [%s]", wID)
	}

	newWallet := newOwnerWallet(s, idInfoIdentity, wID, idInfo)
	if err := s.OwnerWalletsRegistry.RegisterWallet(wID, newWallet); err != nil {
		return nil, errors.WithMessagef(err, "failed to register rwallet [%s]", wID)
	}
	if err := s.OwnerWalletsRegistry.BindIdentity(idInfoIdentity, idInfo.EnrollmentID(), wID, nil); err != nil {
		return nil, errors.WithMessagef(err, "failed to register recipient identity [%s]", idInfoIdentity)
	}
	logger.Debugf("created owner wallet [%s:%s]", idInfo.ID, wID)
	return newWallet, nil
}

func (w *WalletFactory) NewIssuerWallet(s *common.WalletService, idInfo driver.IdentityInfo, wID string) (driver.IssuerWallet, error) {
	idInfoIdentity, _, err := idInfo.Get()
	if err != nil {
		logger.Errorf("failed to get issuer wallet identity for [%s]: %s", wID, err)
		return nil, nil
	}
	newWallet := newIssuerWallet(s, wID, idInfoIdentity)
	if err := s.IssuerWalletsRegistry.RegisterWallet(wID, newWallet); err != nil {
		return nil, errors.WithMessagef(err, "failed to register issuer wallet [%s]", wID)
	}
	if err := s.IssuerWalletsRegistry.BindIdentity(idInfoIdentity, idInfo.EnrollmentID(), wID, nil); err != nil {
		return nil, errors.WithMessagef(err, "failed to register recipient identity [%s]", wID)
	}
	logger.Debugf("created issuer wallet [%s]", wID)
	return newWallet, nil
}

func (w *WalletFactory) NewAuditorWallet(s *common.WalletService, idInfo driver.IdentityInfo, wID string) (driver.AuditorWallet, error) {
	idInfoIdentity, _, err := idInfo.Get()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get auditor wallet identity for [%s]", wID)
	}
	newWallet := newAuditorWallet(s, wID, idInfoIdentity)
	if err := s.AuditorWalletsRegistry.RegisterWallet(wID, newWallet); err != nil {
		return nil, errors.WithMessagef(err, "failed to register auditor wallet [%s]", wID)
	}
	if err := s.AuditorWalletsRegistry.BindIdentity(idInfoIdentity, idInfo.EnrollmentID(), wID, nil); err != nil {
		return nil, errors.WithMessagef(err, "failed to register recipient identity [%s]", wID)
	}
	logger.Debugf("created auditor wallet [%s]", wID)
	return newWallet, nil
}

func (w *WalletFactory) NewCertifierWallet(s *common.WalletService, idInfo driver.IdentityInfo, wID string) (driver.CertifierWallet, error) {
	panic("not supported")
}

type ownerWallet struct {
	tokenService *common.WalletService
	id           string
	identityInfo driver.IdentityInfo
	identity     view.Identity
}

func newOwnerWallet(tokenService *common.WalletService, identity view.Identity, id string, identityInfo driver.IdentityInfo) *ownerWallet {
	return &ownerWallet{
		tokenService: tokenService,
		id:           id,
		identity:     identity,
		identityInfo: identityInfo,
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
	return w.tokenService.IdentityProvider.GetAuditInfo(id)
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

	si, err := w.tokenService.IdentityProvider.GetSigner(w.identity)
	if err != nil {
		return nil, err
	}
	return si, err
}

func (w *ownerWallet) ListTokens(opts *driver.ListTokensOptions) (*token.UnspentTokens, error) {
	logger.Debugf("wallet: list tokens, type [%s]", opts.TokenType)
	it, err := w.tokenService.TokenVault.UnspentTokensIteratorBy(w.id, opts.TokenType)
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
	it, err := w.tokenService.TokenVault.UnspentTokensIteratorBy(w.id, opts.TokenType)
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

type issuerWallet struct {
	tokenService *common.WalletService
	id           string
	identity     view.Identity
}

func newIssuerWallet(tokenService *common.WalletService, id string, identity view.Identity) *issuerWallet {
	return &issuerWallet{
		tokenService: tokenService,
		id:           id,
		identity:     identity,
	}
}

func (w *issuerWallet) ID() string {
	return w.id
}

func (w *issuerWallet) Contains(identity view.Identity) bool {
	return w.identity.Equal(identity)
}

func (w *issuerWallet) ContainsToken(token *token.UnspentToken) bool {
	return w.Contains(token.Owner.Raw)
}

func (w *issuerWallet) GetIssuerIdentity(tokenType string) (view.Identity, error) {
	return w.identity, nil
}

func (w *issuerWallet) GetSigner(identity view.Identity) (driver.Signer, error) {
	if !w.Contains(identity) {
		return nil, errors.Errorf("failed getting signer, the passed identity [%s] does not belong to this wallet [%s]", identity, w.ID())
	}
	si, err := w.tokenService.IdentityProvider.GetSigner(identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting issuer signer for identity [%s] in wallet [%s]", identity, w.identity)
	}
	return si, nil
}

func (w *issuerWallet) HistoryTokens(opts *driver.ListTokensOptions) (*token.IssuedTokens, error) {
	logger.Debugf("issuer wallet [%s]: history tokens, type [%d]", w.ID(), opts.TokenType)
	source, err := w.tokenService.TokenVault.ListHistoryIssuedTokens()
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}

	unspentTokens := &token.IssuedTokens{}
	for _, t := range source.Tokens {
		if len(opts.TokenType) != 0 && t.Type != opts.TokenType {
			logger.Debugf("issuer wallet [%s]: discarding token of type [%s]!=[%s]", w.ID(), t.Type, opts.TokenType)
			continue
		}

		if !w.Contains(t.Issuer.Raw) {
			logger.Debugf("issuer wallet [%s]: discarding token, issuer does not belong to wallet", w.ID())
			continue
		}

		logger.Debugf("issuer wallet [%s]: adding token of type [%s], quantity [%s]", w.ID(), t.Type, t.Quantity)
		unspentTokens.Tokens = append(unspentTokens.Tokens, t)
	}
	logger.Debugf("issuer wallet [%s]: history tokens done, found [%d] issued tokens", w.ID(), len(unspentTokens.Tokens))

	return unspentTokens, nil
}

type auditorWallet struct {
	tokenService *common.WalletService
	id           string
	identity     view.Identity
}

func newAuditorWallet(tokenService *common.WalletService, id string, identity view.Identity) *auditorWallet {
	return &auditorWallet{
		tokenService: tokenService,
		id:           id,
		identity:     identity,
	}
}

func (w *auditorWallet) ID() string {
	return w.id
}

func (w *auditorWallet) Contains(identity view.Identity) bool {
	return w.identity.Equal(identity)
}

func (w *auditorWallet) ContainsToken(token *token.UnspentToken) bool {
	return w.Contains(token.Owner.Raw)
}

func (w *auditorWallet) GetAuditorIdentity() (view.Identity, error) {
	return w.identity, nil
}

func (w *auditorWallet) GetSigner(id view.Identity) (driver.Signer, error) {
	if !w.Contains(id) {
		return nil, errors.Errorf("identity does not belong to this wallet [%s]", id.String())
	}

	si, err := w.tokenService.IdentityProvider.GetSigner(w.identity)
	if err != nil {
		return nil, err
	}
	return si, err
}
