/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	api2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
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
	managementService *ManagementService
}

func (wm *WalletManager) IsMe(id view.Identity) bool {
	s, err := wm.managementService.tms.IdentityProvider().GetSigner(id)
	return err == nil && s != nil
}

// RegisterOwnerWallet registers a new owner wallet with type passed id
func (wm *WalletManager) RegisterOwnerWallet(id string, path string) error {
	return wm.managementService.tms.RegisterOwnerWallet(id, path)
}

// RegisterRecipientIdentity registers a new recipient identity
func (wm *WalletManager) RegisterRecipientIdentity(id view.Identity, auditInfo []byte, metadata []byte) error {
	if err := wm.managementService.tms.IdentityProvider().RegisterRecipientIdentity(id); err != nil {
		return err
	}
	return wm.managementService.tms.RegisterRecipientIdentity(id, auditInfo, metadata)
}

// Wallet returns the wallet bound to the passed identity, if any is available.
// If no wallet is found, it returns nil.
func (wm *WalletManager) Wallet(identity view.Identity) *Wallet {
	w := wm.managementService.tms.Wallet(identity)
	if w == nil {
		return nil
	}
	return &Wallet{w: w, managementService: wm.managementService}
}

// OwnerWallet returns the owner wallet bound to the passed identifier, if any is available.
// The identifier can be a label, as defined in the configuration file, an identity or a wallet ID.
// If no wallet is found, it returns nil.
func (wm *WalletManager) OwnerWallet(id string) *OwnerWallet {
	w := wm.managementService.tms.OwnerWallet(id)
	if w == nil {
		return nil
	}
	return &OwnerWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// OwnerWalletByIdentity returns the owner wallet bound to the passed identity, if any is available.
// If no wallet is found, it returns nil.
func (wm *WalletManager) OwnerWalletByIdentity(identity view.Identity) *OwnerWallet {
	w := wm.managementService.tms.OwnerWalletByIdentity(identity)
	if w == nil {
		return nil
	}
	return &OwnerWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// IssuerWallet returns the issuer wallet bound to the passed identifier, if any is available.
// The identifier can be a label, as defined in the configuration file, an identity or a wallet ID.
// If no wallet is found, it returns nil.
func (wm *WalletManager) IssuerWallet(id string) *IssuerWallet {
	w := wm.managementService.tms.IssuerWallet(id)
	if w == nil {
		return nil
	}
	return &IssuerWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// IssuerWalletByIdentity returns the issuer wallet bound to the passed identity, if any is available.
// If no wallet is found, it returns nil.
func (wm *WalletManager) IssuerWalletByIdentity(identity view.Identity) *IssuerWallet {
	w := wm.managementService.tms.IssuerWalletByIdentity(identity)
	if w == nil {
		return nil
	}
	return &IssuerWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// AuditorWallet returns the auditor wallet bound to the passed identifier, if any is available.
// The identifier can be a label, as defined in the configuration file, an identity or a wallet ID.
// If no wallet is found, it returns nil.
func (wm *WalletManager) AuditorWallet(id string) *AuditorWallet {
	w := wm.managementService.tms.AuditorWallet(id)
	if w == nil {
		return nil
	}
	return &AuditorWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

func (wm *WalletManager) AuditorWalletByIdentity(id view.Identity) *AuditorWallet {
	w := wm.managementService.tms.AuditorWalletByIdentity(id)
	if w == nil {
		return nil
	}
	return &AuditorWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// CertifierWallet returns the certifier wallet bound to the passed identifier, if any is available.
// The identifier can be a label, as defined in the configuration file, an identity or a wallet ID.
// If no wallet is found, it returns nil.
func (wm *WalletManager) CertifierWallet(id string) *CertifierWallet {
	w := wm.managementService.tms.CertifierWallet(id)
	if w == nil {
		return nil
	}
	return &CertifierWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// CertifierWalletByIdentity returns the certifier wallet bound to the passed identity, if any is available.
// If no wallet is found, it returns nil.
func (wm *WalletManager) CertifierWalletByIdentity(identity view.Identity) *CertifierWallet {
	w := wm.managementService.tms.CertifierWalletByIdentity(identity)
	if w == nil {
		return nil
	}
	return &CertifierWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// GetEnrollmentID returns the enrollment ID of passed identity
func (wm *WalletManager) GetEnrollmentID(identity view.Identity) (string, error) {
	auditInfo, err := wm.managementService.tms.IdentityProvider().GetAuditInfo(identity)
	if err != nil {
		return "", errors.WithMessagef(err, "failed to get audit info for identity %s", identity)
	}
	return wm.managementService.tms.IdentityProvider().GetEnrollmentID(auditInfo)
}

// SpentIDs returns the spent keys corresponding to the passed token IDs
func (wm *WalletManager) SpentIDs(ids []*token2.ID) ([]string, error) {
	return wm.managementService.tms.SpentIDs(ids...)
}

// Wallet models a generic wallet that has an identifier and contains one or mode identities.
// These identities own tokens.
type Wallet struct {
	w                 api2.Wallet
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
func (w *Wallet) Contains(identity view.Identity) bool {
	return w.w.Contains(identity)
}

// ContainsToken returns true if the wallet contains an identity that owns the passed token.
func (w *Wallet) ContainsToken(token *token2.UnspentToken) bool {
	return w.w.ContainsToken(token)
}

// AuditorWallet models the wallet of an auditor
type AuditorWallet struct {
	*Wallet
	w api2.AuditorWallet
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
	*Wallet
	w api2.CertifierWallet
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
	*Wallet
	w api2.OwnerWallet
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

func (o *OwnerWallet) EnrollmentID() string {
	return o.w.EnrollmentID()
}

// IssuerWallet models the wallet of an issuer
type IssuerWallet struct {
	*Wallet
	w api2.IssuerWallet
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
	compiledOpts, err := CompileListTokensOption(opts...)
	if err != nil {
		return nil, err
	}
	return i.w.HistoryTokens(compiledOpts)
}

func CompileListTokensOption(opts ...ListTokensOption) (*api2.ListTokensOptions, error) {
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
