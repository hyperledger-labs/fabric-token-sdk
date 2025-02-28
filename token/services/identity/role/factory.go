/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package role

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/membership"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

var toString = map[identity.RoleType]string{
	identity.OwnerRole:     "Owner",
	identity.IssuerRole:    "Issuer",
	identity.AuditorRole:   "Auditor",
	identity.CertifierRole: "Certifier",
}

type StorageProvider interface {
	IdentityDB(tmsID token.TMSID) (driver2.IdentityDB, error)
}

// Factory is the factory for creating wallets, idemix and x509
type Factory struct {
	Logger                 logging.Logger
	TMSID                  token.TMSID
	Config                 driver.Config
	FSCIdentity            driver.Identity
	NetworkDefaultIdentity driver.Identity
	IdentityProvider       driver.IdentityProvider
	SignerService          driver.SigService
	BinderService          driver.BinderService
	StorageProvider        StorageProvider
	DeserializerManager    driver.DeserializerManager
}

// NewFactory creates a new Factory
func NewFactory(
	logger logging.Logger,
	TMSID token.TMSID,
	config driver.Config,
	fscIdentity driver.Identity,
	networkDefaultIdentity driver.Identity,
	identityProvider driver.IdentityProvider,
	signerService driver.SigService,
	binderService driver.BinderService,
	storageProvider StorageProvider,
	deserializerManager driver.DeserializerManager,
) *Factory {
	return &Factory{
		Logger:                 logger,
		TMSID:                  TMSID,
		Config:                 config,
		FSCIdentity:            fscIdentity,
		NetworkDefaultIdentity: networkDefaultIdentity,
		IdentityProvider:       identityProvider,
		SignerService:          signerService,
		BinderService:          binderService,
		StorageProvider:        storageProvider,
		DeserializerManager:    deserializerManager,
	}
}

func (f *Factory) NewRole(role identity.RoleType, defaultAnon bool, targets []driver.Identity, kmps ...membership.KeyManagerProvider) (identity.Role, error) {
	identityDB, err := f.StorageProvider.IdentityDB(f.TMSID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get wallet path storage")
	}
	lm := membership.NewLocalMembership(
		f.Logger.Named(fmt.Sprintf("membership.role.%s", identity.RoleToString(role))),
		f.Config,
		f.NetworkDefaultIdentity,
		f.SignerService,
		f.DeserializerManager,
		identityDB,
		f.BinderService,
		toString[role],
		defaultAnon,
		f.IdentityProvider,
		kmps...,
	)
	identities, err := f.Config.IdentitiesForRole(role)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identities for role [%d]", role)
	}
	if err := lm.Load(identities, targets); err != nil {
		return nil, errors.WithMessage(err, "failed to load identities")
	}
	return NewRole(f.Logger, role, f.TMSID.Network, f.FSCIdentity, lm), nil
}
