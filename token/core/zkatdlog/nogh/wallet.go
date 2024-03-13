/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type WalletRegistry interface {
	WalletIDs() ([]string, error)
	Lock()
	Unlock()
	RegisterIdentity(id string, path string) error
	Lookup(id interface{}) (driver.Wallet, driver.IdentityInfo, string, error)
	RegisterWallet(id string, wallet driver.Wallet) error
	BindIdentity(identity view.Identity, eID string, wID string, meta any) error
	ContainsIdentity(i view.Identity, id string) bool
}

type WalletService struct {
	identityProvider     driver.IdentityProvider
	TokenVault           TokenVault
	PPM                  PublicParametersManager
	DeserializerProvider DeserializerProviderFunc
	ConfigManager        config.Manager

	OwnerWalletsRegistry   WalletRegistry
	IssuerWalletsRegistry  WalletRegistry
	AuditorWalletsRegistry WalletRegistry
}

func NewWalletService(
	identityProvider driver.IdentityProvider,
	tokenVault TokenVault,
	PPM PublicParametersManager,
	deserializerProvider DeserializerProviderFunc,
	configManager config.Manager,
	OwnerWalletRegistry WalletRegistry,
	IssuerWalletRegistry WalletRegistry,
	AuditorWalletRegistry WalletRegistry,
) *WalletService {
	return &WalletService{
		identityProvider:       identityProvider,
		TokenVault:             tokenVault,
		PPM:                    PPM,
		DeserializerProvider:   deserializerProvider,
		ConfigManager:          configManager,
		OwnerWalletsRegistry:   OwnerWalletRegistry,
		IssuerWalletsRegistry:  IssuerWalletRegistry,
		AuditorWalletsRegistry: AuditorWalletRegistry,
	}
}

func (s *WalletService) RegisterOwnerWallet(id string, path string) error {
	return s.OwnerWalletsRegistry.RegisterIdentity(id, path)
}

func (s *WalletService) RegisterIssuerWallet(id string, path string) error {
	return s.IssuerWalletsRegistry.RegisterIdentity(id, path)
}

func (s *WalletService) GetAuditInfo(id view.Identity) ([]byte, error) {
	return s.identityProvider.GetAuditInfo(id)
}

func (s *WalletService) GetEnrollmentID(auditInfo []byte) (string, error) {
	return s.identityProvider.GetEnrollmentID(auditInfo)
}

func (s *WalletService) GetRevocationHandler(auditInfo []byte) (string, error) {
	return s.identityProvider.GetRevocationHandler(auditInfo)
}

func (s *WalletService) RegisterRecipientIdentity(data *driver.RecipientData) error {
	if data == nil {
		return errors.Errorf("cannot recigest empty recipient")
	}
	logger.Debugf("register recipient identity [%s] with audit info [%s]", data.Identity.String(), hash.Hashable(data.AuditInfo).String())

	// recognize identity and register it
	d, err := s.Deserializer()
	if err != nil {
		return errors.Wrap(err, "failed getting deserializer")
	}

	// match identity and audit info
	err = d.Match(data.Identity, data.AuditInfo)
	if err != nil {
		return errors.Wrapf(err, "failed to match identity to audit infor for [%s:%s]", data.Identity, hash.Hashable(data.AuditInfo))
	}

	// register verifier and audit info
	v, err := d.GetOwnerVerifier(data.Identity)
	if err != nil {
		return errors.Wrapf(err, "failed getting verifier for [%s]", data.Identity)
	}
	if err := s.identityProvider.RegisterVerifier(data.Identity, v); err != nil {
		return errors.Wrapf(err, "failed registering verifier for [%s]", data.Identity)
	}
	if err := s.identityProvider.RegisterAuditInfo(data.Identity, data.AuditInfo); err != nil {
		return errors.Wrapf(err, "failed registering audit info for [%s]", data.Identity)
	}

	return nil
}

