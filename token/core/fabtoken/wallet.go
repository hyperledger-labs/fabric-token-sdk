/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabtoken

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func (s *Service) RegisterOwnerWallet(id string, typ string, path string) error {
	return s.IP.RegisterOwnerWallet(id, typ, path)
}

func (s *Service) RegisterRecipientIdentity(id view.Identity, auditInfo []byte, metadata []byte) error {
	logger.Debugf("register recipient identity [%s] with audit info [%s]", id.String(), hash.Hashable(auditInfo).String())

	// recognize identity and register it
	v, err := s.Deserializer.GetOwnerVerifier(id)
	if err != nil {
		return errors.Wrapf(err, "failed getting verifier for [%s]", id)
	}
	if err := view2.GetSigService(s.SP).RegisterVerifier(id, v); err != nil {
		return errors.Wrapf(err, "failed registering verifier for [%s]", id)
	}
	if err := view2.GetSigService(s.SP).RegisterAuditInfo(id, auditInfo); err != nil {
		return errors.Wrapf(err, "failed registering audit info for [%s]", id)
	}

	return nil
}

func (s *Service) RegisterAuditInfo(id view.Identity, auditInfo []byte) error {
	if err := view2.GetSigService(s.SP).RegisterAuditInfo(id, auditInfo); err != nil {
		return err
	}
	return nil
}

func (s *Service) GetAuditInfo(id view.Identity) ([]byte, error) {
	return view2.GetSigService(s.SP).GetAuditInfo(id)
}

func (s *Service) GetEnrollmentID(auditInfo []byte) (string, error) {
	return string(auditInfo), nil
}

func (s *Service) Wallet(identity view.Identity) driver.Wallet {
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

func (s *Service) OwnerWalletByIdentity(identity view.Identity) driver.OwnerWallet {
	return s.OwnerWalletByID(identity)
}

func (s *Service) OwnerWallet(walletID string) driver.OwnerWallet {
	return s.OwnerWalletByID(walletID)
}

func (s *Service) OwnerWalletByID(id interface{}) driver.OwnerWallet {
	s.WalletsLock.Lock()
	defer s.WalletsLock.Unlock()

	// check if there is already a wallet
	identity, walletID := s.IP.LookupIdentifier(driver.OwnerRole, id)
	wID := s.walletID(walletID)
	for _, w := range s.OwnerWallets {
		if w.Contains(identity) || w.ID() == wID {
			logger.Debugf("found owner wallet [%s]", identity.String())
			return w
		}
	}

	// Create the wallet
	if idInfo := s.IP.GetIdentityInfo(driver.OwnerRole, walletID); idInfo != nil {
		id, err := idInfo.GetIdentity()
		if err != nil {
			panic(err)
		}
		wrappedID, err := s.wrapWalletIdentity(id)
		if err != nil {
			panic(err)
		}

		w := newOwnerWallet(s, id, wrappedID, wID)
		s.OwnerWallets = append(s.OwnerWallets, w)
		logger.Debugf("created owner wallet [%s:%s]", idInfo.ID, walletID)
		return w
	}

	return nil
}

func (s *Service) IssuerWallet(id string) driver.IssuerWallet {
	return s.issuerWallet(id)
}

func (s *Service) IssuerWalletByIdentity(id view.Identity) driver.IssuerWallet {
	return s.issuerWallet(id)
}

func (s *Service) issuerWallet(id interface{}) driver.IssuerWallet {
	s.WalletsLock.Lock()
	defer s.WalletsLock.Unlock()

	// check if there is already a wallet
	identity, walletID := s.IP.LookupIdentifier(driver.IssuerRole, id)
	for _, w := range s.IssuerWallets {
		if w.Contains(identity) || w.ID() == walletID {
			logger.Debugf("found issuer wallet [%s]", identity.String())
			return w
		}
	}

	// Create the wallet
	if idInfo := s.IP.GetIdentityInfo(driver.IssuerRole, walletID); idInfo != nil {
		id, err := idInfo.GetIdentity()
		if err != nil {
			panic(err)
		}
		w := newIssuerWallet(s, walletID, id)
		s.IssuerWallets = append(s.IssuerWallets, w)
		logger.Debugf("created issuer wallet [%s]", identity.String())
		return w
	}

	logger.Debugf("no issuer wallet found for [%s]", identity.String())
	return nil
}

func (s *Service) AuditorWallet(id string) driver.AuditorWallet {
	return s.auditorWallet(id)
}

func (s *Service) AuditorWalletByIdentity(id view.Identity) driver.AuditorWallet {
	return s.auditorWallet(id)
}

func (s *Service) auditorWallet(id interface{}) driver.AuditorWallet {
	s.WalletsLock.Lock()
	defer s.WalletsLock.Unlock()

	// check if there is already a wallet
	identity, walletID := s.IP.LookupIdentifier(driver.AuditorRole, id)
	for _, w := range s.AuditorWallets {
		if w.Contains(identity) || w.ID() == walletID {
			logger.Debugf("found auditor wallet [%s]", identity.String())
			return w
		}
	}

	// Create the wallet
	if idInfo := s.IP.GetIdentityInfo(driver.AuditorRole, walletID); idInfo != nil {
		id, err := idInfo.GetIdentity()
		if err != nil {
			panic(err)
		}
		w := newAuditorWallet(s, walletID, id)
		s.AuditorWallets = append(s.AuditorWallets, w)
		logger.Debugf("created auditor wallet [%s]", identity.String())
		return w
	}

	logger.Debugf("no auditor wallet found for [%s]", identity.String())
	return nil
}

func (s *Service) CertifierWallet(id string) driver.CertifierWallet {
	return nil
}

func (s *Service) CertifierWalletByIdentity(id view.Identity) driver.CertifierWallet {
	return nil
}

func (s *Service) wrapWalletIdentity(id view.Identity) (view.Identity, error) {
	raw, err := identity.MarshallRawOwner(&identity.RawOwner{Type: identity.SerializedIdentityType, Identity: id})
	if err != nil {
		return nil, err
	}
	if err := s.IdentityProvider().Bind(raw, id); err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *Service) walletID(id string) string {
	return s.Channel + s.Namespace + id
}

type ownerWallet struct {
	tokenService *Service
	id           string
	identity     view.Identity
	wrappedID    view.Identity
}

func newOwnerWallet(tokenService *Service, identity, wrappedID view.Identity, id string) *ownerWallet {
	return &ownerWallet{
		tokenService: tokenService,
		id:           id,
		identity:     identity,
		wrappedID:    wrappedID,
	}
}

func (w *ownerWallet) ID() string {
	return w.id
}

func (w *ownerWallet) Contains(identity view.Identity) bool {
	return w.identity.Equal(identity) || w.wrappedID.Equal(identity)
}

func (w *ownerWallet) ContainsToken(token *token.UnspentToken) bool {
	return w.Contains(token.Owner.Raw)
}

func (w *ownerWallet) GetRecipientIdentity() (view.Identity, error) {
	return w.wrappedID, nil
}

func (w *ownerWallet) GetAuditInfo(id view.Identity) ([]byte, error) {
	return w.tokenService.IP.GetAuditInfo(id)
}

func (w *ownerWallet) GetTokenMetadata(id view.Identity) ([]byte, error) {
	return nil, nil
}

func (w *ownerWallet) GetSigner(identity view.Identity) (driver.Signer, error) {
	if !w.wrappedID.Equal(identity) {
		return nil, errors.Errorf("identity does not belong to this wallet [%s]", identity.String())
	}

	si, err := w.tokenService.IP.GetSigner(w.wrappedID)
	if err != nil {
		return nil, err
	}
	return si, err
}

func (w *ownerWallet) ListTokens(opts *driver.ListTokensOptions) (*token.UnspentTokens, error) {
	logger.Debugf("wallet: list tokens, type [%s]", opts.TokenType)
	source, err := w.tokenService.ListTokens()
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}

	unspentTokens := &token.UnspentTokens{}
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
	tokenService *Service
	id           string
	identity     view.Identity
}

