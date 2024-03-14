/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenVault interface {
	IsPending(id *token.ID) (bool, error)
	GetTokenInfoAndOutputs(ids []*token.ID, callback driver.QueryCallback2Func) error
	GetTokenOutputs(ids []*token.ID, callback driver.QueryCallbackFunc) error
	UnspentTokensIteratorBy(id, tokenType string) (driver.UnspentTokensIterator, error)
	ListHistoryIssuedTokens() (*token.IssuedTokens, error)
}

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

type WalletFactory interface {
	NewOwnerWallet(s *WalletService, info driver.IdentityInfo, id string) (driver.OwnerWallet, error)
	NewIssuerWallet(s *WalletService, info driver.IdentityInfo, id string) (driver.IssuerWallet, error)
	NewAuditorWallet(s *WalletService, info driver.IdentityInfo, id string) (driver.AuditorWallet, error)
	NewCertifierWallet(s *WalletService, info driver.IdentityInfo, id string) (driver.CertifierWallet, error)
}

type DeserializerProviderFunc = func(params driver.PublicParameters) (driver.Deserializer, error)

type WalletService struct {
	Logger               *flogging.FabricLogger
	IdentityProvider     driver.IdentityProvider
	TokenVault           TokenVault
	PublicParamsManager  driver.PublicParamsManager
	DeserializerProvider DeserializerProviderFunc
	ConfigManager        config.Manager

	WalletFactory            WalletFactory
	OwnerWalletsRegistry     WalletRegistry
	IssuerWalletsRegistry    WalletRegistry
	AuditorWalletsRegistry   WalletRegistry
	CertifierWalletsRegistry WalletRegistry
}

func NewWalletService(
	logger *flogging.FabricLogger,
	identityProvider driver.IdentityProvider,
	tokenVault TokenVault,
	PPM driver.PublicParamsManager,
	deserializerProvider DeserializerProviderFunc,
	ConfigManager config.Manager,
	walletFactory WalletFactory,
	OwnerWalletRegistry WalletRegistry,
	IssuerWalletRegistry WalletRegistry,
	AuditorWalletRegistry WalletRegistry,
	CertifierWalletsRegistry WalletRegistry,
) *WalletService {
	return &WalletService{
		Logger:                   logger,
		IdentityProvider:         identityProvider,
		TokenVault:               tokenVault,
		PublicParamsManager:      PPM,
		ConfigManager:            ConfigManager,
		DeserializerProvider:     deserializerProvider,
		WalletFactory:            walletFactory,
		OwnerWalletsRegistry:     OwnerWalletRegistry,
		IssuerWalletsRegistry:    IssuerWalletRegistry,
		AuditorWalletsRegistry:   AuditorWalletRegistry,
		CertifierWalletsRegistry: CertifierWalletsRegistry,
	}
}

func (s *WalletService) RegisterOwnerWallet(id string, path string) error {
	return s.OwnerWalletsRegistry.RegisterIdentity(id, path)
}

func (s *WalletService) RegisterIssuerWallet(id string, path string) error {
	return s.IssuerWalletsRegistry.RegisterIdentity(id, path)
}

func (s *WalletService) GetAuditInfo(id view.Identity) ([]byte, error) {
	return s.IdentityProvider.GetAuditInfo(id)
}

func (s *WalletService) GetEnrollmentID(auditInfo []byte) (string, error) {
	return s.IdentityProvider.GetEnrollmentID(auditInfo)
}

func (s *WalletService) GetRevocationHandler(auditInfo []byte) (string, error) {
	return s.IdentityProvider.GetRevocationHandler(auditInfo)
}

func (s *WalletService) RegisterRecipientIdentity(data *driver.RecipientData) error {
	if data == nil {
		return errors.Errorf("cannot recigest empty recipient")
	}
	s.Logger.Debugf("register recipient identity [%s] with audit info [%s]", data.Identity.String(), hash.Hashable(data.AuditInfo).String())

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
	if err := s.IdentityProvider.RegisterVerifier(data.Identity, v); err != nil {
		return errors.Wrapf(err, "failed registering verifier for [%s]", data.Identity)
	}
	if err := s.IdentityProvider.RegisterAuditInfo(data.Identity, data.AuditInfo); err != nil {
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

	// create the wallet
	return s.WalletFactory.NewOwnerWallet(s, idInfo, wID)
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

	// create the wallet
	return s.WalletFactory.NewIssuerWallet(s, idInfo, wID)
}

func (s *WalletService) AuditorWallet(id string) (driver.AuditorWallet, error) {
	return s.auditorWallet(id)
}

func (s *WalletService) AuditorWalletByIdentity(identity view.Identity) (driver.AuditorWallet, error) {
	return s.auditorWallet(identity)
}

func (s *WalletService) auditorWallet(id interface{}) (driver.AuditorWallet, error) {
	s.Logger.Debugf("get auditor wallet for [%v]", id)
	s.AuditorWalletsRegistry.Lock()
	defer s.AuditorWalletsRegistry.Unlock()
	s.Logger.Debugf("get auditor wallet for [%v], lock acquired", id)

	// check if there is already a wallet
	w, idInfo, wID, err := s.AuditorWalletsRegistry.Lookup(id)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to lookup identity for auditor wallet [%v]", id)
	}
	s.Logger.Debugf("lookup finished, wallet id is [%s]", wID)
	if w != nil {
		s.Logger.Debugf("existing auditor wallet, return it [%s]", wID)
		return w.(driver.AuditorWallet), nil
	}

	// create the wallet
	return s.WalletFactory.NewAuditorWallet(s, idInfo, wID)
}

func (s *WalletService) CertifierWallet(id string) (driver.CertifierWallet, error) {
	return s.certifierWallet(id)
}

func (s *WalletService) CertifierWalletByIdentity(identity view.Identity) (driver.CertifierWallet, error) {
	return s.certifierWallet(identity)
}

func (s *WalletService) certifierWallet(id interface{}) (driver.CertifierWallet, error) {
	s.CertifierWalletsRegistry.Lock()
	defer s.CertifierWalletsRegistry.Unlock()

	// check if there is already a wallet
	w, idInfo, wID, err := s.CertifierWalletsRegistry.Lookup(id)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to lookup identity for issuer wallet [%v]", id)
	}
	if w != nil {
		return w.(driver.CertifierWallet), nil
	}

	// create the wallet
	return s.WalletFactory.NewCertifierWallet(s, idInfo, wID)
}

// SpentIDs returns the spend ids for the passed token ids
func (s *WalletService) SpentIDs(ids ...*token.ID) ([]string, error) {
	return nil, nil
}

func (s *WalletService) Deserializer() (driver.Deserializer, error) {
	pp := s.PublicParamsManager.PublicParameters()
	if pp == nil {
		return nil, errors.Errorf("public parameters not inizialized")
	}
	d, err := s.DeserializerProvider(pp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deserializer")
	}
	return d, nil
}
