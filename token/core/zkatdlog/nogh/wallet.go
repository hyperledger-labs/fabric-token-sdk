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

type TokenVault interface {
	IsPending(id *token.ID) (bool, error)
	UnspentTokensIteratorBy(id, tokenType string) (driver.UnspentTokensIterator, error)
	ListHistoryIssuedTokens() (*token.IssuedTokens, error)
}

type WalletFactory struct {
	IdentityProvider driver.IdentityProvider
	TokenVault       TokenVault
	ConfigManager    config.Manager
	Deserializer     driver.Deserializer
}

func NewWalletFactory(identityProvider driver.IdentityProvider, tokenVault TokenVault, configManager config.Manager, deserializer driver.Deserializer) *WalletFactory {
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
		newWallet := common.NewIssuerWallet(logger, w.IdentityProvider, w.TokenVault, id, idInfoIdentity)
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
		newWallet := common.NewAuditorWallet(w.IdentityProvider, id, idInfoIdentity)
		if err := walletRegistry.BindIdentity(idInfoIdentity, info.EnrollmentID(), id, nil); err != nil {
			return nil, errors.WithMessagef(err, "programming error, failed to register recipient identity [%s]", id)
		}
		logger.Debugf("created auditor wallet [%s]", id)
		return newWallet, nil
	case driver.CertifierRole:
		return nil, errors.Errorf("certifiers are not supported by this driver")
	default:
		return nil, errors.Errorf("role [%d] not supported", role)
	}
}

type ownerWallet struct {
	IdentityProvider driver.IdentityProvider
	TokenVault       TokenVault
	ConfigManager    config.Manager
	Deserializer     driver.Deserializer
	WalletRegistry   common.WalletRegistry

	id           string
	identityInfo driver.IdentityInfo
	cache        *idemix.WalletIdentityCache
}

func newOwnerWallet(
	IdentityProvider driver.IdentityProvider,
	TokenVault TokenVault,
	ConfigManager config.Manager,
	Deserializer driver.Deserializer,
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
	// match identity and audit info
	err := w.Deserializer.Match(data.Identity, data.AuditInfo)
	if err != nil {
		return errors.Wrapf(err, "failed to match identity to audit infor for [%s:%s]", data.Identity, hash.Hashable(data.AuditInfo))
	}
	// register verifier and audit info
	v, err := w.Deserializer.GetOwnerVerifier(data.Identity)
	if err != nil {
		return errors.Wrapf(err, "failed getting verifier for [%s]", data.Identity)
	}
	if err := w.IdentityProvider.RegisterVerifier(data.Identity, v); err != nil {
		return errors.Wrapf(err, "failed registering verifier for [%s]", data.Identity)
	}
	if err := w.IdentityProvider.RegisterRecipientData(data); err != nil {
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