func (s *WalletService) Wallet(identity view.Identity) driver.Wallet {
	w, _ := s.OwnerWalletByIdentity(identity)
	if w != nil {
		return w
	}
	iw, _ := s.IssuerWalletByIdentity(identity)
	if iw != nil {
		return iw
	}
	return nil
}

func (s *WalletService) OwnerWalletIDs() ([]string, error) {
	return s.OwnerWalletsRegistry.WalletIDs()
}

func (s *WalletService) OwnerWallet(id string) (driver.OwnerWallet, error) {
	return s.OwnerWalletByID(id)
}

func (s *WalletService) OwnerWalletByIdentity(identity view.Identity) (driver.OwnerWallet, error) {
	return s.OwnerWalletByID(identity)
}

func (s *WalletService) OwnerWalletByID(id interface{}) (driver.OwnerWallet, error) {
	s.OwnerWalletsRegistry.Lock()
	defer s.OwnerWalletsRegistry.Unlock()

	// check if there is already a wallet
	w, idInfo, wID, err := s.OwnerWalletsRegistry.Lookup(id)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to lookup identity for owner wallet [%v]", id)
	}
	if w != nil {
		return w.(driver.OwnerWallet), nil
	}

	// Create the wallet
	newWallet, err := newOwnerWallet(s, wID, idInfo)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create new owner wallet [%s]", wID)
	}
	logger.Debugf("created owner wallet [%s]", wID)
	return newWallet, nil
}

func (s *WalletService) IssuerWallet(id string) (driver.IssuerWallet, error) {
	return s.issuerWallet(id)
}

func (s *WalletService) IssuerWalletByIdentity(identity view.Identity) (driver.IssuerWallet, error) {
	return s.issuerWallet(identity)
}

func (s *WalletService) issuerWallet(id interface{}) (driver.IssuerWallet, error) {
	s.IssuerWalletsRegistry.Lock()
	defer s.IssuerWalletsRegistry.Unlock()

	// check if there is already a wallet
	w, idInfo, wID, err := s.IssuerWalletsRegistry.Lookup(id)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to lookup identity for issuer wallet [%v]", id)
	}
	if w != nil {
		return w.(driver.IssuerWallet), nil
	}

	// Create the wallet
	idInfoIdentity, _, err := idInfo.Get()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get issuer wallet identity for [%s]", wID)
	}
	newWallet := newIssuerWallet(s, wID, idInfoIdentity)
	if err := s.IssuerWalletsRegistry.RegisterWallet(wID, newWallet); err != nil {
		return nil, errors.WithMessagef(err, "programming error, failed to register issuer wallet [%s]", wID)
	}
	if err := s.IssuerWalletsRegistry.BindIdentity(idInfoIdentity, idInfo.EnrollmentID(), wID, nil); err != nil {
		return nil, errors.WithMessagef(err, "programming error, failed to register recipient identity [%s]", wID)
	}
	logger.Debugf("created issuer wallet [%s]", wID)
	return newWallet, nil
}

func (s *WalletService) AuditorWallet(id string) (driver.AuditorWallet, error) {
	return s.auditorWallet(id)
}

func (s *WalletService) AuditorWalletByIdentity(identity view.Identity) (driver.AuditorWallet, error) {
	return s.auditorWallet(identity)
}

