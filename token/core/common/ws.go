/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	err "errors"
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var (
	ErrNilRecipientData = err.New("nil recipient data")
)

type TokenVault interface {
	IsPending(id *token.ID) (bool, error)
	GetTokenOutputsAndMeta(ctx context.Context, ids []*token.ID) ([][]byte, [][]byte, []token.TokenType, error)
	GetTokenOutputs(ids []*token.ID, callback driver.QueryCallbackFunc) error
	UnspentTokensIteratorBy(ctx context.Context, id string, tokenType token.TokenType) (driver.UnspentTokensIterator, error)
	ListHistoryIssuedTokens() (*token.IssuedTokens, error)
	PublicParams() ([]byte, error)
	Balance(id string, tokenType token.TokenType) (uint64, error)
}

type WalletRegistry interface {
	WalletIDs() ([]string, error)
	RegisterIdentity(config driver.IdentityConfiguration) error
	Lookup(id driver.WalletLookupID) (driver.Wallet, driver.IdentityInfo, string, error)
	RegisterWallet(id string, wallet driver.Wallet) error
	BindIdentity(identity driver.Identity, eID string, wID string, meta any) error
	ContainsIdentity(i driver.Identity, id string) bool
	GetIdentityMetadata(identity driver.Identity, wID string, meta any) error
}

type WalletFactory interface {
	NewWallet(role driver.IdentityRole, walletRegistry WalletRegistry, info driver.IdentityInfo, id string) (driver.Wallet, error)
}

type RegistryEntry struct {
	Registry WalletRegistry
	Mutex    *sync.RWMutex
}

type WalletService struct {
	Logger           logging.Logger
	IdentityProvider driver.IdentityProvider
	Deserializer     driver.Deserializer

	WalletFactory WalletFactory
	Registries    map[driver.IdentityRole]*RegistryEntry
}

func NewWalletService(
	logger logging.Logger,
	identityProvider driver.IdentityProvider,
	deserializer driver.Deserializer,
	walletFactory WalletFactory,
	OwnerWalletRegistry WalletRegistry,
	IssuerWalletRegistry WalletRegistry,
	AuditorWalletRegistry WalletRegistry,
	CertifierWalletsRegistry WalletRegistry,
) *WalletService {
	registries := map[driver.IdentityRole]*RegistryEntry{}
	registries[driver.OwnerRole] = &RegistryEntry{Registry: OwnerWalletRegistry, Mutex: &sync.RWMutex{}}
	registries[driver.IssuerRole] = &RegistryEntry{Registry: IssuerWalletRegistry, Mutex: &sync.RWMutex{}}
	registries[driver.AuditorRole] = &RegistryEntry{Registry: AuditorWalletRegistry, Mutex: &sync.RWMutex{}}
	registries[driver.CertifierRole] = &RegistryEntry{Registry: CertifierWalletsRegistry, Mutex: &sync.RWMutex{}}

	return &WalletService{
		Logger:           logger,
		IdentityProvider: identityProvider,
		Deserializer:     deserializer,
		WalletFactory:    walletFactory,
		Registries:       registries,
	}
}

func (s *WalletService) RegisterOwnerIdentity(config driver.IdentityConfiguration) error {
	return s.Registries[driver.OwnerRole].Registry.RegisterIdentity(config)
}

func (s *WalletService) RegisterIssuerIdentity(config driver.IdentityConfiguration) error {
	return s.Registries[driver.IssuerRole].Registry.RegisterIdentity(config)
}

func (s *WalletService) GetAuditInfo(id driver.Identity) ([]byte, error) {
	return s.IdentityProvider.GetAuditInfo(id)
}

func (s *WalletService) GetEnrollmentID(identity driver.Identity, auditInfo []byte) (string, error) {
	return s.IdentityProvider.GetEnrollmentID(identity, auditInfo)
}

func (s *WalletService) GetRevocationHandle(identity driver.Identity, auditInfo []byte) (string, error) {
	return s.IdentityProvider.GetRevocationHandler(identity, auditInfo)
}

func (s *WalletService) GetEIDAndRH(identity driver.Identity, auditInfo []byte) (string, string, error) {
	return s.IdentityProvider.GetEIDAndRH(identity, auditInfo)
}

func (s *WalletService) RegisterRecipientIdentity(data *driver.RecipientData) error {
	if data == nil {
		return errors.WithStack(ErrNilRecipientData)
	}
	if s.Logger.IsEnabledFor(zapcore.DebugLevel) {
		s.Logger.Debugf("register recipient identity [%s] with audit info [%s]", data.Identity, Hashable(data.AuditInfo))
	}

	// match identity and audit info
	err := s.Deserializer.MatchOwnerIdentity(data.Identity, data.AuditInfo)
	if err != nil {
		return errors.Wrapf(err, "failed to match identity to audit infor for [%s:%s]", data.Identity, Hashable(data.AuditInfo))
	}

	// register verifier and audit info
	v, err := s.Deserializer.GetOwnerVerifier(data.Identity)
	if err != nil {
		return errors.Wrapf(err, "failed getting verifier for [%s]", data.Identity)
	}
	if err := s.IdentityProvider.RegisterVerifier(data.Identity, v); err != nil {
		return errors.Wrapf(err, "failed registering verifier for [%s]", data.Identity)
	}
	if err := s.IdentityProvider.RegisterRecipientData(data); err != nil {
		return errors.Wrapf(err, "failed registering audit info for [%s]", data.Identity)
	}

	return nil
}

func (s *WalletService) Wallet(identity driver.Identity) driver.Wallet {
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

func (s *WalletService) OwnerWalletIDs() ([]string, error) {
	return s.Registries[driver.OwnerRole].Registry.WalletIDs()
}

func (s *WalletService) OwnerWallet(id driver.WalletLookupID) (driver.OwnerWallet, error) {
	w, err := s.walletByID(driver.OwnerRole, id)
	if err != nil {
		return nil, err
	}
	return w.(driver.OwnerWallet), nil
}

func (s *WalletService) IssuerWallet(id driver.WalletLookupID) (driver.IssuerWallet, error) {
	w, err := s.walletByID(driver.IssuerRole, id)
	if err != nil {
		return nil, err
	}
	return w.(driver.IssuerWallet), nil
}

func (s *WalletService) AuditorWallet(id driver.WalletLookupID) (driver.AuditorWallet, error) {
	w, err := s.walletByID(driver.AuditorRole, id)
	if err != nil {
		return nil, err
	}
	return w.(driver.AuditorWallet), nil
}

func (s *WalletService) CertifierWallet(id driver.WalletLookupID) (driver.CertifierWallet, error) {
	w, err := s.walletByID(driver.CertifierRole, id)
	if err != nil {
		return nil, err
	}
	return w.(driver.CertifierWallet), nil
}

// SpentIDs returns the spend ids for the passed token ids
func (s *WalletService) SpentIDs(ids ...*token.ID) ([]string, error) {
	return nil, nil
}

func (s *WalletService) walletByID(role driver.IdentityRole, id driver.WalletLookupID) (driver.Wallet, error) {
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
	newWallet, err := s.WalletFactory.NewWallet(role, registry, idInfo, wID)
	if err != nil {
		return nil, err
	}
	if err := registry.RegisterWallet(wID, newWallet); err != nil {
		return nil, err
	}
	return newWallet, nil
}
