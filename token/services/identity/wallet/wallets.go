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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type OwnerTokenVault interface {
	UnspentTokensIteratorBy(ctx context.Context, id string, tokenType token.Type) (driver.UnspentTokensIterator, error)
	Balance(id string, tokenType token.Type) (uint64, error)
}

type AuditorWallet struct {
	IdentityProvider driver.IdentityProvider
	WalletID         string
	AuditorIdentity  driver.Identity
}

func NewAuditorWallet(IdentityProvider driver.IdentityProvider, id string, identity driver.Identity) *AuditorWallet {
	return &AuditorWallet{
		IdentityProvider: IdentityProvider,
		WalletID:         id,
		AuditorIdentity:  identity,
	}
}

func (w *AuditorWallet) ID() string {
	return w.WalletID
}

func (w *AuditorWallet) Contains(identity driver.Identity) bool {
	return w.AuditorIdentity.Equal(identity)
}

func (w *AuditorWallet) ContainsToken(token *token.UnspentToken) bool {
	return w.Contains(token.Owner)
}

func (w *AuditorWallet) GetAuditorIdentity() (driver.Identity, error) {
	return w.AuditorIdentity, nil
}

func (w *AuditorWallet) GetSigner(id driver.Identity) (driver.Signer, error) {
	if !w.Contains(id) {
		return nil, errors.Errorf("identity [%s] does not belong to this wallet [%s]", id, w.ID())
	}
	return w.IdentityProvider.GetSigner(id)
}

type IssuerTokenVault interface {
	ListHistoryIssuedTokens() (*token.IssuedTokens, error)
}

type IssuerWallet struct {
	Logger           logging.Logger
	IdentityProvider driver.IdentityProvider
	TokenVault       IssuerTokenVault
	WalletID         string
	IssuerIdentity   driver.Identity
}

func NewIssuerWallet(Logger logging.Logger, IdentityProvider driver.IdentityProvider, TokenVault IssuerTokenVault, id string, identity driver.Identity) *IssuerWallet {
	return &IssuerWallet{
		Logger:           Logger,
		IdentityProvider: IdentityProvider,
		TokenVault:       TokenVault,
		WalletID:         id,
		IssuerIdentity:   identity,
	}
}

func (w *IssuerWallet) ID() string {
	return w.WalletID
}

func (w *IssuerWallet) Contains(identity driver.Identity) bool {
	return w.IssuerIdentity.Equal(identity)
}

func (w *IssuerWallet) ContainsToken(token *token.UnspentToken) bool {
	return w.Contains(token.Owner)
}

func (w *IssuerWallet) GetIssuerIdentity(tokenType token.Type) (driver.Identity, error) {
	return w.IssuerIdentity, nil
}

func (w *IssuerWallet) GetSigner(identity driver.Identity) (driver.Signer, error) {
	if !w.Contains(identity) {
		return nil, errors.Errorf("failed getting signer, the passed identity [%s] does not belong to this wallet [%s]", identity, w.ID())
	}
	return w.IdentityProvider.GetSigner(identity)
}

func (w *IssuerWallet) HistoryTokens(opts *driver.ListTokensOptions) (*token.IssuedTokens, error) {
	w.Logger.Debugf("issuer wallet [%s]: history tokens, type [%d]", w.ID(), opts.TokenType)
	source, err := w.TokenVault.ListHistoryIssuedTokens()
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}

	unspentTokens := &token.IssuedTokens{}
	for _, t := range source.Tokens {
		if len(opts.TokenType) != 0 && t.Type != opts.TokenType {
			w.Logger.Debugf("issuer wallet [%s]: discarding token of type [%s]!=[%s]", w.ID(), t.Type, opts.TokenType)
			continue
		}

		if !w.Contains(t.Issuer) {
			w.Logger.Debugf("issuer wallet [%s]: discarding token, issuer does not belong to wallet", w.ID())
			continue
		}

		w.Logger.Debugf("issuer wallet [%s]: adding token of type [%s], quantity [%s]", w.ID(), t.Type, t.Quantity)
		unspentTokens.Tokens = append(unspentTokens.Tokens, t)
	}
	w.Logger.Debugf("issuer wallet [%s]: history tokens done, found [%d] issued tokens", w.ID(), len(unspentTokens.Tokens))

	return unspentTokens, nil
}

type CertifierWallet struct {
	IdentityProvider  driver.IdentityProvider
	WalletID          string
	CertifierIdentity driver.Identity
}

func NewCertifierWallet(IdentityProvider driver.IdentityProvider, id string, identity driver.Identity) *CertifierWallet {
	return &CertifierWallet{
		IdentityProvider:  IdentityProvider,
		WalletID:          id,
		CertifierIdentity: identity,
	}
}