func (s *WalletService) auditorWallet(id interface{}) (driver.AuditorWallet, error) {
	logger.Debugf("get auditor wallet for [%v]", id)
	s.AuditorWalletsRegistry.Lock()
	defer s.AuditorWalletsRegistry.Unlock()
	logger.Debugf("get auditor wallet for [%v], lock acquired", id)

	// check if there is already a wallet
	w, idInfo, wID, err := s.AuditorWalletsRegistry.Lookup(id)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to lookup identity for auditor wallet [%v]", id)
	}
	logger.Debugf("lookup finished, wallet id is [%s]", wID)
	if w != nil {
		logger.Debugf("existing auditor wallet, return it [%s]", wID)
		return w.(driver.AuditorWallet), nil
	}

	// Create the wallet
	logger.Debugf("no wallet found, create it [%s]", wID)
	idInfoIdentity, _, err := idInfo.Get()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get auditor wallet identity for [%s:%s]", wID, id)
	}
	newWallet := newAuditorWallet(s, wID, idInfoIdentity)
	if err := s.AuditorWalletsRegistry.RegisterWallet(wID, newWallet); err != nil {
		return nil, errors.WithMessagef(err, "programming error, failed to register auditor wallet [%s]", wID)
	}
	if err := s.AuditorWalletsRegistry.BindIdentity(idInfoIdentity, idInfo.EnrollmentID(), wID, nil); err != nil {
		return nil, errors.WithMessagef(err, "programming error, failed to register recipient identity [%s]", wID)
	}
	logger.Debugf("created auditor wallet [%s]", wID)
	return newWallet, nil
}

func (s *WalletService) CertifierWallet(id string) (driver.CertifierWallet, error) {
	return nil, nil
}

func (s *WalletService) CertifierWalletByIdentity(identity view.Identity) (driver.CertifierWallet, error) {
	return nil, nil
}

// SpentIDs returns the spend ids for the passed token ids
func (s *WalletService) SpentIDs(ids ...*token.ID) ([]string, error) {
	return nil, nil
}

func (s *WalletService) Deserializer() (driver.Deserializer, error) {
	pp := s.PPM.PublicParams()
	if pp == nil {
		return nil, errors.Errorf("public parameters not inizialized")
	}
	d, err := s.DeserializerProvider(pp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deserializer")
	}
	return d, nil
}

type ownerWallet struct {
	WalletService *WalletService
	id            string
	identityInfo  driver.IdentityInfo
	cache         *idemix.WalletIdentityCache
}

func newOwnerWallet(walletService *WalletService, id string, identityInfo driver.IdentityInfo) (*ownerWallet, error) {
	w := &ownerWallet{
		WalletService: walletService,
		id:            id,
		identityInfo:  identityInfo,
	}
	if err := walletService.OwnerWalletsRegistry.RegisterWallet(id, w); err != nil {
		return nil, errors.Wrapf(err, "failed to register owner wallet [%s]", id)
	}

	cacheSize := 0
	tmsConfig := walletService.ConfigManager.TMS()
	conf := tmsConfig.GetOwnerWallet(id)
	if conf == nil {
		cacheSize = tmsConfig.GetWalletDefaultCacheSize()
	} else {
		cacheSize = conf.CacheSize
	}

	w.cache = idemix.NewWalletIdentityCache(w.getRecipientIdentity, cacheSize)
	logger.Debugf("added wallet cache for id %s with cache of size %d", id+"@"+identityInfo.EnrollmentID(), cacheSize)
	return w, nil
}

func (w *ownerWallet) ID() string {
	return w.id
}

func (w *ownerWallet) Contains(identity view.Identity) bool {
	return w.WalletService.OwnerWalletsRegistry.ContainsIdentity(identity, w.id)
}

// ContainsToken returns true if the passed token is owned by this wallet
func (w *ownerWallet) ContainsToken(token *token.UnspentToken) bool {
	return w.Contains(token.Owner.Raw)
}

func (w *ownerWallet) GetRecipientIdentity() (view.Identity, error) {
	return w.cache.Identity()
}

func (w *ownerWallet) getRecipientIdentity() (view.Identity, error) {
	// Get a new pseudonym
	pseudonym, _, err := w.identityInfo.Get()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting recipient identity from wallet [%s]", w.ID())
	}

	// Register the pseudonym
	if err := w.WalletService.OwnerWalletsRegistry.BindIdentity(pseudonym, w.identityInfo.EnrollmentID(), w.id, nil); err != nil {
		return nil, errors.WithMessagef(err, "failed storing recipient identity in wallet [%s]", w.ID())
	}
	return pseudonym, nil
}

