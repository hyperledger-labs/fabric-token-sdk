/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	math "github.com/IBM/mathlib"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	identity2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/common"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/config"
	idemix2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix"
	x5092 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509"
	"github.com/pkg/errors"
)

const (
	// OwnerMSPID is the default MSP ID for the owner wallet
	OwnerMSPID = "OwnerMSPID"
	// IssuerMSPID is the default MSP ID for the issuer wallet
	IssuerMSPID = "IssuerMSPID"
	// AuditorMSPID is the default MSP ID for the auditor wallet
	AuditorMSPID = "AuditorMSPID"
	// CertifierMSPID is the default MSP ID for the certifier wallet
	CertifierMSPID = "CertifierMSPID"
)

// RoleToMSPID maps the role to the MSP ID
var RoleToMSPID = map[driver.IdentityRole]string{
	driver.OwnerRole:     OwnerMSPID,
	driver.IssuerRole:    IssuerMSPID,
	driver.AuditorRole:   AuditorMSPID,
	driver.CertifierRole: CertifierMSPID,
}

// RoleFactory is the factory for creating wallets, idemix and x509
type RoleFactory struct {
	TMSID                  token.TMSID
	Config                 config2.Config
	FSCIdentity            view2.Identity
	NetworkDefaultIdentity view2.Identity
	SignerService          common.SigService
	BinderService          common.BinderService
	StorageProvider        identity2.StorageProvider
	DeserializerManager    deserializer.Manager
	ignoreRemote           bool
}

// NewRoleFactory creates a new RoleFactory
func NewRoleFactory(
	TMSID token.TMSID,
	config config2.Config,
	fscIdentity view2.Identity,
	networkDefaultIdentity view2.Identity,
	signerService common.SigService,
	binderService common.BinderService,
	storageProvider identity2.StorageProvider,
	deserializerManager deserializer.Manager,
	ignoreRemote bool,
) *RoleFactory {
	return &RoleFactory{
		TMSID:                  TMSID,
		Config:                 config,
		FSCIdentity:            fscIdentity,
		NetworkDefaultIdentity: networkDefaultIdentity,
		SignerService:          signerService,
		BinderService:          binderService,
		StorageProvider:        storageProvider,
		DeserializerManager:    deserializerManager,
		ignoreRemote:           ignoreRemote,
	}
}

// NewIdemix creates a new Idemix-based role
func (f *RoleFactory) NewIdemix(role driver.IdentityRole, cacheSize int, curveID math.CurveID) (identity2.Role, error) {
	identities, err := f.IdentitiesForRole(role)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identities for role [%d]", role)
	}

	walletPathStorage, err := f.StorageProvider.OpenIdentityDB(f.TMSID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get wallet path storage")
	}
	keystore, err := f.StorageProvider.NewKeystore()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get new keystore")
	}
	lm := idemix2.NewLocalMembership(
		f.Config,
		f.NetworkDefaultIdentity,
		f.SignerService,
		f.DeserializerManager,
		walletPathStorage,
		keystore,
		RoleToMSPID[role],
		cacheSize,
		curveID,
		identities,
		f.ignoreRemote,
	)
	return &BindingRole{Role: idemix2.NewRole(f.TMSID.Network, f.FSCIdentity, lm), Provider: f}, nil
}

// NewX509 creates a new X509-based role
func (f *RoleFactory) NewX509(role driver.IdentityRole) (identity2.Role, error) {
	identities, err := f.IdentitiesForRole(role)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identities for role [%d]", role)
	}
	walletPathStorage, err := f.StorageProvider.OpenIdentityDB(f.TMSID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get wallet path storage")
	}
	lm := x5092.NewLocalMembership(
		f.Config,
		f.NetworkDefaultIdentity,
		f.SignerService,
		f.BinderService,
		f.DeserializerManager,
		walletPathStorage,
		RoleToMSPID[role],
		false,
	)
	if err := lm.Load(identities); err != nil {
		return nil, errors.WithMessage(err, "failed to load owners")
	}
	return &BindingRole{Role: x5092.NewRole(f.TMSID.Network, f.FSCIdentity, lm), Provider: f}, nil
}

// NewX509IgnoreRemote creates a new X509-based role treating the long-term identities as local
func (f *RoleFactory) NewX509IgnoreRemote(role driver.IdentityRole) (identity2.Role, error) {
	identities, err := f.IdentitiesForRole(role)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identities for role [%d]", role)
	}
	walletPathStorage, err := f.StorageProvider.OpenIdentityDB(f.TMSID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get wallet path storage")
	}
	lm := x5092.NewLocalMembership(
		f.Config,
		f.NetworkDefaultIdentity,
		f.SignerService,
		f.BinderService,
		f.DeserializerManager,
		walletPathStorage,
		RoleToMSPID[role],
		true,
	)
	if err := lm.Load(identities); err != nil {
		return nil, errors.WithMessage(err, "failed to load owners")
	}
	return &BindingRole{Role: x5092.NewRole(f.TMSID.Network, f.FSCIdentity, lm), Provider: f}, nil
}

// IdentitiesForRole returns the configured identities for the passed role
func (f *RoleFactory) IdentitiesForRole(role driver.IdentityRole) ([]*config.Identity, error) {
	return f.Config.IdentitiesForRole(role)
}

type BindingRole struct {
	identity2.Role
	Provider *RoleFactory
}

func (r *BindingRole) GetIdentityInfo(id string) (driver.IdentityInfo, error) {
	info, err := r.Role.GetIdentityInfo(id)
	if err != nil {
		return nil, err
	}
	return &Info{IdentityInfo: info, Provider: r.Provider}, nil
}

// Info wraps a driver.IdentityInfo to further register the audit info,
// and binds the new identity to the default FSC node identity
type Info struct {
	driver.IdentityInfo
	Provider *RoleFactory
}

func (i *Info) ID() string {
	return i.IdentityInfo.ID()
}

func (i *Info) EnrollmentID() string {
	return i.IdentityInfo.EnrollmentID()
}

func (i *Info) Get() (view2.Identity, []byte, error) {
	// get the identity
	id, ai, err := i.IdentityInfo.Get()
	if err != nil {
		return nil, nil, err
	}
	// register the audit info
	if err := i.Provider.SignerService.RegisterAuditInfo(id, ai); err != nil {
		return nil, nil, err
	}
	// bind the identity to the default FSC node identity
	if i.Provider.BinderService != nil {
		if err := i.Provider.BinderService.Bind(i.Provider.FSCIdentity, id); err != nil {
			return nil, nil, err
		}
	}
	return id, ai, nil
}
