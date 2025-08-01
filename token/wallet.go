/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// WalletLookupID defines the type of identifiers that can be used to retrieve a given wallet.
// It can be a string, as the name of the wallet, or an identity contained in that wallet.
// Ultimately, it is the token driver to decide which types are allowed.
type WalletLookupID = driver.WalletLookupID

// ListTokensOptions options for listing tokens
type ListTokensOptions = driver.ListTokensOptions

// ListTokensOption is a function that configures a ListTokensOptions
type ListTokensOption func(*ListTokensOptions) error

// WithType returns a list token option that filter by the passed token type.
// If the passed token type is the empty string, all token types are selected.
func WithType(tokenType token.Type) ListTokensOption {
	return func(o *ListTokensOptions) error {
		o.TokenType = tokenType
		return nil
	}
}

// WithContext return a list tokens option that contains the passed context
func WithContext(ctx context.Context) ListTokensOption {
	return func(o *ListTokensOptions) error {
		o.Context = ctx
		return nil
	}
}

type IdentityConfiguration = driver.IdentityConfiguration

// WalletManager defines the interface for managing wallets.
type WalletManager struct {
	walletService     driver.WalletService
	managementService *ManagementService
}

// RegisterOwnerIdentity registers an owner long-term identity. The identity will be loaded from the passed url.
// Depending on the support, the url can be a path in the file system or something else.
func (wm *WalletManager) RegisterOwnerIdentity(ctx context.Context, id string, url string) error {
	return wm.walletService.RegisterOwnerIdentity(ctx, driver.IdentityConfiguration{
		ID:  id,
		URL: url,
	})
}

// RegisterOwnerIdentityConfiguration registers an owner long-term identity via a identity configuration
func (wm *WalletManager) RegisterOwnerIdentityConfiguration(ctx context.Context, conf IdentityConfiguration) error {
	return wm.walletService.RegisterOwnerIdentity(ctx, conf)
}

// RegisterIssuerIdentity registers an issuer long-term identity. The identity will be loaded from the passed url.
// Depending on the support, the url can be a path in the file system or something else.
func (wm *WalletManager) RegisterIssuerIdentity(ctx context.Context, id string, url string) error {
	return wm.walletService.RegisterIssuerIdentity(ctx, driver.IdentityConfiguration{
		ID:  id,
		URL: url,
	})
}

// RegisterRecipientIdentity registers a new recipient identity
func (wm *WalletManager) RegisterRecipientIdentity(ctx context.Context, data *RecipientData) error {
	return wm.walletService.RegisterRecipientIdentity(ctx, data)
}

// Wallet returns the wallet bound to the passed identity, if any is available.
// If no wallet is found, it returns nil.
func (wm *WalletManager) Wallet(ctx context.Context, identity Identity) *Wallet {
	w := wm.walletService.Wallet(ctx, identity)
	if w == nil {
		return nil
	}
	return &Wallet{w: w, managementService: wm.managementService}
}

// OwnerWalletIDs returns the list of owner wallet identifiers
func (wm *WalletManager) OwnerWalletIDs(ctx context.Context) ([]string, error) {
	ids, err := wm.walletService.OwnerWalletIDs(ctx)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get the list of owner wallet identifiers")
	}
	return ids, nil
}

