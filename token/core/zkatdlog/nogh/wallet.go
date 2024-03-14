/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type WalletFactory struct{}

func (w *WalletFactory) NewOwnerWallet(s *common.WalletService, idInfo driver.IdentityInfo, wID string) (driver.OwnerWallet, error) {
	newWallet, err := newOwnerWallet(s, wID, idInfo)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create new owner wallet [%s]", wID)
	}
	logger.Debugf("created owner wallet [%s]", wID)
	return newWallet, nil
}

func (w *WalletFactory) NewIssuerWallet(s *common.WalletService, idInfo driver.IdentityInfo, wID string) (driver.IssuerWallet, error) {
	idInfoIdentity, _, err := idInfo.Get()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get issuer wallet identity for [%s]", wID)
	}
	newWallet := newIssuerWallet(s, wID, idInfoIdentity)
	if err := s.IssuerWalletsRegistry.RegisterWallet(wID, newWallet); err != nil {
		return nil, errors.WithMessagef(err, "programming error, failed to register issuer wallet [%s]", wID)
	}
	if err := s.IssuerWalletsRegistry.BindIdentity(idInfoIdentity, idInfo.EnrollmentID(), wID, nil); err != nil {
		return nil, errors.WithMessagef(err, "programming error, failed to register recipient identity [%s]", wID)
	}
	logger.Debugf("created issuer wallet [%s]", wID)
	return newWallet, nil
}

func (w *WalletFactory) NewAuditorWallet(s *common.WalletService, idInfo driver.IdentityInfo, wID string) (driver.AuditorWallet, error) {
	logger.Debugf("no wallet found, create it [%s]", wID)
	idInfoIdentity, _, err := idInfo.Get()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get auditor wallet identity for [%s]", wID)
	}
	newWallet := newAuditorWallet(s, wID, idInfoIdentity)
	if err := s.AuditorWalletsRegistry.RegisterWallet(wID, newWallet); err != nil {
		return nil, errors.WithMessagef(err, "programming error, failed to register auditor wallet [%s]", wID)
	}
	if err := s.AuditorWalletsRegistry.BindIdentity(idInfoIdentity, idInfo.EnrollmentID(), wID, nil); err != nil {
		return nil, errors.WithMessagef(err, "programming error, failed to register recipient identity [%s]", wID)
	}
	logger.Debugf("created auditor wallet [%s]", wID)
	return newWallet, nil
}

func (w *WalletFactory) NewCertifierWallet(s *common.WalletService, idInfo driver.IdentityInfo, wID string) (driver.CertifierWallet, error) {
	//TODO implement me
	panic("implement me")
}

type ownerWallet struct {
	WalletService *common.WalletService
	id            string
	identityInfo  driver.IdentityInfo
	cache         *idemix.WalletIdentityCache
}

func newOwnerWallet(walletService *common.WalletService, id string, identityInfo driver.IdentityInfo) (*ownerWallet, error) {
	w := &ownerWallet{
		WalletService: walletService,
		id:            id,
		identityInfo:  identityInfo,
	}
	if err := walletService.OwnerWalletsRegistry.RegisterWallet(id, w); err != nil {
		return nil, errors.Wrapf(err, "failed to register owner wallet [%s]", id)
	}

	cacheSize := 0
	tmsConfig := walletService.ConfigManager.TMS()
	conf := tmsConfig.GetOwnerWallet(id)
	if conf == nil {
		cacheSize = tmsConfig.GetWalletDefaultCacheSize()
	} else {
		cacheSize = conf.CacheSize
	}

	w.cache = idemix.NewWalletIdentityCache(w.getRecipientIdentity, cacheSize)
	logger.Debugf("added wallet cache for id %s with cache of size %d", id+"@"+identityInfo.EnrollmentID(), cacheSize)
	return w, nil
}

func (w *ownerWallet) ID() string {
	return w.id
}

func (w *ownerWallet) Contains(identity view.Identity) bool {
	return w.WalletService.OwnerWalletsRegistry.ContainsIdentity(identity, w.id)
}

// ContainsToken returns true if the passed token is owned by this wallet
func (w *ownerWallet) ContainsToken(token *token.UnspentToken) bool {
	return w.Contains(token.Owner.Raw)
}

func (w *ownerWallet) GetRecipientIdentity() (view.Identity, error) {
	return w.cache.Identity()
}