func newIssuerWallet(tokenService *Service, id string, identity view.Identity) *issuerWallet {
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

func (w *issuerWallet) ContainsToken(token *token.UnspentToken) bool {
	return w.Contains(token.Owner.Raw)
}

func (w *issuerWallet) GetIssuerIdentity(tokenType string) (view.Identity, error) {
	return w.identity, nil
}

func (w *issuerWallet) GetSigner(identity view.Identity) (driver.Signer, error) {
	if !w.Contains(identity) {
		return nil, errors.Errorf("failed getting signer, the passed identity [%s] does not belong to this wallet [%s]", identity, w.ID())
	}
	si, err := w.tokenService.IP.GetSigner(identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting issuer signer for identity [%s] in wallet [%s]", identity, w.identity)
	}
	return si, nil
}

func (w *issuerWallet) HistoryTokens(opts *driver.ListTokensOptions) (*token.IssuedTokens, error) {
	logger.Debugf("issuer wallet [%s]: history tokens, type [%d]", w.ID(), opts.TokenType)
	source, err := w.tokenService.HistoryIssuedTokens()
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
	tokenService *Service
	id           string
	identity     view.Identity
}

func newAuditorWallet(tokenService *Service, id string, identity view.Identity) *auditorWallet {
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

func (w *auditorWallet) ContainsToken(token *token.UnspentToken) bool {
	return w.Contains(token.Owner.Raw)
}

func (w *auditorWallet) GetAuditorIdentity() (view.Identity, error) {
	return w.identity, nil
}

func (w *auditorWallet) GetSigner(id view.Identity) (driver.Signer, error) {
	if !w.Contains(id) {
		return nil, errors.Errorf("identity does not belong to this wallet [%s]", id.String())
	}

	si, err := w.tokenService.IP.GetSigner(w.identity)
	if err != nil {
		return nil, err
	}
	return si, err
}
