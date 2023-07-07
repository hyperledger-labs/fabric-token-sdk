/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
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
	walletService     driver.WalletService
	managementService *ManagementService
}

func (wm *WalletManager) IsMe(id view.Identity) bool {
	s, err := wm.managementService.tms.IdentityProvider().GetSigner(id)
	return err == nil && s != nil
}

// RegisterOwnerWallet registers a new owner wallet with the passed id
func (wm *WalletManager) RegisterOwnerWallet(id string, path string) error {
	return wm.walletService.RegisterOwnerWallet(id, path)
}

// RegisterIssuerWallet registers a new issuer wallet with the passed id
func (wm *WalletManager) RegisterIssuerWallet(id string, path string) error {
	return wm.walletService.RegisterIssuerWallet(id, path)
}

// RegisterRecipientIdentity registers a new recipient identity
func (wm *WalletManager) RegisterRecipientIdentity(id view.Identity, auditInfo []byte, metadata []byte) error {
	if err := wm.managementService.tms.IdentityProvider().RegisterRecipientIdentity(id); err != nil {
		return err
	}
	return wm.walletService.RegisterRecipientIdentity(id, auditInfo, metadata)
}

// Wallet returns the wallet bound to the passed identity, if any is available.
// If no wallet is found, it returns nil.
func (wm *WalletManager) Wallet(identity view.Identity) *Wallet {
	w := wm.walletService.Wallet(identity)
	if w == nil {
		return nil
	}
	return &Wallet{w: w, managementService: wm.managementService}
}

// OwnerWalletIDs returns the list of owner wallet identifiers
func (wm *WalletManager) OwnerWalletIDs() ([]string, error) {
	ids, err := wm.walletService.OwnerWalletIDs()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get the list of owner wallet identifiers")
	}
	return ids, nil
}

