/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"fmt"
	"runtime/debug"

	math "github.com/IBM/mathlib"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/tms"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue/anonym"
	api2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func (s *Service) IssuerIdentity(label string) (view.Identity, error) {
	logger.Debugf("searching issuer for [%s] at [%s]", label, s.Channel)

	pp := s.PublicParams()
	for _, issuer := range s.Issuers {
		logger.Debugf("issuer for [%s] at [%s]", issuer.label, s.Channel)
		if issuer.label == label {
			if issuer.fID != nil {
				return issuer.fID, nil
			}

			logger.Debugf("issuer for [%s] at [%s] found (index %d)", issuer.label, s.Channel, issuer.index)

			witness := anonym.NewWitness(issuer.sk, nil, nil, nil, nil, issuer.index)
			signer := anonym.NewSigner(witness, nil, nil, 0, pp.ZKATPedParams, math.Curves[pp.Curve])

			fID, err := signer.ToUniqueIdentifier()
			if err != nil {
				return nil, errors.Wrapf(err, "failed serializing signer for [%s]", label)
			}

			if err := view2.GetSigService(s.SP).RegisterSigner(fID, signer, signer); err != nil {
				return nil, errors.Wrapf(err, "failed registering signer for [%s]", label)
			}

			if err := view2.GetEndpointService(s.SP).Bind(view2.GetIdentityProvider(s.SP).DefaultIdentity(), fID); err != nil {
				return nil, errors.Wrapf(err, "failed binding to long term identity or [%s]", view.Identity(fID).UniqueID())
			}
			issuer.fID = fID

			return fID, nil
		}
	}

	// Fetch from vault if not found

	return nil, errors.Errorf("identity not found at [%s]", label)
}

func (s *Service) GenerateIssuerKeyPair(tokenType string) (api2.Key, api2.Key, error) {
	return anonym.GenerateKeyPair(tokenType, s.PublicParams())
}

func (s *Service) RegisterIssuer(label string, sk api2.Key, pk api2.Key) error {
	if err := s.FetchPublicParams(); err != nil {
		return errors.WithMessagef(err, "failed fetching public params")
	}

	// search for pk
	ip, err := s.PublicParams().GetIssuingPolicy()
	if err != nil {
		return errors.WithMessagef(err, "failed parsing issuing policy")
	}

	_pk := pk.(*math.G1)

	for index, issuer := range ip.Issuers {
		if issuer.Equals(_pk) {
			s.Issuers = append(s.Issuers, &struct {
				label string
				index int
				sk    *math.Zr
				pk    *math.G1
				fID   view.Identity
			}{label: label, index: index, sk: sk.(*math.Zr), pk: _pk, fID: nil})

			logger.Debugf("registered issuer for [%s] at [%s], fetching public params", label, s.Channel)

			return nil
		}
	}

	return errors.Errorf("public key not found in public parameters")
}

func (s *Service) GetAuditInfo(id view.Identity) ([]byte, error) {
	return s.identityProvider.GetAuditInfo(id)
}

func (s *Service) GetEnrollmentID(auditInfo []byte) (string, error) {
	return s.identityProvider.GetEnrollmentID(auditInfo)
}

func (s *Service) registerIssuerSigner(signer SigningIdentity) error {
	fID, err := signer.Serialize()
	if err != nil {
		return errors.Wrapf(err, "failed serializing signer")
	}

	if err := view2.GetSigService(s.SP).RegisterSigner(fID, signer, signer); err != nil {
		return errors.Wrapf(err, "failed registering signer for [%s]", view.Identity(fID).UniqueID())
	}

	if err := view2.GetEndpointService(s.SP).Bind(view2.GetIdentityProvider(s.SP).DefaultIdentity(), fID); err != nil {
		return errors.Wrapf(err, "failed binding to long term identity or [%s]", view.Identity(fID).UniqueID())
	}

	return nil
}

