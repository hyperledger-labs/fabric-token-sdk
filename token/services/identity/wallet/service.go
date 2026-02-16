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
	// This makes sure that Service implements driver.WalletService
	_ driver.WalletService = &Service{}
)

var (
	// ErrNilRecipientData is returned when a nil RecipientData is passed to RegisterRecipientIdentity
	ErrNilRecipientData = errors.New("nil recipient data")
)

type RoleRegistries map[identity.RoleType]RoleRegistry

// RoleRegistry models an external registry that holds wallets for a given role.
// It is used by the wallet service to lookup and register identities and wallets.
//
//go:generate counterfeiter -o mock/registry.go -fake-name Registry . Registry
type RoleRegistry interface {
	WalletIDs(ctx context.Context) ([]string, error)
	RegisterIdentity(ctx context.Context, config driver.IdentityConfiguration) error
	WalletByID(ctx context.Context, role identity.RoleType, id driver.WalletLookupID) (driver.Wallet, error)
}

// Service implements the driver.WalletService interface.
// Service exposes wallet-related helper operations used by the token management layer.
// It delegates identity operations to the configured IdentityProvider and uses
// registries for role-specific wallet lookups and registrations.
type Service struct {
	Logger           logging.Logger
	IdentityProvider driver.IdentityProvider
	Deserializer     driver.Deserializer
	RoleRegistries   RoleRegistries
}

// NewService creates a new wallet Service.
func NewService(
	logger logging.Logger,
	identityProvider driver.IdentityProvider,
	deserializer driver.Deserializer,
	roleRegistries RoleRegistries,
) *Service {
	return &Service{
		Logger:           logger,
		IdentityProvider: identityProvider,
		Deserializer:     deserializer,
		RoleRegistries:   roleRegistries,
	}
}

// RegisterOwnerIdentity registers a long-term owner identity using the owner registry.
func (s *Service) RegisterOwnerIdentity(ctx context.Context, config driver.IdentityConfiguration) error {
	return s.RoleRegistries[identity.OwnerRole].RegisterIdentity(ctx, config)
}

// RegisterIssuerIdentity registers a long-term issuer identity using the issuer registry.
func (s *Service) RegisterIssuerIdentity(ctx context.Context, config driver.IdentityConfiguration) error {
	return s.RoleRegistries[identity.IssuerRole].RegisterIdentity(ctx, config)
}

// GetAuditInfo retrieves audit information for the given identity using the configured IdentityProvider.
func (s *Service) GetAuditInfo(ctx context.Context, id driver.Identity) ([]byte, error) {
	return s.IdentityProvider.GetAuditInfo(ctx, id)
}

// GetEnrollmentID extracts the enrollment id from the passed audit information using the IdentityProvider.
func (s *Service) GetEnrollmentID(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, error) {
	return s.IdentityProvider.GetEnrollmentID(ctx, identity, auditInfo)
}

// GetRevocationHandle extracts the revocation handle from the passed audit information using the IdentityProvider.
func (s *Service) GetRevocationHandle(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, error) {
	return s.IdentityProvider.GetRevocationHandler(ctx, identity, auditInfo)
}

// GetEIDAndRH returns both enrollment ID and revocation handle from audit info via the IdentityProvider.
func (s *Service) GetEIDAndRH(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, string, error) {
	return s.IdentityProvider.GetEIDAndRH(ctx, identity, auditInfo)
}

// RegisterRecipientIdentity registers the passed identity as a third-party recipient identity.
// The function performs these steps:
//   - validate the input
//   - ask the IdentityProvider to register the recipient identity
//   - match the identity against the provided audit info using the Deserializer
//   - obtain the owner verifier and register it with the IdentityProvider
//   - store the recipient data via the IdentityProvider
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

// Wallet returns a wallet bound to the passed identity. It tries to resolve an owner wallet first
// and then an issuer wallet. It returns nil if no wallet is found.
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

// OwnerWalletIDs returns the list of owner wallet identifiers from the owner registry.
func (s *Service) OwnerWalletIDs(ctx context.Context) ([]string, error) {
	return s.RoleRegistries[identity.OwnerRole].WalletIDs(ctx)
}

// OwnerWallet returns the OwnerWallet instance bound to the passed lookup id.
func (s *Service) OwnerWallet(ctx context.Context, id driver.WalletLookupID) (driver.OwnerWallet, error) {
	w, err := s.RoleRegistries[identity.OwnerRole].WalletByID(ctx, identity.OwnerRole, id)
	if err != nil {
		return nil, err
	}

	return w.(driver.OwnerWallet), nil
}

// IssuerWallet returns the IssuerWallet instance bound to the passed lookup id.
func (s *Service) IssuerWallet(ctx context.Context, id driver.WalletLookupID) (driver.IssuerWallet, error) {
	w, err := s.RoleRegistries[identity.IssuerRole].WalletByID(ctx, identity.IssuerRole, id)
	if err != nil {
		return nil, err
	}

	return w.(driver.IssuerWallet), nil
}

// AuditorWallet returns the AuditorWallet instance bound to the passed lookup id.
func (s *Service) AuditorWallet(ctx context.Context, id driver.WalletLookupID) (driver.AuditorWallet, error) {
	w, err := s.RoleRegistries[identity.AuditorRole].WalletByID(ctx, identity.AuditorRole, id)
	if err != nil {
		return nil, err
	}

	return w.(driver.AuditorWallet), nil
}

// CertifierWallet returns the CertifierWallet instance bound to the passed lookup id.
func (s *Service) CertifierWallet(ctx context.Context, id driver.WalletLookupID) (driver.CertifierWallet, error) {
	w, err := s.RoleRegistries[identity.CertifierRole].WalletByID(ctx, identity.CertifierRole, id)
	if err != nil {
		return nil, err
	}

	return w.(driver.CertifierWallet), nil
}

// SpendIDs returns the spend ids for the passed token ids.
// The current implementation converts token.ID into its string representation.
func (s *Service) SpendIDs(ids ...*token.ID) ([]string, error) {
	if len(ids) == 0 {
		return []string{}, nil
	}
	res := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == nil {
			// skip nil ids
			continue
		}
		res = append(res, id.String())
	}

	return res, nil
}

// Convert converts a map of concrete registries into a map of the RoleRegistry interface type.
func Convert[T RoleRegistry](s map[identity.RoleType]T) RoleRegistries {
	res := make(RoleRegistries, len(s))
	for role, v := range s {
		res[role] = v
	}

	return res
}