func (w *CertifierWallet) ID() string {
	return w.WalletID
}

func (w *CertifierWallet) Contains(identity driver.Identity) bool {
	return w.CertifierIdentity.Equal(identity)
}

func (w *CertifierWallet) ContainsToken(token *token.UnspentToken) bool {
	return w.Contains(token.Owner)
}

func (w *CertifierWallet) GetCertifierIdentity() (driver.Identity, error) {
	return w.CertifierIdentity, nil
}

func (w *CertifierWallet) GetSigner(id driver.Identity) (driver.Signer, error) {
	if !w.Contains(id) {
		return nil, errors.Errorf("identity does not belong to this AnonymousOwnerWallet [%s]", id)
	}
	return w.IdentityProvider.GetSigner(id)
}

type LongTermOwnerWallet struct {
	IdentityProvider  driver.IdentityProvider
	TokenVault        OwnerTokenVault
	WalletID          string
	OwnerIdentityInfo identity.Info
	OwnerIdentity     driver.Identity
	OwnerAuditInfo    []byte
}

func NewLongTermOwnerWallet(IdentityProvider driver.IdentityProvider, TokenVault OwnerTokenVault, id string, identityInfo identity.Info) (*LongTermOwnerWallet, error) {
	identity, auditInfo, err := identityInfo.Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get identity info")
	}

	return &LongTermOwnerWallet{
		IdentityProvider:  IdentityProvider,
		TokenVault:        TokenVault,
		WalletID:          id,
		OwnerIdentityInfo: identityInfo,
		OwnerIdentity:     identity,
		OwnerAuditInfo:    auditInfo,
	}, nil
}

func (w *LongTermOwnerWallet) ID() string {
	return w.WalletID
}

func (w *LongTermOwnerWallet) Contains(identity driver.Identity) bool {
	return w.OwnerIdentity.Equal(identity)
}

func (w *LongTermOwnerWallet) ContainsToken(token *token.UnspentToken) bool {
	return w.Contains(token.Owner)
}

func (w *LongTermOwnerWallet) GetRecipientIdentity() (driver.Identity, error) {
	return w.OwnerIdentity, nil
}

func (w *LongTermOwnerWallet) GetRecipientData() (*driver.RecipientData, error) {
	return &driver.RecipientData{
		Identity:  w.OwnerIdentity,
		AuditInfo: w.OwnerAuditInfo,
	}, nil
}

func (w *LongTermOwnerWallet) GetAuditInfo(id driver.Identity) ([]byte, error) {
	return w.IdentityProvider.GetAuditInfo(id)
}

func (w *LongTermOwnerWallet) GetTokenMetadata(id driver.Identity) ([]byte, error) {
	return nil, nil
}

func (w *LongTermOwnerWallet) GetTokenMetadataAuditInfo(id driver.Identity) ([]byte, error) {
	return nil, nil
}

func (w *LongTermOwnerWallet) GetSigner(identity driver.Identity) (driver.Signer, error) {
	if !w.Contains(identity) {
		return nil, errors.Errorf("identity [%s] does not belong to this wallet [%s]", identity, w.ID())
	}
	return w.IdentityProvider.GetSigner(identity)
}

func (w *LongTermOwnerWallet) ListTokens(opts *driver.ListTokensOptions) (*token.UnspentTokens, error) {
	it, err := w.TokenVault.UnspentTokensIteratorBy(opts.Context, w.WalletID, opts.TokenType)
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
		unspentTokens.Tokens = append(unspentTokens.Tokens, t)
	}
	return unspentTokens, nil
}

func (w *LongTermOwnerWallet) Balance(opts *driver.ListTokensOptions) (uint64, error) {
	balance, err := w.TokenVault.Balance(w.WalletID, opts.TokenType)
	if err != nil {
		return 0, errors.Wrap(err, "token selection failed")
	}
	return balance, nil
}

func (w *LongTermOwnerWallet) ListTokensIterator(opts *driver.ListTokensOptions) (driver.UnspentTokensIterator, error) {
	it, err := w.TokenVault.UnspentTokensIteratorBy(opts.Context, w.WalletID, opts.TokenType)
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}
	return it, nil
}

func (w *LongTermOwnerWallet) EnrollmentID() string {
	return w.OwnerIdentityInfo.EnrollmentID()
}

func (w *LongTermOwnerWallet) RegisterRecipient(data *driver.RecipientData) error {
	// TODO: if identity is equal to the one this wallet is bound to, then we are good. Otherwise return an error
	return nil
}

