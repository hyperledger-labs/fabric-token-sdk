/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type AuditorWallet struct {
	IdentityProvider driver.IdentityProvider
	id               string
	identity         view.Identity
}

func NewAuditorWallet(IdentityProvider driver.IdentityProvider, id string, identity view.Identity) *AuditorWallet {
	return &AuditorWallet{
		IdentityProvider: IdentityProvider,
		id:               id,
		identity:         identity,
	}
}

func (w *AuditorWallet) ID() string {
	return w.id
}

func (w *AuditorWallet) Contains(identity view.Identity) bool {
	return w.identity.Equal(identity)
}

func (w *AuditorWallet) ContainsToken(token *token.UnspentToken) bool {
	return w.Contains(token.Owner.Raw)
}

func (w *AuditorWallet) GetAuditorIdentity() (view.Identity, error) {
	return w.identity, nil
}

func (w *AuditorWallet) GetSigner(id view.Identity) (driver.Signer, error) {
	if !w.Contains(id) {
		return nil, errors.Errorf("identity [%s] does not belong to this wallet [%s]", id, w.ID())
	}

	si, err := w.IdentityProvider.GetSigner(w.identity)
	if err != nil {
		return nil, err
	}
	return si, err
}

type IssuerTokenVault interface {
	ListHistoryIssuedTokens() (*token.IssuedTokens, error)
}

type IssuerWallet struct {
	Logger           *flogging.FabricLogger
	IdentityProvider driver.IdentityProvider
	TokenVault       IssuerTokenVault
	id               string
	identity         view.Identity
}

func NewIssuerWallet(Logger *flogging.FabricLogger, IdentityProvider driver.IdentityProvider, TokenVault IssuerTokenVault, id string, identity view.Identity) *IssuerWallet {
	return &IssuerWallet{
		Logger:           Logger,
		IdentityProvider: IdentityProvider,
		TokenVault:       TokenVault,
		id:               id,
		identity:         identity,
	}
}

func (w *IssuerWallet) ID() string {
	return w.id
}

func (w *IssuerWallet) Contains(identity view.Identity) bool {
	return w.identity.Equal(identity)
}

func (w *IssuerWallet) ContainsToken(token *token.UnspentToken) bool {
	return w.Contains(token.Owner.Raw)
}

func (w *IssuerWallet) GetIssuerIdentity(tokenType string) (view.Identity, error) {
	return w.identity, nil
}

func (w *IssuerWallet) GetSigner(identity view.Identity) (driver.Signer, error) {
	if !w.Contains(identity) {
		return nil, errors.Errorf("failed getting signer, the passed identity [%s] does not belong to this wallet [%s]", identity, w.ID())
	}
	si, err := w.IdentityProvider.GetSigner(identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting issuer signer for identity [%s] in wallet [%s]", identity, w.identity)
	}
	return si, nil
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

		if !w.Contains(t.Issuer.Raw) {
			w.Logger.Debugf("issuer wallet [%s]: discarding token, issuer does not belong to wallet", w.ID())
			continue
		}

		w.Logger.Debugf("issuer wallet [%s]: adding token of type [%s], quantity [%s]", w.ID(), t.Type, t.Quantity)
		unspentTokens.Tokens = append(unspentTokens.Tokens, t)
	}
	w.Logger.Debugf("issuer wallet [%s]: history tokens done, found [%d] issued tokens", w.ID(), len(unspentTokens.Tokens))

	return unspentTokens, nil
}
