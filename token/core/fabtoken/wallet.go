/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabtoken

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	api2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func (s *service) Wallet(identity view.Identity) api2.Wallet {
	w := s.OwnerWalletByIdentity(identity)
	if w != nil {
		return w
	}
	iw := s.IssuerWalletByIdentity(identity)
	if iw != nil {
		return iw
	}
	return nil
}

func (s *service) OwnerWalletByIdentity(identity view.Identity) api2.OwnerWallet {
	return s.ownerWallet(identity)
}

func (s *service) OwnerWallet(walletID string) api2.OwnerWallet {
	return s.ownerWallet(walletID)
}

func (s *service) ownerWallet(id interface{}) api2.OwnerWallet {
	s.walletsLock.Lock()
	defer s.walletsLock.Unlock()

	// check if there is already a wallet
	identity, walletID := s.identityProvider.LookupIdentifier(api2.OwnerRole, id)
	for _, w := range s.ownerWallets {
		if w.Contains(identity) || w.ID() == walletID {
			logger.Debugf("found owner wallet [%s]", identity.String())
			return w
		}
	}

	// Create the wallet
	if idInfo := s.identityProvider.GetIdentityInfo(api2.OwnerRole, walletID); idInfo != nil {
		id, err := idInfo.GetIdentity()
		if err != nil {
			panic(err)
		}
		id, err = s.wrapWalletIdentity(id)
		if err != nil {
			panic(err)
		}
		w := newOwnerWallet(s, idInfo.ID, id)
		s.ownerWallets = append(s.ownerWallets, w)
		logger.Debugf("created owner wallet [%s]", walletID)
		return w
	}

	return nil
}

func (s *service) IssuerWallet(id string) api2.IssuerWallet {
	return s.issuerWallet(id)
}

func (s *service) IssuerWalletByIdentity(id view.Identity) api2.IssuerWallet {
	return s.issuerWallet(id)
}

func (s *service) issuerWallet(id interface{}) api2.IssuerWallet {
	s.walletsLock.Lock()
	defer s.walletsLock.Unlock()

	// check if there is already a wallet
	identity, walletID := s.identityProvider.LookupIdentifier(api2.IssuerRole, id)
	for _, w := range s.issuerWallets {
		if w.Contains(identity) || w.ID() == walletID {
			logger.Debugf("found issuer wallet [%s]", identity.String())
			return w
		}
	}

	// Create the wallet
	if idInfo := s.identityProvider.GetIdentityInfo(api2.IssuerRole, walletID); idInfo != nil {
		id, err := idInfo.GetIdentity()
		if err != nil {
			panic(err)
		}
		id, err = s.wrapWalletIdentity(id)
		if err != nil {
			panic(err)
		}
		w := newIssuerWallet(s, idInfo.ID, id)
		s.issuerWallets = append(s.issuerWallets, w)
		logger.Debugf("created issuer wallet [%s]", identity.String())
		return w
	}

	logger.Debugf("no issuer wallet found for [%s]", identity.String())
	return nil
}

func (s *service) AuditorWallet(id string) api2.AuditorWallet {
	return s.auditorWallet(id)
}

func (s *service) AuditorWalletByIdentity(id view.Identity) api2.AuditorWallet {
	return s.auditorWallet(id)
}

func (s *service) auditorWallet(id interface{}) api2.AuditorWallet {
	s.walletsLock.Lock()
	defer s.walletsLock.Unlock()

	// check if there is already a wallet
	identity, walletID := s.identityProvider.LookupIdentifier(api2.AuditorRole, id)
	for _, w := range s.auditorWallets {
		if w.Contains(identity) || w.ID() == walletID {
			logger.Debugf("found auditor wallet [%s]", identity.String())
			return w
		}
	}

	// Create the wallet
	if idInfo := s.identityProvider.GetIdentityInfo(api2.AuditorRole, walletID); idInfo != nil {
		id, err := idInfo.GetIdentity()
		if err != nil {
			panic(err)
		}
		w := newAuditorWallet(s, idInfo.ID, id)
		s.auditorWallets = append(s.auditorWallets, w)
		logger.Debugf("created auditor wallet [%s]", identity.String())
		return w
	}

	logger.Debugf("no auditor wallet found for [%s]", identity.String())
	return nil
}

func (s *service) CertifierWallet(id string) api2.CertifierWallet {
	return nil
}

func (s *service) CertifierWalletByIdentity(id view.Identity) api2.CertifierWallet {
	return nil
}

func (s *service) wrapWalletIdentity(id view.Identity) (view.Identity, error) {
	ro := &RawOwner{Type: SerializedIdentityType, Identity: id}
	raw, err := json.Marshal(ro)
	if err != nil {
		return nil, err
	}
	if err := s.IdentityProvider().Bind(raw, id); err != nil {
		return nil, err
	}
	return raw, nil
}