func (s *Service) RegisterRecipientIdentity(id view.Identity, auditInfo []byte, metadata []byte) error {
	logger.Debugf("register recipient identity [%s] with audit info [%s]", id.String(), hash.Hashable(auditInfo).String())

	// recognize identity and register it
	d, err := s.Deserializer()
	if err != nil {
		return errors.Wrap(err, "failed getting deserializer")
	}
	v, err := d.GetOwnerVerifier(id)
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

func (s *Service) Wallet(identity view.Identity) api2.Wallet {
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

func (s *Service) OwnerWallet(walletID string) api2.OwnerWallet {
	return s.OwnerWalletByID(walletID)
}

func (s *Service) OwnerWalletByIdentity(identity view.Identity) api2.OwnerWallet {
	return s.OwnerWalletByID(identity)
}

func (s *Service) OwnerWalletByID(id interface{}) api2.OwnerWallet {
	s.WalletsLock.Lock()
	defer s.WalletsLock.Unlock()

	// check if there is already a wallet
	identity, walletID := s.identityProvider.LookupIdentifier(api2.OwnerRole, id)
	wID := s.walletID(walletID)
	for _, w := range s.OwnerWallets {
		if w.ID() == wID || (len(identity) != 0 && w.Contains(identity)) {
			logger.Debugf("found owner wallet [%s:%s:%s]", identity, walletID, w.ID())
			return w
		}
	}

	// Create the wallet
	if idInfo := s.identityProvider.GetIdentityInfo(api2.OwnerRole, walletID); idInfo != nil {
		w := newOwnerWallet(s, wID, idInfo)
		s.OwnerWallets = append(s.OwnerWallets, w)
		logger.Debugf("created owner wallet [%s:%s]", identity, walletID)
		return w
	}

	logger.Debugf("no owner wallet found for [%s:%s:%s] [%s]", id, identity, walletID, debug.Stack())
	return nil
}

func (s *Service) IssuerWallet(id string) api2.IssuerWallet {
	return s.issuerWallet(id)
}

func (s *Service) IssuerWalletByIdentity(id view.Identity) api2.IssuerWallet {
	return s.issuerWallet(id)
}

func (s *Service) issuerWallet(id interface{}) api2.IssuerWallet {
	s.WalletsLock.Lock()
	defer s.WalletsLock.Unlock()

	// check if there is already a wallet
	identity, walletID := s.identityProvider.LookupIdentifier(api2.IssuerRole, id)
	for _, w := range s.IssuerWallets {
		if w.Contains(identity) || w.ID() == walletID {
			logger.Debugf("found issuer wallet [%s:%s]", identity, walletID)
			return w
		}
	}

	// Create the wallet
	if idInfo := s.identityProvider.GetIdentityInfo(api2.IssuerRole, walletID); idInfo != nil {
		id, err := idInfo.GetIdentity()
		if err != nil {
			panic(err)
		}
		w := newIssuerWallet(s, walletID, id)
		s.IssuerWallets = append(s.IssuerWallets, w)
		logger.Debugf("created issuer wallet [%s:%s]", identity, walletID)
		return w
	}

	logger.Debugf("no issuer wallet found for [%s:%s]", identity, walletID)
	return nil
}

func (s *Service) AuditorWallet(id string) api2.AuditorWallet {
	return s.auditorWallet(id)
}

func (s *Service) AuditorWalletByIdentity(id view.Identity) api2.AuditorWallet {
	return s.auditorWallet(id)
}

func (s *Service) auditorWallet(id interface{}) api2.AuditorWallet {
	s.WalletsLock.Lock()
	defer s.WalletsLock.Unlock()

	// check if there is already a wallet
	identity, walletID := s.identityProvider.LookupIdentifier(api2.AuditorRole, id)
	for _, w := range s.AuditorWallets {
		if w.Contains(identity) || w.ID() == walletID {
			logger.Debugf("found auditor wallet [%s:%s]", identity, walletID)
			return w
		}
	}

	// Create the wallet
	if idInfo := s.identityProvider.GetIdentityInfo(api2.AuditorRole, walletID); idInfo != nil {
		id, err := idInfo.GetIdentity()
		if err != nil {
			panic(err)
		}
		w := newAuditorWallet(s, walletID, id)
		s.AuditorWallets = append(s.AuditorWallets, w)
		logger.Debugf("created auditor wallet [%s:%s]", identity, walletID)
		return w
	}

	logger.Debugf("no auditor wallet found for [%s:%s]", identity, walletID)
	return nil
}

func (s *Service) CertifierWallet(id string) api2.CertifierWallet {
	return nil
}

func (s *Service) CertifierWalletByIdentity(id view.Identity) api2.CertifierWallet {
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

type wallet struct {
	tokenService *Service
	id           string
	identityInfo *api2.IdentityInfo
	prefix       string
	cache        *tms.WalletIdentityCache
}

func newOwnerWallet(tokenService *Service, id string, identityInfo *api2.IdentityInfo) *wallet {
	w := &wallet{
		tokenService: tokenService,
		id:           id,
		identityInfo: identityInfo,
		prefix:       fmt.Sprintf("%s:%s:%s", tokenService.Channel, tokenService.Namespace, id),
	}
	w.cache = tms.NewWalletIdentityCache(w.getRecipientIdentity, 200)

	return w
}

func (w *wallet) ID() string {
	return w.id
}

func (w *wallet) Contains(identity view.Identity) bool {
	return w.existsRecipientIdentity(identity)
}

func (w *wallet) ContainsToken(token *token2.UnspentToken) bool {
	return w.Contains(token.Owner.Raw)
}

func (w *wallet) GetRecipientIdentity() (view.Identity, error) {
	return w.cache.Identity()
}

func (w *wallet) getRecipientIdentity() (view.Identity, error) {
	// Get a new pseudonym
	pseudonym, err := w.identityInfo.GetIdentity()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting recipient identity from wallet [%s]", w.ID())
	}

	pseudonym, err = w.tokenService.wrapWalletIdentity(pseudonym)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed wrapping recipient identity from wallet [%s]", w.ID())
	}

	// Register the pseudonym
	if err := w.putRecipientIdentity(pseudonym, []byte{}); err != nil {
		return nil, errors.WithMessagef(err, "failed storing recipient identity in wallet [%s]", w.ID())
	}
	return pseudonym, nil
}

func (w *wallet) GetAuditInfo(id view.Identity) ([]byte, error) {
	return w.tokenService.identityProvider.GetAuditInfo(id)
}

func (w *wallet) GetTokenMetadata(id view.Identity) ([]byte, error) {
	return nil, nil
}

func (w *wallet) GetSigner(identity view.Identity) (api2.Signer, error) {
	if !w.Contains(identity) {
		return nil, errors.Errorf("identity [%s] does not belong to this wallet [%s]", identity, w.ID())
	}

	si, err := w.tokenService.identityProvider.GetSigner(identity)
	if err != nil {
		return nil, err
	}
	return si, err
}

func (w *wallet) ListTokens(opts *api2.ListTokensOptions) (*token2.UnspentTokens, error) {
	logger.Debugf("wallet: list tokens, type [%s]", opts.TokenType)
	source, err := w.tokenService.QE.ListUnspentTokens()
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

func (w *wallet) existsRecipientIdentity(id view.Identity) bool {
	kvss := kvs.GetService(w.tokenService.SP)
	return kvss.Exists(w.prefix + id.Hash())
}

func (w *wallet) putRecipientIdentity(id view.Identity, meta []byte) error {
	kvss := kvs.GetService(w.tokenService.SP)
	if err := kvss.Put(w.prefix+id.Hash(), meta); err != nil {
		return err
	}
	return nil
}

type IssuerKeyPair struct {
	Pk view.Identity
	Sk interface{}
}

type issuerWallet struct {
	tokenService *Service

	id       string
	identity view.Identity
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

func (w *issuerWallet) ContainsToken(token *token2.UnspentToken) bool {
	return w.Contains(token.Owner.Raw)
}

func (w *issuerWallet) GetIssuerIdentity(tokenType string) (view.Identity, error) {
	return w.identity, nil
}

func (w *issuerWallet) GetSigner(identity view.Identity) (api2.Signer, error) {
	if !w.Contains(identity) {
		return nil, errors.Errorf("identity [%s] does not belong to this wallet [%s]", identity, w.ID())
	}
	si, err := w.tokenService.identityProvider.GetSigner(identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting signer for identity [%s] in wallet [%s]", identity, w.identity)
	}
	return si, nil
}

func (w *issuerWallet) HistoryTokens(opts *api2.ListTokensOptions) (*token2.IssuedTokens, error) {
	logger.Debugf("issuer wallet [%s]: history tokens, type [%d]", w.ID(), opts.TokenType)
	source, err := w.tokenService.QE.ListHistoryIssuedTokens()
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

func (w *auditorWallet) ContainsToken(token *token2.UnspentToken) bool {
	return w.Contains(token.Owner.Raw)
}

func (w *auditorWallet) GetAuditorIdentity() (view.Identity, error) {
	return w.identity, nil
}

func (w *auditorWallet) GetSigner(id view.Identity) (api2.Signer, error) {
	if !w.Contains(id) {
		return nil, errors.Errorf("identity [%s] does not belong to this wallet [%s]", id, w.ID())
	}

	si, err := w.tokenService.identityProvider.GetSigner(w.identity)
	if err != nil {
		return nil, err
	}
	return si, err
}
