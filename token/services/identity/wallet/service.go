/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wallet

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
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
	WalletByID(ctx context.Context, role identity.RoleType, id driver.WalletLookupID) (driver.Wallet, error)
}

type Service struct {
	Logger           logging.Logger
	IdentityProvider driver.IdentityProvider
	Deserializer     driver.Deserializer
	Registries       map[identity.RoleType]Registry
}

func NewService(
	logger logging.Logger,
	identityProvider driver.IdentityProvider,
	deserializer driver.Deserializer,
	registries map[identity.RoleType]Registry,
) *Service {
	return &Service{
		Logger:           logger,
		IdentityProvider: identityProvider,
		Deserializer:     deserializer,
		Registries:       registries,
	}
}

func (s *Service) RegisterOwnerIdentity(ctx context.Context, config driver.IdentityConfiguration) error {
	return s.Registries[identity.OwnerRole].RegisterIdentity(ctx, config)
}

func (s *Service) RegisterIssuerIdentity(ctx context.Context, config driver.IdentityConfiguration) error {
	return s.Registries[identity.IssuerRole].RegisterIdentity(ctx, config)
}

func (s *Service) GetAuditInfo(ctx context.Context, id driver.Identity) ([]byte, error) {
	return s.IdentityProvider.GetAuditInfo(ctx, id)
}

func (s *Service) GetEnrollmentID(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, error) {
	return s.IdentityProvider.GetEnrollmentID(ctx, identity, auditInfo)
}

func (s *Service) GetRevocationHandle(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, error) {
	return s.IdentityProvider.GetRevocationHandler(ctx, identity, auditInfo)
}

func (s *Service) GetEIDAndRH(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, string, error) {
	return s.IdentityProvider.GetEIDAndRH(ctx, identity, auditInfo)
}

func (s *Service) RegisterRecipientIdentity(ctx context.Context, data *driver.RecipientData) error {
	if data == nil {
		return errors.Wrapf(ErrNilRecipientData, "invalid recipient data")
	}

	// RegisterRecipientIdentity register the passed identity as a third-party recipient identity.
	if err := s.IdentityProvider.RegisterRecipientIdentity(ctx, data.Identity); err != nil {
		return errors.Wrapf(err, "failed to register recipient identity")
	}

	s.Logger.DebugfContext(ctx, "register recipient identity [%s] with audit info [%s]", data.Identity, utils.Hashable(data.AuditInfo))

	// match identity and audit info
	err := s.Deserializer.MatchIdentity(ctx, data.Identity, data.AuditInfo)
	if err != nil {
		return errors.Wrapf(err, "failed to match identity to audit infor for [%s]:[%s]", data.Identity, utils.Hashable(data.AuditInfo))
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
	return s.Registries[identity.OwnerRole].WalletIDs(ctx)
}

func (s *Service) OwnerWallet(ctx context.Context, id driver.WalletLookupID) (driver.OwnerWallet, error) {
	w, err := s.Registries[identity.OwnerRole].WalletByID(ctx, identity.OwnerRole, id)
	if err != nil {
		return nil, err
	}
	return w.(driver.OwnerWallet), nil
}

func (s *Service) IssuerWallet(ctx context.Context, id driver.WalletLookupID) (driver.IssuerWallet, error) {
	w, err := s.Registries[identity.IssuerRole].WalletByID(ctx, identity.IssuerRole, id)
	if err != nil {
		return nil, err
	}
	return w.(driver.IssuerWallet), nil
}

func (s *Service) AuditorWallet(ctx context.Context, id driver.WalletLookupID) (driver.AuditorWallet, error) {
	w, err := s.Registries[identity.AuditorRole].WalletByID(ctx, identity.AuditorRole, id)
	if err != nil {
		return nil, err
	}
	return w.(driver.AuditorWallet), nil
}

func (s *Service) CertifierWallet(ctx context.Context, id driver.WalletLookupID) (driver.CertifierWallet, error) {
	w, err := s.Registries[identity.CertifierRole].WalletByID(ctx, identity.CertifierRole, id)
	if err != nil {
		return nil, err
	}
	return w.(driver.CertifierWallet), nil
}

// SpendIDs returns the spend ids for the passed token ids
func (s *Service) SpendIDs(ids ...*token.ID) ([]string, error) {
	return nil, nil
}

func Convert[T Registry](s map[identity.RoleType]T) map[identity.RoleType]Registry {
	res := make(map[identity.RoleType]Registry, len(s))
	for role, v := range s {
		s[role] = v
	}
	return res
}