type ownerWallet struct {
	tokenService *service
	id           string
	identity     view.Identity
}

func newOwnerWallet(tokenService *service, id string, identity view.Identity) *ownerWallet {
	return &ownerWallet{
		tokenService: tokenService,
		id:           id,
		identity:     identity,
	}
}

func (w *ownerWallet) ID() string {
	return w.id
}

func (w *ownerWallet) Contains(identity view.Identity) bool {
	return w.identity.Equal(identity)
}

func (w *ownerWallet) GetRecipientIdentity() (view.Identity, error) {
	return w.identity, nil
}

func (w *ownerWallet) GetAuditInfo(id view.Identity) ([]byte, error) {
	return w.tokenService.identityProvider.GetAuditInfo(id)
}

func (w *ownerWallet) GetTokenMetadata(id view.Identity) ([]byte, error) {
	return nil, nil
}

func (w *ownerWallet) GetSigner(identity view.Identity) (api2.Signer, error) {
	if !w.identity.Equal(identity) {
		return nil, errors.Errorf("identity does not belong to this wallet [%s]", identity.String())
	}

	si, err := w.tokenService.identityProvider.GetSigner(w.identity)
	if err != nil {
		return nil, err
	}
	return si, err
}

func (w *ownerWallet) ListTokens(opts *api2.ListTokensOptions) (*token2.UnspentTokens, error) {
	logger.Debugf("wallet: list tokens, type [%s]", opts.TokenType)
	source, err := w.tokenService.ListTokens()
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}

	unspentTokens := &token2.UnspentTokens{}
	for _, t := range source.Tokens {
		if len(opts.TokenType) != 0 && t.Type != opts.TokenType {
			logger.Debugf("wallet: discarding token of type [%s]!=[%s]", t.Type, opts.TokenType)
			continue
		}

		if !w.Contains(t.Owner.Raw) {
			logger.Debugf("wallet: discarding token, owner does not belong to this wallet")
			continue
		}

		logger.Debugf("wallet: adding token of type [%s], quantity [%s]", t.Type, t.Quantity)
		unspentTokens.Tokens = append(unspentTokens.Tokens, t)
	}
	logger.Debugf("wallet: list tokens done, found [%d] unspent tokens", len(unspentTokens.Tokens))

	return unspentTokens, nil
}

type issuerWallet struct {
	tokenService *service
	id           string
	identity     view.Identity
}

func newIssuerWallet(tokenService *service, id string, identity view.Identity) *issuerWallet {
	return &issuerWallet{
		tokenService: tokenService,
		id:           id,
		identity:     identity,
	}
}

func (w *issuerWallet) ID() string {
	return w.id
}

func (w *issuerWallet) Contains(identity view.Identity) bool {
	return w.identity.Equal(identity)
}

func (w *issuerWallet) GetIssuerIdentity(tokenType string) (view.Identity, error) {
	return w.identity, nil
}

func (w *issuerWallet) GetSigner(identity view.Identity) (api2.Signer, error) {
	if !w.Contains(identity) {
		return nil, errors.Errorf("failed getting signer, the passed identity [%s] does not belong to this wallet [%s]", identity, w.ID())
	}
	si, err := w.tokenService.identityProvider.GetSigner(identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting issuer signer for identity [%s] in wallet [%s]", identity, w.identity)
	}
	return si, nil
}

func (w *issuerWallet) HistoryTokens(opts *api2.ListTokensOptions) (*token2.IssuedTokens, error) {
	logger.Debugf("issuer wallet [%s]: history tokens, type [%d]", w.ID(), opts.TokenType)
	source, err := w.tokenService.HistoryIssuedTokens()
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}

	unspentTokens := &token2.IssuedTokens{}
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
	tokenService *service
	id           string
	identity     view.Identity
}

func newAuditorWallet(tokenService *service, id string, identity view.Identity) *auditorWallet {
	return &auditorWallet{
		tokenService: tokenService,
		id:           id,
		identity:     identity,
	}
}

func (w *auditorWallet) ID() string {
	return w.id
}

func (w *auditorWallet) Contains(identity view.Identity) bool {
	return w.identity.Equal(identity)
}

func (w *auditorWallet) GetAuditorIdentity() (view.Identity, error) {
	return w.identity, nil
}

func (w *auditorWallet) GetSigner(id view.Identity) (api2.Signer, error) {
	if !w.Contains(id) {
		return nil, errors.Errorf("identity does not belong to this wallet [%s]", id.String())
	}

	si, err := w.tokenService.identityProvider.GetSigner(w.identity)
	if err != nil {
		return nil, err
	}
	return si, err
}
