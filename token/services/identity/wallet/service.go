/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wallet

import (
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var (
	ErrNilRecipientData = errors.New("nil recipient data")
)

type Registry interface {
	WalletIDs() ([]string, error)
	RegisterIdentity(config driver.IdentityConfiguration) error
	Lookup(id driver.WalletLookupID) (driver.Wallet, identity.Info, string, error)
	RegisterWallet(id string, wallet driver.Wallet) error
	BindIdentity(identity driver.Identity, eID string, wID string, meta any) error
	ContainsIdentity(i driver.Identity, id string) bool
	GetIdentityMetadata(identity driver.Identity, wID string, meta any) error
}

type Factory interface {
	NewWallet(id string, role identity.RoleType, wr Registry, info identity.Info) (driver.Wallet, error)
}

type RegistryEntry struct {
	Registry Registry
	Mutex    *sync.RWMutex
}

type Service struct {
	Logger           logging.Logger
	IdentityProvider driver.IdentityProvider
	Deserializer     driver.Deserializer

	WalletFactory Factory
	Registries    map[identity.RoleType]*RegistryEntry
}

func NewService(
	logger logging.Logger,
	identityProvider driver.IdentityProvider,
	deserializer driver.Deserializer,
	walletFactory Factory,
	walletRegistries map[identity.RoleType]Registry,
) *Service {
	registries := map[identity.RoleType]*RegistryEntry{}
	for roleType, registry := range walletRegistries {
		registries[roleType] = &RegistryEntry{
			Registry: registry,
			Mutex:    &sync.RWMutex{},
		}
	}
	return &Service{
		Logger:           logger,
		IdentityProvider: identityProvider,
		Deserializer:     deserializer,
		WalletFactory:    walletFactory,
		Registries:       registries,
	}
}

func (s *Service) RegisterOwnerIdentity(config driver.IdentityConfiguration) error {
	return s.Registries[identity.OwnerRole].Registry.RegisterIdentity(config)
}

func (s *Service) RegisterIssuerIdentity(config driver.IdentityConfiguration) error {
	return s.Registries[identity.IssuerRole].Registry.RegisterIdentity(config)
}

func (s *Service) GetAuditInfo(id driver.Identity) ([]byte, error) {
	return s.IdentityProvider.GetAuditInfo(id)
}

func (s *Service) GetEnrollmentID(identity driver.Identity, auditInfo []byte) (string, error) {
	return s.IdentityProvider.GetEnrollmentID(identity, auditInfo)
}

func (s *Service) GetRevocationHandle(identity driver.Identity, auditInfo []byte) (string, error) {
	return s.IdentityProvider.GetRevocationHandler(identity, auditInfo)
}

func (s *Service) GetEIDAndRH(identity driver.Identity, auditInfo []byte) (string, string, error) {
	return s.IdentityProvider.GetEIDAndRH(identity, auditInfo)
}

func (s *Service) RegisterRecipientIdentity(data *driver.RecipientData) error {
	if data == nil {
		return errors.WithStack(ErrNilRecipientData)
	}

	// RegisterRecipientIdentity register the passed identity as a third-party recipient identity.
	if err := s.IdentityProvider.RegisterRecipientIdentity(data.Identity); err != nil {
		return errors.Wrapf(err, "failed to register recipient identity")
	}

	if s.Logger.IsEnabledFor(zapcore.DebugLevel) {
		s.Logger.Debugf("register recipient identity [%s] with audit info [%s]", data.Identity, utils.Hashable(data.AuditInfo))
	}

	// match identity and audit info
	err := s.Deserializer.MatchOwnerIdentity(data.Identity, data.AuditInfo)
	if err != nil {
		return errors.Wrapf(err, "failed to match identity to audit infor for [%s:%s]", data.Identity, utils.Hashable(data.AuditInfo))
	}

	// register verifier and audit info
	v, err := s.Deserializer.GetOwnerVerifier(data.Identity)
	if err != nil {
		return errors.Wrapf(err, "failed getting verifier for owner [%s]", data.Identity)
	}
	if err := s.IdentityProvider.RegisterVerifier(data.Identity, v); err != nil {
		return errors.Wrapf(err, "failed registering verifier for owner [%s]", data.Identity)
	}
	if err := s.IdentityProvider.RegisterRecipientData(data); err != nil {
		return errors.Wrapf(err, "failed registering audit info for owner [%s]", data.Identity)
	}

	return nil
}

func (s *Service) Wallet(identity driver.Identity) driver.Wallet {
	w, _ := s.OwnerWallet(identity)
	if w != nil {
		return w
	}
	iw, _ := s.IssuerWallet(identity)
	if iw != nil {
		return iw
	}
	return nil
}

func (s *Service) OwnerWalletIDs() ([]string, error) {
	return s.Registries[identity.OwnerRole].Registry.WalletIDs()
}

func (s *Service) OwnerWallet(id driver.WalletLookupID) (driver.OwnerWallet, error) {
	w, err := s.walletByID(identity.OwnerRole, id)
	if err != nil {
		return nil, err
	}
	return w.(driver.OwnerWallet), nil
}

func (s *Service) IssuerWallet(id driver.WalletLookupID) (driver.IssuerWallet, error) {
	w, err := s.walletByID(identity.IssuerRole, id)
	if err != nil {
		return nil, err
	}
	return w.(driver.IssuerWallet), nil
}

func (s *Service) AuditorWallet(id driver.WalletLookupID) (driver.AuditorWallet, error) {
	w, err := s.walletByID(identity.AuditorRole, id)
	if err != nil {
		return nil, err
	}
	return w.(driver.AuditorWallet), nil
}

func (s *Service) CertifierWallet(id driver.WalletLookupID) (driver.CertifierWallet, error) {
	w, err := s.walletByID(identity.CertifierRole, id)
	if err != nil {
		return nil, err
	}
	return w.(driver.CertifierWallet), nil
}

// SpentIDs returns the spend ids for the passed token ids
func (s *Service) SpentIDs(ids ...*token.ID) ([]string, error) {
	return nil, nil
}

func (s *Service) walletByID(role identity.RoleType, id driver.WalletLookupID) (driver.Wallet, error) {
	entry := s.Registries[role]
	registry := entry.Registry
	mutex := entry.Mutex

	mutex.RLock()
	w, _, _, err := registry.Lookup(id)
	if err != nil {
		mutex.RUnlock()
		return nil, errors.WithMessagef(err, "failed to lookup identity for owner wallet [%s]", id)
	}
	if w != nil {
		mutex.RUnlock()
		return w, nil
	}
	mutex.RUnlock()

	mutex.Lock()
	defer mutex.Unlock()

	w, idInfo, wID, err := registry.Lookup(id)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to lookup identity for owner wallet [%s]", id)
	}
	if w != nil {
		return w, nil
	}

	// create the wallet
	newWallet, err := s.WalletFactory.NewWallet(wID, role, registry, idInfo)
	if err != nil {
		return nil, err
	}
	if err := registry.RegisterWallet(wID, newWallet); err != nil {
		return nil, err
	}
	return newWallet, nil
}
