/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"fmt"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

func (s *Service) RegisterOwnerWallet(id string, path string) error {
	return s.IP.RegisterOwnerWallet(id, path)
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
	return s.IP.GetEnrollmentID(auditInfo)
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
	s.OwnerWalletsRegistry.Lock()
	defer s.OwnerWalletsRegistry.Unlock()

	// check if there is already a wallet
	w, idInfo, wID, err := s.OwnerWalletsRegistry.Lookup(id)
	if err != nil {
		logger.Errorf("failed to lookup identity for owner wallet [%s]", err)
		return nil
	}
	if w != nil {
		return w.(driver.OwnerWallet)
	}

	// Create the wallet
	idInfoIdentity, _, err := idInfo.Get()
	if err != nil {
		logger.Errorf("failed to get owner wallet identity for [%s]: %s", wID, err)
		return nil
	}
	wrappedID, err := s.wrapWalletIdentity(idInfoIdentity)
	if err != nil {
		logger.Errorf("failed to wrap owner wallet identity for [%s]: %s", wID, err)
		return nil
	}

	newWallet := newOwnerWallet(s, idInfoIdentity, wrappedID, wID, idInfo)
	s.OwnerWalletsRegistry.RegisterWallet(wID, newWallet)
	if err := s.OwnerWalletsRegistry.RegisterIdentity(wrappedID, wID); err != nil {
		panic(fmt.Sprintf("programming error, failed to register recipient identity [%s]", err))
	}
	logger.Debugf("created owner wallet [%s:%s]", idInfo.ID, wID)
	return newWallet
}

func (s *Service) IssuerWallet(id string) driver.IssuerWallet {
	return s.issuerWallet(id)
}

func (s *Service) IssuerWalletByIdentity(id view.Identity) driver.IssuerWallet {
	return s.issuerWallet(id)
}

func (s *Service) issuerWallet(id interface{}) driver.IssuerWallet {
	s.IssuerWalletsRegistry.Lock()
	defer s.IssuerWalletsRegistry.Unlock()

	// check if there is already a wallet
	w, idInfo, wID, err := s.IssuerWalletsRegistry.Lookup(id)
	if err != nil {
		logger.Errorf("failed to lookup identity for issuer wallet [%s]", err)
		return nil
	}
	if w != nil {
		return w.(driver.IssuerWallet)
	}

	// Create the wallet
	idInfoIdentity, _, err := idInfo.Get()
	if err != nil {
		logger.Errorf("failed to get issuer wallet identity for [%s]: %s", wID, err)
		return nil
	}
	newWallet := newIssuerWallet(s, wID, idInfoIdentity)
	s.IssuerWalletsRegistry.RegisterWallet(wID, newWallet)
	if err := s.IssuerWalletsRegistry.RegisterIdentity(idInfoIdentity, wID); err != nil {
		panic(fmt.Sprintf("programming error, failed to register recipient identity [%s]", err))
	}
	logger.Debugf("created issuer wallet [%s]", wID)
	return newWallet
}

func (s *Service) AuditorWallet(id string) driver.AuditorWallet {
	return s.auditorWallet(id)
}

func (s *Service) AuditorWalletByIdentity(id view.Identity) driver.AuditorWallet {
	return s.auditorWallet(id)
}

func (s *Service) auditorWallet(id interface{}) driver.AuditorWallet {
	s.AuditorWalletsRegistry.Lock()
	defer s.AuditorWalletsRegistry.Unlock()

	// check if there is already a wallet
	w, idInfo, wID, err := s.AuditorWalletsRegistry.Lookup(id)
	if err != nil {
		logger.Errorf("failed to lookup identity for auditor wallet [%s]", err)
		return nil
	}
	if w != nil {
		return w.(driver.AuditorWallet)
	}

	// Create the wallet
	idInfoIdentity, _, err := idInfo.Get()
	if err != nil {
		logger.Errorf("failed to get auditor wallet identity for [%s:%s]: %s", wID, id, err)
		return nil
	}
	newWallet := newAuditorWallet(s, wID, idInfoIdentity)
	s.AuditorWalletsRegistry.RegisterWallet(wID, newWallet)
	if err := s.AuditorWalletsRegistry.RegisterIdentity(idInfoIdentity, wID); err != nil {
		panic(fmt.Sprintf("programming error, failed to register recipient identity [%s]", err))
	}
	logger.Debugf("created auditor wallet [%s]", wID)
	return newWallet
}

func (s *Service) CertifierWallet(id string) driver.CertifierWallet {
	return nil
}

func (s *Service) CertifierWalletByIdentity(id view.Identity) driver.CertifierWallet {
	return nil
}

// SpentIDs returns the spend ids for the passed token ids
func (s *Service) SpentIDs(ids ...*token.ID) ([]string, error) {
	sIDs := make([]string, len(ids))
	var err error
	for i, id := range ids {
		sIDs[i], err = keys.CreateTokenKey(id.TxId, id.Index)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot compute spent id for [%v]", id)
		}
	}
	return sIDs, nil
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

type ownerWallet struct {
	tokenService *Service
	id           string
	identityInfo driver.IdentityInfo
	identity     view.Identity
	wrappedID    view.Identity
}

func newOwnerWallet(tokenService *Service, identity, wrappedID view.Identity, id string, identityInfo driver.IdentityInfo) *ownerWallet {
	return &ownerWallet{
		tokenService: tokenService,
		id:           id,
		identity:     identity,
		wrappedID:    wrappedID,
		identityInfo: identityInfo,
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
	it, err := w.tokenService.QE.UnspentTokensIteratorBy(w.id, opts.TokenType)
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
	it, err := w.tokenService.QE.UnspentTokensIteratorBy(w.id, opts.TokenType)
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}
	return it, nil
}

func (w *ownerWallet) EnrollmentID() string {
	return w.identityInfo.EnrollmentID()
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