func (w *ownerWallet) getRecipientIdentity() (view.Identity, error) {
	// Get a new pseudonym
	pseudonym, _, err := w.identityInfo.Get()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting recipient identity from wallet [%s]", w.ID())
	}

	// Register the pseudonym
	if err := w.WalletService.OwnerWalletsRegistry.BindIdentity(pseudonym, w.identityInfo.EnrollmentID(), w.id, nil); err != nil {
		return nil, errors.WithMessagef(err, "failed storing recipient identity in wallet [%s]", w.ID())
	}
	return pseudonym, nil
}

func (w *ownerWallet) GetAuditInfo(id view.Identity) ([]byte, error) {
	return w.WalletService.IdentityProvider.GetAuditInfo(id)
}

func (w *ownerWallet) GetTokenMetadata(id view.Identity) ([]byte, error) {
	return nil, nil
}

func (w *ownerWallet) GetTokenMetadataAuditInfo(id view.Identity) ([]byte, error) {
	return nil, nil
}

func (w *ownerWallet) EnrollmentID() string {
	return w.identityInfo.EnrollmentID()
}

func (w *ownerWallet) RegisterRecipient(data *driver.RecipientData) error {
	logger.Debugf("register recipient identity on wallet [%s][%s]", data.Identity, w.id)
	if err := w.WalletService.RegisterRecipientIdentity(data); err != nil {
		return errors.WithMessagef(err, "failed to register recipient identity")
	}

	if err := w.WalletService.OwnerWalletsRegistry.BindIdentity(data.Identity, w.EnrollmentID(), w.id, nil); err != nil {
		return errors.WithMessagef(err, "failed storing recipient identity in wallet [%s]", w.id)
	}

	return nil
}

func (w *ownerWallet) GetSigner(identity view.Identity) (driver.Signer, error) {
	if !w.Contains(identity) {
		return nil, errors.Errorf("identity [%s] does not belong to this wallet [%s]", identity, w.ID())
	}

	si, err := w.WalletService.IdentityProvider.GetSigner(identity)
	if err != nil {
		return nil, err
	}
	return si, err
}

func (w *ownerWallet) ListTokens(opts *driver.ListTokensOptions) (*token.UnspentTokens, error) {
	logger.Debugf("wallet: list tokens, type [%s]", opts.TokenType)
	if w.WalletService.TokenVault == nil {
		return nil, errors.Errorf("no backend provided, cannot perform operation")
	}
	it, err := w.WalletService.TokenVault.UnspentTokensIteratorBy(w.id, opts.TokenType)
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
	if w.WalletService.TokenVault == nil {
		return nil, errors.Errorf("no backend provided, cannot perform operation")
	}
	it, err := w.WalletService.TokenVault.UnspentTokensIteratorBy(w.id, opts.TokenType)
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}
	return it, nil
}

func (w *ownerWallet) Remote() bool {
	return w.identityInfo.Remote()
}

type issuerWallet struct {
	walletService *common.WalletService

	id       string
	identity view.Identity
}

func newIssuerWallet(tokenService *common.WalletService, id string, identity view.Identity) *issuerWallet {
	return &issuerWallet{
		walletService: tokenService,
		id:            id,
		identity:      identity,
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
		return nil, errors.Errorf("identity [%s] does not belong to this wallet [%s]", identity, w.ID())
	}
	si, err := w.walletService.IdentityProvider.GetSigner(identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting signer for identity [%s] in wallet [%s]", identity, w.identity)
	}
	return si, nil
}

func (w *issuerWallet) HistoryTokens(opts *driver.ListTokensOptions) (*token.IssuedTokens, error) {
	logger.Debugf("issuer wallet [%s]: history tokens, type [%d]", w.ID(), opts.TokenType)
	if w.walletService.TokenVault == nil {
		return nil, errors.Errorf("no backend provided, cannot perform operation")
	}
	source, err := w.walletService.TokenVault.ListHistoryIssuedTokens()
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
	WalletService *common.WalletService
	id            string
	identity      view.Identity
}

func newAuditorWallet(tokenService *common.WalletService, id string, identity view.Identity) *auditorWallet {
	return &auditorWallet{
		WalletService: tokenService,
		id:            id,
		identity:      identity,
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
		return nil, errors.Errorf("identity [%s] does not belong to this wallet [%s]", id, w.ID())
	}

	si, err := w.WalletService.IdentityProvider.GetSigner(w.identity)
	if err != nil {
		return nil, err
	}
	return si, err
}
