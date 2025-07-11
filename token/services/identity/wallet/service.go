/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wallet

import (
	"context"
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

var (
	ErrNilRecipientData = errors.New("nil recipient data")
)

type Registry interface {
	WalletIDs(ctx context.Context) ([]string, error)
	RegisterIdentity(ctx context.Context, config driver.IdentityConfiguration) error
	Lookup(ctx context.Context, id driver.WalletLookupID) (driver.Wallet, identity.Info, string, error)
	RegisterWallet(ctx context.Context, id string, wallet driver.Wallet) error
	BindIdentity(ctx context.Context, identity driver.Identity, eID string, wID string, meta any) error
	ContainsIdentity(ctx context.Context, i driver.Identity, id string) bool
	GetIdentityMetadata(ctx context.Context, identity driver.Identity, wID string, meta any) error
}

type walletFactory interface {
	NewWallet(ctx context.Context, id string, role identity.RoleType, wr Registry, info identity.Info) (driver.Wallet, error)
}

type RegistryEntry struct {
	Registry Registry
	Mutex    *sync.RWMutex
}

type Service struct {
	Logger           logging.Logger
	IdentityProvider driver.IdentityProvider
	Deserializer     driver.Deserializer

	WalletFactory walletFactory
	Registries    map[identity.RoleType]*RegistryEntry
}

func NewService(
	logger logging.Logger,
	identityProvider driver.IdentityProvider,
	deserializer driver.Deserializer,
	walletFactory walletFactory,
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

func (s *Service) RegisterOwnerIdentity(ctx context.Context, config driver.IdentityConfiguration) error {
	return s.Registries[identity.OwnerRole].Registry.RegisterIdentity(ctx, config)
}

func (s *Service) RegisterIssuerIdentity(ctx context.Context, config driver.IdentityConfiguration) error {
	return s.Registries[identity.IssuerRole].Registry.RegisterIdentity(ctx, config)
}

func (s *Service) GetAuditInfo(ctx context.Context, id driver.Identity) ([]byte, error) {
	return s.IdentityProvider.GetAuditInfo(ctx, id)
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

func (s *Service) RegisterRecipientIdentity(ctx context.Context, data *driver.RecipientData) error {
	if data == nil {
		return errors.WithStack(ErrNilRecipientData)
	}

	// RegisterRecipientIdentity register the passed identity as a third-party recipient identity.
	if err := s.IdentityProvider.RegisterRecipientIdentity(data.Identity); err != nil {
		return errors.Wrapf(err, "failed to register recipient identity")
	}

	s.Logger.DebugfContext(ctx, "register recipient identity [%s] with audit info [%s]", data.Identity, utils.Hashable(data.AuditInfo))

	// match identity and audit info
	err := s.Deserializer.MatchIdentity(data.Identity, data.AuditInfo)
	if err != nil {
		return errors.Wrapf(err, "failed to match identity to audit infor for [%s]:[%s]", data.Identity, utils.Hashable(data.AuditInfo))
	}

	// register verifier and audit info
	v, err := s.Deserializer.GetOwnerVerifier(data.Identity)
	if err != nil {
		return errors.Wrapf(err, "failed getting verifier for owner [%s]", data.Identity)
	}
	if err := s.IdentityProvider.RegisterVerifier(ctx, data.Identity, v); err != nil {
		return errors.Wrapf(err, "failed registering verifier for owner [%s]", data.Identity)
	}
	if err := s.IdentityProvider.RegisterRecipientData(ctx, data); err != nil {
		return errors.Wrapf(err, "failed registering audit info for owner [%s]", data.Identity)
	}

	return nil
}

func (s *Service) Wallet(ctx context.Context, identity driver.Identity) driver.Wallet {
	w, _ := s.OwnerWallet(ctx, identity)
	if w != nil {
		return w
	}
	iw, _ := s.IssuerWallet(ctx, identity)
	if iw != nil {
		return iw
	}
	return nil
}

func (s *Service) OwnerWalletIDs(ctx context.Context) ([]string, error) {
	return s.Registries[identity.OwnerRole].Registry.WalletIDs(ctx)
}

func (s *Service) OwnerWallet(ctx context.Context, id driver.WalletLookupID) (driver.OwnerWallet, error) {
	w, err := s.walletByID(ctx, identity.OwnerRole, id)
	if err != nil {
		return nil, err
	}
	return w.(driver.OwnerWallet), nil
}

func (s *Service) IssuerWallet(ctx context.Context, id driver.WalletLookupID) (driver.IssuerWallet, error) {
	w, err := s.walletByID(ctx, identity.IssuerRole, id)
	if err != nil {
		return nil, err
	}
	return w.(driver.IssuerWallet), nil
}

func (s *Service) AuditorWallet(ctx context.Context, id driver.WalletLookupID) (driver.AuditorWallet, error) {
	w, err := s.walletByID(ctx, identity.AuditorRole, id)
	if err != nil {
		return nil, err
	}
	return w.(driver.AuditorWallet), nil
}

func (s *Service) CertifierWallet(ctx context.Context, id driver.WalletLookupID) (driver.CertifierWallet, error) {
	w, err := s.walletByID(ctx, identity.CertifierRole, id)
	if err != nil {
		return nil, err
	}
	return w.(driver.CertifierWallet), nil
}

// SpentIDs returns the spend ids for the passed token ids
func (s *Service) SpendIDs(ids ...*token.ID) ([]string, error) {
	return nil, nil
}

func (s *Service) walletByID(ctx context.Context, role identity.RoleType, id driver.WalletLookupID) (driver.Wallet, error) {
	entry := s.Registries[role]
	registry := entry.Registry
	mutex := entry.Mutex

	mutex.RLock()
	w, _, _, err := registry.Lookup(ctx, id)
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

	w, idInfo, wID, err := registry.Lookup(ctx, id)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to lookup identity for owner wallet [%s]", id)
	}
	if w != nil {
		return w, nil
	}

	// create the wallet
	newWallet, err := s.WalletFactory.NewWallet(ctx, wID, role, registry, idInfo)
	if err != nil {
		return nil, err
	}
	if err := registry.RegisterWallet(ctx, wID, newWallet); err != nil {
		return nil, err
	}
	return newWallet, nil
}
