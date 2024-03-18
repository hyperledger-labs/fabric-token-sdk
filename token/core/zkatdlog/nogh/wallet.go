/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type DeserializerFunc func() (driver.Deserializer, error)

type WalletFactory struct {
	IdentityProvider driver.IdentityProvider
	TokenVault       TokenVault
	ConfigManager    config.Manager
	Deserializer     DeserializerFunc
}

func NewWalletFactory(identityProvider driver.IdentityProvider, tokenVault TokenVault, configManager config.Manager, deserializer DeserializerFunc) *WalletFactory {
	return &WalletFactory{IdentityProvider: identityProvider, TokenVault: tokenVault, ConfigManager: configManager, Deserializer: deserializer}
}

func (w *WalletFactory) NewWallet(role driver.IdentityRole, walletRegistry common.WalletRegistry, info driver.IdentityInfo, id string) (driver.Wallet, error) {
	switch role {
	case driver.OwnerRole:
		newWallet, err := newOwnerWallet(w.IdentityProvider, w.TokenVault, w.ConfigManager, w.Deserializer, walletRegistry, id, info)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to create new owner wallet [%s]", id)
		}
		logger.Debugf("created owner wallet [%s]", id)
		return newWallet, nil
	case driver.IssuerRole:
		idInfoIdentity, _, err := info.Get()
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to get issuer wallet identity for [%s]", id)
		}
		newWallet := newIssuerWallet(w.IdentityProvider, w.TokenVault, id, idInfoIdentity)
		if err := walletRegistry.BindIdentity(idInfoIdentity, info.EnrollmentID(), id, nil); err != nil {
			return nil, errors.WithMessagef(err, "programming error, failed to register recipient identity [%s]", id)
		}
		logger.Debugf("created issuer wallet [%s]", id)
		return newWallet, nil
	case driver.AuditorRole:
		logger.Debugf("no wallet found, create it [%s]", id)
		idInfoIdentity, _, err := info.Get()
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to get auditor wallet identity for [%s]", id)
		}
		newWallet := newAuditorWallet(w.IdentityProvider, id, idInfoIdentity)
		if err := walletRegistry.BindIdentity(idInfoIdentity, info.EnrollmentID(), id, nil); err != nil {
			return nil, errors.WithMessagef(err, "programming error, failed to register recipient identity [%s]", id)
		}
		logger.Debugf("created auditor wallet [%s]", id)
		return newWallet, nil
	default:
		return nil, errors.Errorf("role [%d] not supported", role)
	}
}

type ownerWallet struct {
	IdentityProvider driver.IdentityProvider
	TokenVault       TokenVault
	ConfigManager    config.Manager
	Deserializer     DeserializerFunc
	WalletRegistry   common.WalletRegistry

	id           string
	identityInfo driver.IdentityInfo
	cache        *idemix.WalletIdentityCache
}

func newOwnerWallet(
	IdentityProvider driver.IdentityProvider,
	TokenVault TokenVault,
	ConfigManager config.Manager,
	Deserializer DeserializerFunc,
	walletRegistry common.WalletRegistry,
	id string,
	identityInfo driver.IdentityInfo,
) (*ownerWallet, error) {
	w := &ownerWallet{
		IdentityProvider: IdentityProvider,
		TokenVault:       TokenVault,
		Deserializer:     Deserializer,
		WalletRegistry:   walletRegistry,
		id:               id,
		identityInfo:     identityInfo,
	}
	cacheSize := 0
	tmsConfig := ConfigManager.TMS()
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
	return w.WalletRegistry.ContainsIdentity(identity, w.id)
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
	if err := w.WalletRegistry.BindIdentity(pseudonym, w.identityInfo.EnrollmentID(), w.id, nil); err != nil {
		return nil, errors.WithMessagef(err, "failed storing recipient identity in wallet [%s]", w.ID())
	}
	return pseudonym, nil
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

func (w *ownerWallet) EnrollmentID() string {
	return w.identityInfo.EnrollmentID()
}

func (w *ownerWallet) RegisterRecipient(data *driver.RecipientData) error {
	if data == nil {
		return errors.WithStack(common.ErrNilRecipientData)
	}
	logger.Debugf("register recipient identity [%s] with audit info [%s]", data.Identity.String(), hash.Hashable(data.AuditInfo).String())

	// recognize identity and register it
	d, err := w.Deserializer()
	if err != nil {
		return errors.Wrap(err, "failed getting deserializer")
	}
	// match identity and audit info
	err = d.Match(data.Identity, data.AuditInfo)
	if err != nil {
		return errors.Wrapf(err, "failed to match identity to audit infor for [%s:%s]", data.Identity, hash.Hashable(data.AuditInfo))
	}
	// register verifier and audit info
	v, err := d.GetOwnerVerifier(data.Identity)
	if err != nil {
		return errors.Wrapf(err, "failed getting verifier for [%s]", data.Identity)
	}
	if err := w.IdentityProvider.RegisterVerifier(data.Identity, v); err != nil {
		return errors.Wrapf(err, "failed registering verifier for [%s]", data.Identity)
	}
	if err := w.IdentityProvider.RegisterAuditInfo(data.Identity, data.AuditInfo); err != nil {
		return errors.Wrapf(err, "failed registering audit info for [%s]", data.Identity)
	}
	if err := w.WalletRegistry.BindIdentity(data.Identity, w.EnrollmentID(), w.id, nil); err != nil {
		return errors.WithMessagef(err, "failed storing recipient identity in wallet [%s]", w.id)
	}
	return nil
}

func (w *ownerWallet) GetSigner(identity view.Identity) (driver.Signer, error) {
	if !w.Contains(identity) {
		return nil, errors.Errorf("identity [%s] does not belong to this wallet [%s]", identity, w.ID())
	}

	si, err := w.IdentityProvider.GetSigner(identity)
	if err != nil {
		return nil, err
	}
	return si, err
}

func (w *ownerWallet) ListTokens(opts *driver.ListTokensOptions) (*token.UnspentTokens, error) {
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

func (w *ownerWallet) Remote() bool {
	return w.identityInfo.Remote()
}

type issuerWallet struct {
	IdentityProvider driver.IdentityProvider
	TokenVault       TokenVault

	id       string
	identity view.Identity
}

func newIssuerWallet(IdentityProvider driver.IdentityProvider, TokenVault TokenVault, id string, identity view.Identity) *issuerWallet {
	return &issuerWallet{
		IdentityProvider: IdentityProvider,
		TokenVault:       TokenVault,
		id:               id,
		identity:         identity,
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
	si, err := w.IdentityProvider.GetSigner(identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting signer for identity [%s] in wallet [%s]", identity, w.identity)
	}
	return si, nil
}

func (w *issuerWallet) HistoryTokens(opts *driver.ListTokensOptions) (*token.IssuedTokens, error) {
	logger.Debugf("issuer wallet [%s]: history tokens, type [%d]", w.ID(), opts.TokenType)
	if w.TokenVault == nil {
		return nil, errors.Errorf("no backend provided, cannot perform operation")
	}
	source, err := w.TokenVault.ListHistoryIssuedTokens()
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
	IdentityProvider driver.IdentityProvider
	id               string
	identity         view.Identity
}

func newAuditorWallet(IdentityProvider driver.IdentityProvider, id string, identity view.Identity) *auditorWallet {
	return &auditorWallet{
		IdentityProvider: IdentityProvider,
		id:               id,
		identity:         identity,
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

	si, err := w.IdentityProvider.GetSigner(w.identity)
	if err != nil {
		return nil, err
	}
	return si, err
}