func (w *LongTermOwnerWallet) Remote() bool {
	return w.OwnerIdentityInfo.Remote()
}

type AnonymousOwnerWallet struct {
	*LongTermOwnerWallet
	Logger         logging.Logger
	Deserializer   driver.Deserializer
	WalletRegistry Registry
	IdentityCache  *IdentityCache
}

func NewAnonymousOwnerWallet(
	logger logging.Logger,
	IdentityProvider driver.IdentityProvider,
	TokenVault OwnerTokenVault,
	Deserializer driver.Deserializer,
	walletRegistry Registry,
	id string,
	identityInfo identity.Info,
	cacheSize int,
) (*AnonymousOwnerWallet, error) {
	w := &AnonymousOwnerWallet{
		LongTermOwnerWallet: &LongTermOwnerWallet{
			IdentityProvider:  IdentityProvider,
			TokenVault:        TokenVault,
			WalletID:          id,
			OwnerIdentityInfo: identityInfo,
		},
		Logger:         logger,
		WalletRegistry: walletRegistry,
		Deserializer:   Deserializer,
	}
	w.IdentityCache = NewIdentityCache(logger, w.getRecipientIdentity, cacheSize)
	logger.Debugf("added wallet cache for id %s with cache of size %d", id+"@"+identityInfo.EnrollmentID(), cacheSize)
	return w, nil
}

func (w *AnonymousOwnerWallet) Contains(identity driver.Identity) bool {
	return w.WalletRegistry.ContainsIdentity(identity, w.WalletID)
}

// ContainsToken returns true if the passed token is owned by this wallet
func (w *AnonymousOwnerWallet) ContainsToken(token *token.UnspentToken) bool {
	return w.Contains(token.Owner)
}

func (w *AnonymousOwnerWallet) GetRecipientIdentity() (driver.Identity, error) {
	rd, err := w.IdentityCache.RecipientData()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get recipient data")
	}
	return rd.Identity, nil
}

func (w *AnonymousOwnerWallet) GetRecipientData() (*driver.RecipientData, error) {
	return w.IdentityCache.RecipientData()
}

func (w *AnonymousOwnerWallet) RegisterRecipient(data *driver.RecipientData) error {
	if data == nil {
		return errors.WithStack(ErrNilRecipientData)
	}
	w.Logger.Debugf("register recipient identity [%s] with audit info [%s]", data.Identity, utils.Hashable(data.AuditInfo))

	// recognize identity and register it
	// match identity and audit info
	err := w.Deserializer.MatchIdentity(data.Identity, data.AuditInfo)
	if err != nil {
		return errors.Wrapf(err, "failed to match identity to audit infor for [%s]:[%s]", data.Identity, utils.Hashable(data.AuditInfo))
	}
	// register verifier and audit info
	v, err := w.Deserializer.GetOwnerVerifier(data.Identity)
	if err != nil {
		return errors.Wrapf(err, "failed getting verifier for owner [%s]", data.Identity)
	}
	if err := w.IdentityProvider.RegisterVerifier(data.Identity, v); err != nil {
		return errors.Wrapf(err, "failed registering verifier for owner [%s]", data.Identity)
	}
	if err := w.IdentityProvider.RegisterRecipientData(data); err != nil {
		return errors.Wrapf(err, "failed registering audit info for owner [%s]", data.Identity)
	}
	if err := w.WalletRegistry.BindIdentity(data.Identity, w.EnrollmentID(), w.WalletID, nil); err != nil {
		return errors.WithMessagef(err, "failed storing recipient identity in wallet [%s]", w.WalletID)
	}
	return nil
}

func (w *AnonymousOwnerWallet) getRecipientIdentity() (*driver.RecipientData, error) {
	// Get a new pseudonym
	pseudonym, auditInfo, err := w.OwnerIdentityInfo.Get()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting recipient identity from wallet [%s:%s]", w.ID(), w.OwnerIdentityInfo.EnrollmentID())
	}

	// Register the pseudonym
	if err := w.WalletRegistry.BindIdentity(pseudonym, w.OwnerIdentityInfo.EnrollmentID(), w.WalletID, nil); err != nil {
		return nil, errors.WithMessagef(err, "failed storing recipient identity in wallet [%s]", w.ID())
	}
	return &driver.RecipientData{
		Identity:  pseudonym,
		AuditInfo: auditInfo,
	}, nil
}

func (w *AnonymousOwnerWallet) GetSigner(identity driver.Identity) (driver.Signer, error) {
	if !w.Contains(identity) {
		return nil, errors.Errorf("identity [%s] does not belong to this wallet [%s]", identity, w.ID())
	}
	return w.IdentityProvider.GetSigner(identity)
}
