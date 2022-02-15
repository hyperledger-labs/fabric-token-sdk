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

type ListTokensOptions struct {
	TokenType string
}

type ListTokensOption func(*ListTokensOptions) error

// WithType returns a list token option that filter by the passed token type.
// If the passed token type is the empty string, all token types are selected.
func WithType(tokenType string) ListTokensOption {
	return func(o *ListTokensOptions) error {
		o.TokenType = tokenType
		return nil
	}
}

type WalletManager struct {
	ts api2.TokenManagerService
}

func (t *WalletManager) IsMe(id view.Identity) bool {
	s, err := t.ts.IdentityProvider().GetSigner(id)
	return err == nil && s != nil
}

func (t *WalletManager) RegisterOwnerWallet(id string, typ string, path string) error {
	return t.ts.RegisterOwnerWallet(id, typ, path)
}

func (t *WalletManager) RegisterRecipientIdentity(id view.Identity, auditInfo []byte, metadata []byte) error {
	if err := t.ts.IdentityProvider().RegisterRecipientIdentity(id); err != nil {
		return err
	}
	return t.ts.RegisterRecipientIdentity(id, auditInfo, metadata)
}

func (t *WalletManager) Wallet(identity view.Identity) *Wallet {
	w := t.ts.Wallet(identity)
	if w == nil {
		return nil
	}
	return &Wallet{w: w}
}

func (t *WalletManager) OwnerWallet(id string) *OwnerWallet {
	w := t.ts.OwnerWallet(id)
	if w == nil {
		return nil
	}
	return &OwnerWallet{w: w}
}

func (t *WalletManager) OwnerWalletByIdentity(identity view.Identity) *OwnerWallet {
	w := t.ts.OwnerWalletByIdentity(identity)
	if w == nil {
		return nil
	}
	return &OwnerWallet{w: w}
}

func (t *WalletManager) IssuerWallet(id string) *IssuerWallet {
	w := t.ts.IssuerWallet(id)
	if w == nil {
		return nil
	}
	return &IssuerWallet{w: w}
}

func (t *WalletManager) IssuerWalletByIdentity(identity view.Identity) *IssuerWallet {
	w := t.ts.IssuerWalletByIdentity(identity)
	if w == nil {
		return nil
	}
	return &IssuerWallet{w: w}
}

func (t *WalletManager) AuditorWallet(id string) *AuditorWallet {
	w := t.ts.AuditorWallet(id)
	if w == nil {
		return nil
	}
	return &AuditorWallet{w: w}
}

func (t *WalletManager) CertifierWallet(id string) *CertifierWallet {
	w := t.ts.CertifierWallet(id)
	if w == nil {
		return nil
	}
	return &CertifierWallet{w: w}
}

func (t *WalletManager) CertifierWalletByIdentity(identity view.Identity) *CertifierWallet {
	w := t.ts.CertifierWalletByIdentity(identity)
	if w == nil {
		return nil
	}
	return &CertifierWallet{w: w}
}

type Signer interface {
	// Sign signs message bytes and returns the signature or an error on failure.
	Sign(message []byte) ([]byte, error)
}

type Wallet struct {
	w api2.Wallet
}

func (w *Wallet) ID() string {
	return w.w.ID()
}

func (w *Wallet) Contains(identity view.Identity) bool {
	return w.w.Contains(identity)
}

func (w *Wallet) ContainsToken(token *token2.UnspentToken) bool {
	return w.ContainsToken(token)
}

type AuditorWallet struct {
	w api2.AuditorWallet
}

func (a *AuditorWallet) ID() string {
	return a.w.ID()
}

func (a *AuditorWallet) Contains(identity view.Identity) bool {
	return a.w.Contains(identity)
}

func (a *AuditorWallet) ContainsToken(token *token2.UnspentToken) bool {
	return a.w.ContainsToken(token)
}

func (a *AuditorWallet) GetAuditorIdentity() (view.Identity, error) {
	return a.w.GetAuditorIdentity()
}

func (a *AuditorWallet) GetSigner(id view.Identity) (api2.Signer, error) {
	return a.w.GetSigner(id)
}

type CertifierWallet struct {
	w api2.CertifierWallet
}

func (a *CertifierWallet) ID() string {
	return a.w.ID()
}

func (a *CertifierWallet) Contains(identity view.Identity) bool {
	return a.w.Contains(identity)
}

func (a *CertifierWallet) ContainsToken(token *token2.UnspentToken) bool {
	return a.w.ContainsToken(token)
}

func (a *CertifierWallet) GetCertifierIdentity() (view.Identity, error) {
	return a.w.GetCertifierIdentity()
}

func (a *CertifierWallet) GetSigner(id view.Identity) (api2.Signer, error) {
	return a.w.GetSigner(id)
}

type OwnerWallet struct {
	w api2.OwnerWallet
}

func (o *OwnerWallet) ID() string {
	return o.w.ID()
}

func (o *OwnerWallet) Contains(identity view.Identity) bool {
	return o.w.Contains(identity)
}

func (o *OwnerWallet) ContainsToken(token *token2.UnspentToken) bool {
	return o.w.ContainsToken(token)
}

func (o *OwnerWallet) GetRecipientIdentity() (view.Identity, error) {
	return o.w.GetRecipientIdentity()
}

func (o *OwnerWallet) GetAuditInfo(id view.Identity) ([]byte, error) {
	return o.w.GetAuditInfo(id)
}

func (o *OwnerWallet) GetSigner(identity view.Identity) (api2.Signer, error) {
	return o.w.GetSigner(identity)
}

func (o *OwnerWallet) GetTokenMetadata(token []byte) ([]byte, error) {
	return o.w.GetTokenMetadata(token)
}

// ListUnspentTokens returns a list of unspent tokens owned by identities in this wallet and filtered by the passed options.
func (o *OwnerWallet) ListUnspentTokens(opts ...ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := compileListTokensOption(opts...)
	if err != nil {
		return nil, err
	}
	return o.w.ListTokens(compiledOpts)
}

type IssuerWallet struct {
	w api2.IssuerWallet
}

func (i *IssuerWallet) ID() string {
	return i.w.ID()
}

func (i *IssuerWallet) Contains(identity view.Identity) bool {
	return i.w.Contains(identity)
}

func (w *IssuerWallet) ContainsToken(token *token2.UnspentToken) bool {
	return w.Contains(token.Owner.Raw)
}

func (i *IssuerWallet) GetIssuerIdentity(tokenType string) (view.Identity, error) {
	return i.w.GetIssuerIdentity(tokenType)
}

func (i *IssuerWallet) GetSigner(identity view.Identity) (Signer, error) {
	return i.w.GetSigner(identity)
}

// ListIssuedTokens returns the list of tokens issued by identities in this wallet and filter by the passed options
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
