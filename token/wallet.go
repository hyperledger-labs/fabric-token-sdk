/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	api2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// ListTokensOptions options for listing tokens
type ListTokensOptions struct {
	TokenType string
}

// ListTokensOption is a function that configures a ListTokensOptions
type ListTokensOption func(*ListTokensOptions) error

// WithType returns a list token option that filter by the passed token type.
// If the passed token type is the empty string, all token types are selected.
func WithType(tokenType string) ListTokensOption {
	return func(o *ListTokensOptions) error {
		o.TokenType = tokenType
		return nil
	}
}

// WalletManager defines the interface for managing wallets.
type WalletManager struct {
	ts *ManagementService
}

func (t *WalletManager) IsMe(id view.Identity) bool {
	s, err := t.ts.tms.IdentityProvider().GetSigner(id)
	return err == nil && s != nil
}

// RegisterOwnerWallet registers a new owner wallet with type passed id
func (t *WalletManager) RegisterOwnerWallet(id string, typ string, path string) error {
	return t.ts.tms.RegisterOwnerWallet(id, typ, path)
}

// RegisterRecipientIdentity registers a new recipient identity
func (t *WalletManager) RegisterRecipientIdentity(id view.Identity, auditInfo []byte, metadata []byte) error {
	if err := t.ts.tms.IdentityProvider().RegisterRecipientIdentity(id); err != nil {
		return err
	}
	return t.ts.tms.RegisterRecipientIdentity(id, auditInfo, metadata)
}

// Wallet returns the wallet bound to the passed identity, if any is available.
// If no wallet is found, it returns nil.
func (t *WalletManager) Wallet(identity view.Identity) *Wallet {
	w := t.ts.tms.Wallet(identity)
	if w == nil {
		return nil
	}
	return &Wallet{w: w}
}

// OwnerWallet returns the owner wallet bound to the passed identifier, if any is available.
// The identifier can be a label, an identity or a wallet ID.
// If no wallet is found, it returns nil.
func (t *WalletManager) OwnerWallet(id string) *OwnerWallet {
	w := t.ts.tms.OwnerWallet(id)
	if w == nil {
		return nil
	}
	return &OwnerWallet{w: w, tms: t.ts}
}

// OwnerWalletByIdentity returns the owner wallet bound to the passed identity, if any is available.
// If no wallet is found, it returns nil.
func (t *WalletManager) OwnerWalletByIdentity(identity view.Identity) *OwnerWallet {
	w := t.ts.tms.OwnerWalletByIdentity(identity)
	if w == nil {
		return nil
	}
	return &OwnerWallet{w: w, tms: t.ts}
}

// IssuerWallet returns the issuer wallet bound to the passed identifier, if any is available.
// The identifier can be a label, an identity or a wallet ID.
// If no wallet is found, it returns nil.
func (t *WalletManager) IssuerWallet(id string) *IssuerWallet {
	w := t.ts.tms.IssuerWallet(id)
	if w == nil {
		return nil
	}
	return &IssuerWallet{w: w, tms: t.ts}
}

// IssuerWalletByIdentity returns the issuer wallet bound to the passed identity, if any is available.
// If no wallet is found, it returns nil.
func (t *WalletManager) IssuerWalletByIdentity(identity view.Identity) *IssuerWallet {
	w := t.ts.tms.IssuerWalletByIdentity(identity)
	if w == nil {
		return nil
	}
	return &IssuerWallet{w: w, tms: t.ts}
}

// AuditorWallet returns the auditor wallet bound to the passed identifier, if any is available.
// The identifier can be a label, an identity or a wallet ID.
// If no wallet is found, it returns nil.
func (t *WalletManager) AuditorWallet(id string) *AuditorWallet {
	w := t.ts.tms.AuditorWallet(id)
	if w == nil {
		return nil
	}
	return &AuditorWallet{w: w, tms: t.ts}
}

// CertifierWallet returns the certifier wallet bound to the passed identifier, if any is available.
// The identifier can be a label, an identity or a wallet ID.
// If no wallet is found, it returns nil.
func (t *WalletManager) CertifierWallet(id string) *CertifierWallet {
	w := t.ts.tms.CertifierWallet(id)
	if w == nil {
		return nil
	}
	return &CertifierWallet{w: w}
}

// CertifierWalletByIdentity returns the certifier wallet bound to the passed identity, if any is available.
// If no wallet is found, it returns nil.
func (t *WalletManager) CertifierWalletByIdentity(identity view.Identity) *CertifierWallet {
	w := t.ts.tms.CertifierWalletByIdentity(identity)
	if w == nil {
		return nil
	}
	return &CertifierWallet{w: w}
}

// Wallet models a generic wallet that has an identifier and contains one or mode identities.
// These identities own tokens.
type Wallet struct {
	w api2.Wallet
}

// ID returns the wallet identifier.
func (w *Wallet) ID() string {
	return w.w.ID()
}

// Contains returns true if the wallet contains the passed identity.
func (w *Wallet) Contains(identity view.Identity) bool {
	return w.w.Contains(identity)
}

// ContainsToken returns true if the wallet contains an identity that owns the passed token.
func (w *Wallet) ContainsToken(token *token2.UnspentToken) bool {
	return w.ContainsToken(token)
}

// AuditorWallet models the wallet of an auditor
type AuditorWallet struct {
	w   api2.AuditorWallet
	tms *ManagementService
}

// ID returns the wallet identifier.
func (a *AuditorWallet) ID() string {
	return a.w.ID()
}

// TMS returns the token management service.
func (a *AuditorWallet) TMS() *ManagementService {
	return a.tms
}

// Contains returns true if the wallet contains the passed identity.
func (a *AuditorWallet) Contains(identity view.Identity) bool {
	return a.w.Contains(identity)
}