func (w *ownerWallet) GetAuditInfo(id view.Identity) ([]byte, error) {
	return w.WalletService.identityProvider.GetAuditInfo(id)
}

func (w *ownerWallet) GetTokenMetadata(id view.Identity) ([]byte, error) {
	return nil, nil
}

func (w *ownerWallet) GetTokenMetadataAuditInfo(id view.Identity) ([]byte, error) {
	return nil, nil
}

func (w *ownerWallet) EnrollmentID() string {
	return w.identityInfo.EnrollmentID()
}

func (w *ownerWallet) RegisterRecipient(data *driver.RecipientData) error {
	logger.Debugf("register recipient identity on wallet [%s][%s]", data.Identity, w.id)
	if err := w.WalletService.RegisterRecipientIdentity(data); err != nil {
		return errors.WithMessagef(err, "failed to register recipient identity")
	}

	if err := w.WalletService.OwnerWalletsRegistry.BindIdentity(data.Identity, w.EnrollmentID(), w.id, nil); err != nil {
		return errors.WithMessagef(err, "failed storing recipient identity in wallet [%s]", w.id)
	}

	return nil
}

func (w *ownerWallet) GetSigner(identity view.Identity) (driver.Signer, error) {
	if !w.Contains(identity) {
		return nil, errors.Errorf("identity [%s] does not belong to this wallet [%s]", identity, w.ID())
	}

	si, err := w.WalletService.identityProvider.GetSigner(identity)
	if err != nil {
		return nil, err
	}
	return si, err
}

func (w *ownerWallet) ListTokens(opts *driver.ListTokensOptions) (*token.UnspentTokens, error) {
	logger.Debugf("wallet: list tokens, type [%s]", opts.TokenType)
	if w.WalletService.TokenVault == nil {
		return nil, errors.Errorf("no backend provided, cannot perform operation")
	}
	it, err := w.WalletService.TokenVault.UnspentTokensIteratorBy(w.id, opts.TokenType)
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
	if w.WalletService.TokenVault == nil {
		return nil, errors.Errorf("no backend provided, cannot perform operation")
	}
	it, err := w.WalletService.TokenVault.UnspentTokensIteratorBy(w.id, opts.TokenType)
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}
	return it, nil
}

func (w *ownerWallet) Remote() bool {
	return w.identityInfo.Remote()
}

type issuerWallet struct {
	walletService *WalletService

	id       string
	identity view.Identity
}

func newIssuerWallet(tokenService *WalletService, id string, identity view.Identity) *issuerWallet {
	return &issuerWallet{
		walletService: tokenService,
		id:            id,
		identity:      identity,
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
		return nil, errors.Errorf("identity [%s] does not belong to this wallet [%s]", identity, w.ID())
	}
	si, err := w.walletService.identityProvider.GetSigner(identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting signer for identity [%s] in wallet [%s]", identity, w.identity)
	}
	return si, nil
}

func (w *issuerWallet) HistoryTokens(opts *driver.ListTokensOptions) (*token.IssuedTokens, error) {
	logger.Debugf("issuer wallet [%s]: history tokens, type [%d]", w.ID(), opts.TokenType)
	if w.walletService.TokenVault == nil {
		return nil, errors.Errorf("no backend provided, cannot perform operation")
	}
	source, err := w.walletService.TokenVault.ListHistoryIssuedTokens()
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
	WalletService *WalletService
	id            string
	identity      view.Identity
}

func newAuditorWallet(tokenService *WalletService, id string, identity view.Identity) *auditorWallet {
	return &auditorWallet{
		WalletService: tokenService,
		id:            id,
		identity:      identity,
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
		return nil, errors.Errorf("identity [%s] does not belong to this wallet [%s]", id, w.ID())
	}

	si, err := w.WalletService.identityProvider.GetSigner(w.identity)
	if err != nil {
		return nil, err
	}
	return si, err
}