// OwnerWallet returns the owner wallet bound to the passed identifier, if any is available.
// The identifier can be a label, as defined in the configuration file, an identity or a wallet ID.
// If no wallet is found, it returns nil.
func (wm *WalletManager) OwnerWallet(ctx context.Context, id WalletLookupID) *OwnerWallet {
	w, err := wm.walletService.OwnerWallet(ctx, id)
	if err != nil {
		wm.managementService.logger.DebugfContext(ctx, "failed to get owner wallet for id [%s]: [%s]", id, err)
		return nil
	}
	return &OwnerWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// IssuerWallet returns the issuer wallet bound to the passed identifier, if any is available.
// The identifier can be a label, as defined in the configuration file, an identity or a wallet ID.
// If no wallet is found, it returns nil.
func (wm *WalletManager) IssuerWallet(ctx context.Context, id WalletLookupID) *IssuerWallet {
	w, err := wm.walletService.IssuerWallet(ctx, id)
	if err != nil {
		wm.managementService.logger.DebugfContext(ctx, "failed to get issuer wallet for id [%s]: [%s]", id, err)
		return nil
	}
	return &IssuerWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// AuditorWallet returns the auditor wallet bound to the passed identifier, if any is available.
// The identifier can be a label, as defined in the configuration file, an identity or a wallet ID.
// If no wallet is found, it returns nil.
func (wm *WalletManager) AuditorWallet(ctx context.Context, id WalletLookupID) *AuditorWallet {
	w, err := wm.walletService.AuditorWallet(ctx, id)
	if err != nil {
		wm.managementService.logger.DebugfContext(ctx, "failed to get auditor wallet for id [%s]: [%s]", id, err)
		return nil
	}
	return &AuditorWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// CertifierWallet returns the certifier wallet bound to the passed identifier, if any is available.
// The identifier can be a label, as defined in the configuration file, an identity or a wallet ID.
// If no wallet is found, it returns nil.
func (wm *WalletManager) CertifierWallet(ctx context.Context, id WalletLookupID) *CertifierWallet {
	w, err := wm.walletService.CertifierWallet(ctx, id)
	if err != nil {
		wm.managementService.logger.DebugfContext(ctx, "failed to get certifier wallet for id [%s]: [%s]", id, err)
		return nil
	}
	return &CertifierWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// GetEnrollmentID returns the enrollment ID of passed identity
func (wm *WalletManager) GetEnrollmentID(ctx context.Context, identity Identity) (string, error) {
	auditInfo, err := wm.walletService.GetAuditInfo(ctx, identity)
	if err != nil {
		return "", errors.WithMessagef(err, "failed to get audit info for identity %s", identity)
	}
	return wm.walletService.GetEnrollmentID(ctx, identity, auditInfo)
}

// GetRevocationHandle returns the revocation handle of the passed identity
func (wm *WalletManager) GetRevocationHandle(ctx context.Context, identity Identity) (string, error) {
	auditInfo, err := wm.walletService.GetAuditInfo(ctx, identity)
	if err != nil {
		return "", errors.WithMessagef(err, "failed to get audit info for identity %s", identity)
	}

	return wm.walletService.GetRevocationHandle(ctx, identity, auditInfo)
}

// SpentIDs returns the spent keys corresponding to the passed token IDs
func (wm *WalletManager) SpentIDs(ctx context.Context, ids []*token.ID) ([]string, error) {
	return wm.walletService.SpendIDs(ctx, ids...)
}

// Wallet models a generic wallet that has an identifier and contains one or mode identities.
// These identities own tokens.
type Wallet struct {
	w                 driver.Wallet
	managementService *ManagementService
}

// ID returns the wallet identifier.
func (w *Wallet) ID() string {
	return w.w.ID()
}

// TMS returns the token management service.
func (w *Wallet) TMS() *ManagementService {
	return w.managementService
}

// Contains returns true if the wallet contains the passed identity.
func (w *Wallet) Contains(identity Identity) bool {
	return w.w.Contains(identity)
}

// ContainsToken returns true if the wallet contains an identity that owns the passed token.
func (w *Wallet) ContainsToken(token *token.UnspentToken) bool {
	return w.w.ContainsToken(token)
}

// AuditorWallet models the wallet of an auditor
type AuditorWallet struct {
	*Wallet
	w driver.AuditorWallet
}

// GetAuditorIdentity returns the auditor identity. This can be a long term identity or a pseudonym depending
// on the underlying token driver.
func (a *AuditorWallet) GetAuditorIdentity() (Identity, error) {
	return a.w.GetAuditorIdentity()
}

// GetSigner returns the signer bound to the passed auditor identity.
func (a *AuditorWallet) GetSigner(ctx context.Context, identity driver.Identity) (driver.Signer, error) {
	return a.w.GetSigner(ctx, identity)
}

// CertifierWallet models the wallet of a certifier
type CertifierWallet struct {
	*Wallet
	w driver.CertifierWallet
}

// GetCertifierIdentity returns the certifier identity. This can be a long term identity or a pseudonym depending
// on the underlying token driver.
func (a *CertifierWallet) GetCertifierIdentity() (Identity, error) {
	return a.w.GetCertifierIdentity()
}

// GetSigner returns the signer bound to the passed certifier identity.
func (a *CertifierWallet) GetSigner(ctx context.Context, identity driver.Identity) (driver.Signer, error) {
	return a.w.GetSigner(ctx, identity)
}

// OwnerWallet models the wallet of an owner
type OwnerWallet struct {
	*Wallet
	w driver.OwnerWallet
}

// GetRecipientIdentity returns the owner identity. This can be a long term identity or a pseudonym depending
// on the underlying token driver.
func (o *OwnerWallet) GetRecipientIdentity(ctx context.Context) (Identity, error) {
	return o.w.GetRecipientIdentity(ctx)
}

// GetRecipientData return the owner recipient identity, it does not include token metadata audit info
func (o *OwnerWallet) GetRecipientData(ctx context.Context) (*RecipientData, error) {
	return o.w.GetRecipientData(ctx)
}

// GetAuditInfo returns auditing information for the passed identity
func (o *OwnerWallet) GetAuditInfo(ctx context.Context, id Identity) ([]byte, error) {
	return o.w.GetAuditInfo(ctx, id)
}

// GetTokenMetadata returns the public information related to the token to be assigned to passed recipient identity.
func (o *OwnerWallet) GetTokenMetadata(token []byte) ([]byte, error) {
	return o.w.GetTokenMetadata(token)
}

// GetTokenMetadataAuditInfo returns private information about the token metadata assigned to the passed recipient identity.
func (o *OwnerWallet) GetTokenMetadataAuditInfo(token []byte) ([]byte, error) {
	return o.w.GetTokenMetadataAuditInfo(token)
}

// GetSigner returns the signer bound to the passed owner identity.
func (o *OwnerWallet) GetSigner(ctx context.Context, identity driver.Identity) (driver.Signer, error) {
	return o.w.GetSigner(ctx, identity)
}

// ListUnspentTokens returns a list of unspent tokens owned by identities in this wallet and filtered by the passed options.
// Options: WithType
func (o *OwnerWallet) ListUnspentTokens(opts ...ListTokensOption) (*token.UnspentTokens, error) {
	compiledOpts, err := CompileListTokensOption(opts...)
	if err != nil {
		return nil, err
	}

	return o.w.ListTokens(compiledOpts)
}

// ListUnspentTokensIterator returns an iterator of unspent tokens owned by identities in this wallet and filtered by the passed options.
// Options: WithType
func (o *OwnerWallet) ListUnspentTokensIterator(opts ...ListTokensOption) (*UnspentTokensIterator, error) {
	compiledOpts, err := CompileListTokensOption(opts...)
	if err != nil {
		return nil, err
	}
	it, err := o.w.ListTokensIterator(compiledOpts)
	if err != nil {
		return nil, err
	}
	return &UnspentTokensIterator{UnspentTokensIterator: it}, nil
}

// Balance returns the sun of the amounts, with 64 bits of precision, of the tokens with type and EID equal to those passed as arguments.
func (o *OwnerWallet) Balance(ctx context.Context, opts ...ListTokensOption) (uint64, error) {
	compiledOpts, err := CompileListTokensOption(opts...)
	if err != nil {
		return 0, err
	}
	sum, err := o.w.Balance(ctx, compiledOpts)
	if err != nil {
		return 0, err
	}
	return sum, nil
}

func (o *OwnerWallet) EnrollmentID() string {
	return o.w.EnrollmentID()
}

// RegisterRecipient register the passed recipient data. The data is passed as pointer to allow the underlying token driver
// to modify them if needed.
func (o *OwnerWallet) RegisterRecipient(ctx context.Context, data *RecipientData) error {
	return o.w.RegisterRecipient(ctx, data)
}

// Remote returns true if this wallet is verify only, meaning that the corresponding secret key is external to this wallet
func (o *OwnerWallet) Remote() bool {
	return o.w.Remote()
}

// IssuerWallet models the wallet of an issuer
type IssuerWallet struct {
	*Wallet
	w driver.IssuerWallet
}

// GetIssuerIdentity returns the issuer identity. This can be a long term identity or a pseudonym depending
// on the underlying token driver.
func (i *IssuerWallet) GetIssuerIdentity(tokenType token.Type) (Identity, error) {
	return i.w.GetIssuerIdentity(tokenType)
}

// GetSigner returns the signer bound to the passed issuer identity.
func (i *IssuerWallet) GetSigner(ctx context.Context, identity driver.Identity) (driver.Signer, error) {
	return i.w.GetSigner(ctx, identity)
}

// ListIssuedTokens returns the list of tokens issued by identities in this wallet and filter by the passed options.
// Options: WithType
func (i *IssuerWallet) ListIssuedTokens(ctx context.Context, opts ...ListTokensOption) (*token.IssuedTokens, error) {
	compiledOpts, err := CompileListTokensOption(opts...)
	if err != nil {
		return nil, err
	}
	return i.w.HistoryTokens(ctx, compiledOpts)
}

func CompileListTokensOption(opts ...ListTokensOption) (*driver.ListTokensOptions, error) {
	txOptions := &ListTokensOptions{}
	for _, opt := range opts {
		if err := opt(txOptions); err != nil {
			return nil, err
		}
	}
	if txOptions.Context == nil {
		txOptions.Context = context.Background()
	}
	return &driver.ListTokensOptions{
		TokenType: txOptions.TokenType,
		Context:   txOptions.Context,
	}, nil
}