// ContainsToken returns true if the wallet contains an identity that owns the passed token.
func (a *AuditorWallet) ContainsToken(token *token2.UnspentToken) bool {
	return a.w.ContainsToken(token)
}

// GetAuditorIdentity returns the auditor identity. This can be a long term identity or a pseudonym depending
// on the underlying token driver.
func (a *AuditorWallet) GetAuditorIdentity() (view.Identity, error) {
	return a.w.GetAuditorIdentity()
}

// GetSigner returns the signer bound to the passed auditor identity.
func (a *AuditorWallet) GetSigner(id view.Identity) (api2.Signer, error) {
	return a.w.GetSigner(id)
}

// CertifierWallet models the wallet of a certifier
type CertifierWallet struct {
	w api2.CertifierWallet
}

// ID returns the wallet identifier.
func (a *CertifierWallet) ID() string {
	return a.w.ID()
}

// Contains returns true if the wallet contains the passed identity.
func (a *CertifierWallet) Contains(identity view.Identity) bool {
	return a.w.Contains(identity)
}

// ContainsToken returns true if the wallet contains an identity that owns the passed token.
func (a *CertifierWallet) ContainsToken(token *token2.UnspentToken) bool {
	return a.w.ContainsToken(token)
}

// GetCertifierIdentity returns the certifier identity. This can be a long term identity or a pseudonym depending
// on the underlying token driver.
func (a *CertifierWallet) GetCertifierIdentity() (view.Identity, error) {
	return a.w.GetCertifierIdentity()
}

// GetSigner returns the signer bound to the passed certifier identity.
func (a *CertifierWallet) GetSigner(id view.Identity) (api2.Signer, error) {
	return a.w.GetSigner(id)
}

// OwnerWallet models the wallet of an owner
type OwnerWallet struct {
	w   api2.OwnerWallet
	tms *ManagementService
}

// ID returns the wallet identifier.
func (o *OwnerWallet) ID() string {
	return o.w.ID()
}

// TMS returns the token management service.
func (o *OwnerWallet) TMS() *ManagementService {
	return o.tms
}

// Contains returns true if the wallet contains the passed identity.
func (o *OwnerWallet) Contains(identity view.Identity) bool {
	return o.w.Contains(identity)
}

// ContainsToken returns true if the wallet contains an identity that owns the passed token.
func (o *OwnerWallet) ContainsToken(token *token2.UnspentToken) bool {
	return o.w.ContainsToken(token)
}

// GetRecipientIdentity returns the owner identity. This can be a long term identity or a pseudonym depending
// on the underlying token driver.
func (o *OwnerWallet) GetRecipientIdentity() (view.Identity, error) {
	return o.w.GetRecipientIdentity()
}

// GetAuditInfo returns the audit info bound to the passed owner identity.
func (o *OwnerWallet) GetAuditInfo(id view.Identity) ([]byte, error) {
	return o.w.GetAuditInfo(id)
}

// GetSigner returns the signer bound to the passed owner identity.
func (o *OwnerWallet) GetSigner(identity view.Identity) (api2.Signer, error) {
	return o.w.GetSigner(identity)
}

// GetTokenMetadata returns the token metadata bound to the passed owner identity.
func (o *OwnerWallet) GetTokenMetadata(token []byte) ([]byte, error) {
	return o.w.GetTokenMetadata(token)
}

// ListUnspentTokens returns a list of unspent tokens owned by identities in this wallet and filtered by the passed options.
// Options: WithType
func (o *OwnerWallet) ListUnspentTokens(opts ...ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := compileListTokensOption(opts...)
	if err != nil {
		return nil, err
	}
	return o.w.ListTokens(compiledOpts)
}

func (o *OwnerWallet) EnrollmentID() string {
	return o.w.EnrollmentID()
}

// IssuerWallet models the wallet of an issuer
type IssuerWallet struct {
	w   api2.IssuerWallet
	tms *ManagementService
}

// ID returns the wallet identifier.
func (i *IssuerWallet) ID() string {
	return i.w.ID()
}

// TMS returns the token management service.
func (i *IssuerWallet) TMS() *ManagementService {
	return i.tms
}

// Contains returns true if the wallet contains the passed identity.
func (i *IssuerWallet) Contains(identity view.Identity) bool {
	return i.w.Contains(identity)
}

// ContainsToken returns true if the wallet contains an identity that owns the passed token.
func (i *IssuerWallet) ContainsToken(token *token2.UnspentToken) bool {
	return i.Contains(token.Owner.Raw)
}

// GetIssuerIdentity returns the issuer identity. This can be a long term identity or a pseudonym depending
// on the underlying token driver.
func (i *IssuerWallet) GetIssuerIdentity(tokenType string) (view.Identity, error) {
	return i.w.GetIssuerIdentity(tokenType)
}

// GetSigner returns the signer bound to the passed issuer identity.
func (i *IssuerWallet) GetSigner(identity view.Identity) (Signer, error) {
	return i.w.GetSigner(identity)
}

// ListIssuedTokens returns the list of tokens issued by identities in this wallet and filter by the passed options.
// Options: WithType
func (i *IssuerWallet) ListIssuedTokens(opts ...ListTokensOption) (*token2.IssuedTokens, error) {
	compiledOpts, err := compileListTokensOption(opts...)
	if err != nil {
		return nil, err
	}
	return i.w.HistoryTokens(compiledOpts)
}

func compileListTokensOption(opts ...ListTokensOption) (*api2.ListTokensOptions, error) {
	txOptions := &ListTokensOptions{}
	for _, opt := range opts {
		if err := opt(txOptions); err != nil {
			return nil, err
		}
	}
	return &api2.ListTokensOptions{
		TokenType: txOptions.TokenType,
	}, nil
}