// OwnerWallet returns the owner wallet bound to the passed identifier, if any is available.
// The identifier can be a label, as defined in the configuration file, an identity or a wallet ID.
// If no wallet is found, it returns nil.
func (wm *WalletManager) OwnerWallet(id string) *OwnerWallet {
	w, err := wm.walletService.OwnerWallet(id)
	if err != nil {
		logger.Debugf("failed to get owner wallet for id [%s]: [%s]", id, err)
		return nil
	}
	return &OwnerWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// OwnerWalletByIdentity returns the owner wallet bound to the passed identity, if any is available.
// If no wallet is found, it returns nil.
func (wm *WalletManager) OwnerWalletByIdentity(identity view.Identity) *OwnerWallet {
	w, err := wm.walletService.OwnerWalletByIdentity(identity)
	if err != nil {
		logger.Debugf("failed to get owner wallet for id [%s]: [%s]", identity, err)
		return nil
	}
	return &OwnerWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// IssuerWallet returns the issuer wallet bound to the passed identifier, if any is available.
// The identifier can be a label, as defined in the configuration file, an identity or a wallet ID.
// If no wallet is found, it returns nil.
func (wm *WalletManager) IssuerWallet(id string) *IssuerWallet {
	w, err := wm.walletService.IssuerWallet(id)
	if err != nil {
		logger.Debugf("failed to get issuer wallet for id [%s]: [%s]", id, err)
		return nil
	}
	return &IssuerWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// IssuerWalletByIdentity returns the issuer wallet bound to the passed identity, if any is available.
// If no wallet is found, it returns nil.
func (wm *WalletManager) IssuerWalletByIdentity(identity view.Identity) *IssuerWallet {
	w, err := wm.walletService.IssuerWalletByIdentity(identity)
	if err != nil {
		logger.Debugf("failed to get issuer wallet for id [%s]: [%s]", identity, err)
		return nil
	}
	return &IssuerWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// AuditorWallet returns the auditor wallet bound to the passed identifier, if any is available.
// The identifier can be a label, as defined in the configuration file, an identity or a wallet ID.
// If no wallet is found, it returns nil.
func (wm *WalletManager) AuditorWallet(id string) *AuditorWallet {
	w, err := wm.walletService.AuditorWallet(id)
	if err != nil {
		logger.Debugf("failed to get auditor wallet for id [%s]: [%s]", id, err)
		return nil
	}
	return &AuditorWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

func (wm *WalletManager) AuditorWalletByIdentity(id view.Identity) *AuditorWallet {
	w, err := wm.walletService.AuditorWalletByIdentity(id)
	if err != nil {
		logger.Debugf("failed to get auditor wallet for id [%s]: [%s]", id, err)
		return nil
	}
	return &AuditorWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// CertifierWallet returns the certifier wallet bound to the passed identifier, if any is available.
// The identifier can be a label, as defined in the configuration file, an identity or a wallet ID.
// If no wallet is found, it returns nil.
func (wm *WalletManager) CertifierWallet(id string) *CertifierWallet {
	w, err := wm.walletService.CertifierWallet(id)
	if err != nil {
		logger.Debugf("failed to get certifier wallet for id [%s]: [%s]", id, err)
		return nil
	}
	return &CertifierWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// CertifierWalletByIdentity returns the certifier wallet bound to the passed identity, if any is available.
// If no wallet is found, it returns nil.
func (wm *WalletManager) CertifierWalletByIdentity(identity view.Identity) *CertifierWallet {
	w, err := wm.walletService.CertifierWalletByIdentity(identity)
	if err != nil {
		logger.Debugf("failed to get certifier wallet for id [%s]: [%s]", identity, err)
		return nil
	}
	return &CertifierWallet{Wallet: &Wallet{w: w, managementService: wm.managementService}, w: w}
}

// GetEnrollmentID returns the enrollment ID of passed identity
func (wm *WalletManager) GetEnrollmentID(identity view.Identity) (string, error) {
	auditInfo, err := wm.walletService.GetAuditInfo(identity)
	if err != nil {
		return "", errors.WithMessagef(err, "failed to get audit info for identity %s", identity)
	}
	return wm.walletService.GetEnrollmentID(auditInfo)
}

// GetRevocationHandle returns the revocation handle of the passed identity
func (wm *WalletManager) GetRevocationHandle(identity view.Identity) (string, error) {
	auditInfo, err := wm.walletService.GetAuditInfo(identity)
	if err != nil {
		return "", errors.WithMessagef(err, "failed to get audit info for identity %s", identity)
	}

	return wm.walletService.GetRevocationHandler(auditInfo)
}

// SpentIDs returns the spent keys corresponding to the passed token IDs
func (wm *WalletManager) SpentIDs(ids []*token.ID) ([]string, error) {
	return wm.walletService.SpentIDs(ids...)
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
func (w *Wallet) Contains(identity view.Identity) bool {
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
func (a *AuditorWallet) GetAuditorIdentity() (view.Identity, error) {
	return a.w.GetAuditorIdentity()
}

// GetSigner returns the signer bound to the passed auditor identity.
func (a *AuditorWallet) GetSigner(id view.Identity) (driver.Signer, error) {
	return a.w.GetSigner(id)
}

// CertifierWallet models the wallet of a certifier
type CertifierWallet struct {
	*Wallet
	w driver.CertifierWallet
}

// GetCertifierIdentity returns the certifier identity. This can be a long term identity or a pseudonym depending
// on the underlying token driver.
func (a *CertifierWallet) GetCertifierIdentity() (view.Identity, error) {
	return a.w.GetCertifierIdentity()
}

// GetSigner returns the signer bound to the passed certifier identity.
func (a *CertifierWallet) GetSigner(id view.Identity) (driver.Signer, error) {
	return a.w.GetSigner(id)
}

// OwnerWallet models the wallet of an owner
type OwnerWallet struct {
	*Wallet
	w driver.OwnerWallet
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
func (o *OwnerWallet) GetSigner(identity view.Identity) (driver.Signer, error) {
	return o.w.GetSigner(identity)
}

// GetTokenMetadata returns the token metadata bound to the passed owner identity.
func (o *OwnerWallet) GetTokenMetadata(token []byte) ([]byte, error) {
	return o.w.GetTokenMetadata(token)
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

func (o *OwnerWallet) EnrollmentID() string {
	return o.w.EnrollmentID()
}

func (o *OwnerWallet) RegisterRecipient(identity view.Identity, auditInfo []byte, metadata []byte) error {
	return o.w.RegisterRecipient(identity, auditInfo, metadata)
}

// IssuerWallet models the wallet of an issuer
type IssuerWallet struct {
	*Wallet
	w driver.IssuerWallet
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
func (i *IssuerWallet) ListIssuedTokens(opts ...ListTokensOption) (*token.IssuedTokens, error) {
	compiledOpts, err := CompileListTokensOption(opts...)
	if err != nil {
		return nil, err
	}
	return i.w.HistoryTokens(compiledOpts)
}

func CompileListTokensOption(opts ...ListTokensOption) (*driver.ListTokensOptions, error) {
	txOptions := &ListTokensOptions{}
	for _, opt := range opts {
		if err := opt(txOptions); err != nil {
			return nil, err
		}
	}
	return &driver.ListTokensOptions{
		TokenType: txOptions.TokenType,
	}, nil
}
