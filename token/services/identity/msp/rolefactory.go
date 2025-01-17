/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	idemix2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix/msp"
	x5092 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
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
	Logger                 logging.Logger
	TMSID                  token.TMSID
	Config                 driver2.Config
	FSCIdentity            driver.Identity
	NetworkDefaultIdentity driver.Identity
	IdentityProvider       driver2.IdentityProvider
	SignerService          driver2.SigService
	BinderService          driver2.BinderService
	StorageProvider        identity.StorageProvider
	DeserializerManager    driver2.DeserializerManager
	ignoreRemote           bool
}

// NewRoleFactory creates a new RoleFactory
func NewRoleFactory(
	logger logging.Logger,
	TMSID token.TMSID,
	config driver2.Config,
	fscIdentity driver.Identity,
	networkDefaultIdentity driver.Identity,
	identityProvider driver2.IdentityProvider,
	signerService driver2.SigService,
	binderService driver2.BinderService,
	storageProvider identity.StorageProvider,
	deserializerManager driver2.DeserializerManager,
	ignoreRemote bool,
) *RoleFactory {
	return &RoleFactory{
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
		ignoreRemote:           ignoreRemote,
	}
}

// NewIdemix creates a new Idemix-based role
func (f *RoleFactory) NewIdemix(role driver.IdentityRole, cacheSize int, issuerPublicKey *crypto.IdemixIssuerPublicKey, additionalKMPs ...common.KeyManagerProvider) (identity.Role, error) {
	f.Logger.Debugf("create idemix role for [%s]", driver.IdentityRoleStrings[role])
	if issuerPublicKey == nil && len(additionalKMPs) == 0 {
		return nil, errors.New("expected a non-nil idemix public key")
	}

	kmps := additionalKMPs
	if issuerPublicKey != nil {
		backend, err := f.StorageProvider.NewKeystore()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get new keystore backend")
		}
		keyStore, err := msp.NewKeyStore(issuerPublicKey.Curve, backend)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to instantiate bccsp key store")
		}
		kmp := idemix2.NewKeyManagerProvider(
			issuerPublicKey.PublicKey,
			issuerPublicKey.Curve,
			RoleToMSPID[role],
			keyStore,
			f.SignerService,
			f.Config,
			cacheSize,
			f.ignoreRemote,
		)
		kmps = append([]common.KeyManagerProvider{kmp}, additionalKMPs...)
	}

	identityDB, err := f.StorageProvider.OpenIdentityDB(f.TMSID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get wallet path storage")
	}
	lm := common.NewLocalMembership(
		f.Logger,
		f.Config,
		f.NetworkDefaultIdentity,
		f.SignerService,
		f.DeserializerManager,
		identityDB,
		f.BinderService,
		RoleToMSPID[role],
		common.NewMultiplexerKeyManagerProvider(kmps),
		true,
	)
	identities, err := f.IdentitiesForRole(role)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identities for role [%d]", role)
	}
	if err := lm.Load(identities, nil); err != nil {
		return nil, errors.WithMessage(err, "failed to load identities")
	}
	return &WrappingBindingRole{
		Role:             common.NewAnonymousRole(f.Logger, role, f.TMSID.Network, f.FSCIdentity, lm),
		IdentityType:     IdemixIdentity,
		RootIdentity:     f.FSCIdentity,
		IdentityProvider: f.IdentityProvider,
		BinderService:    f.BinderService,
	}, nil
}

// NewX509 creates a new X509-based role
func (f *RoleFactory) NewX509(role driver.IdentityRole, targets ...driver.Identity) (identity.Role, error) {
	return f.newX509WithType(role, "", false, targets...)
}

func (f *RoleFactory) NewWrappedX509(role driver.IdentityRole, ignoreRemote bool) (identity.Role, error) {
	return f.newX509WithType(role, X509Identity, ignoreRemote)
}

func (f *RoleFactory) newX509WithType(role driver.IdentityRole, identityType string, ignoreRemote bool, targets ...driver.Identity) (identity.Role, error) {
	f.Logger.Debugf("create x509 role for [%s]", driver.IdentityRoleStrings[role])

	kmp := x5092.NewKeyManagerProvider(f.Config, RoleToMSPID[role], f.SignerService, ignoreRemote)

	identityDB, err := f.StorageProvider.OpenIdentityDB(f.TMSID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get wallet path storage")
	}
	lm := common.NewLocalMembership(
		logging.MustGetLogger("token-sdk.services.identity.msp.x509"),
		f.Config,
		f.NetworkDefaultIdentity,
		f.SignerService,
		f.DeserializerManager,
		identityDB,
		f.BinderService,
		RoleToMSPID[role],
		kmp,
		false,
	)
	identities, err := f.IdentitiesForRole(role)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identities for role [%d]", role)
	}
	if err := lm.Load(identities, targets); err != nil {
		return nil, errors.WithMessage(err, "failed to load identities")
	}

	return &WrappingBindingRole{
		Role:             common.NewLongTermRole(f.Logger, role, f.TMSID.Network, f.FSCIdentity, lm),
		IdentityType:     identityType,
		RootIdentity:     f.FSCIdentity,
		IdentityProvider: f.IdentityProvider,
		BinderService:    f.BinderService,
	}, nil
}

// IdentitiesForRole returns the configured identities for the passed role
func (f *RoleFactory) IdentitiesForRole(role driver.IdentityRole) ([]*config.Identity, error) {
	return f.Config.IdentitiesForRole(role)
}

// BindingRole returns a new role wrapping the passed one.
// Identities will be bound to the long term identities this factory refers to.
func (f *RoleFactory) BindingRole(role identity.Role) (identity.Role, error) {
	return &WrappingBindingRole{
		Role:             role,
		RootIdentity:     f.FSCIdentity,
		IdentityProvider: f.IdentityProvider,
		BinderService:    f.BinderService,
	}, nil
}

type WrappingBindingRole struct {
	identity.Role
	IdentityType identity.Type

	RootIdentity     driver.Identity
	IdentityProvider driver2.IdentityProvider
	BinderService    driver2.BinderService
}

func (r *WrappingBindingRole) GetIdentityInfo(id string) (driver.IdentityInfo, error) {
	info, err := r.Role.GetIdentityInfo(id)
	if err != nil {
		return nil, err
	}
	return &WrappingBindingInfo{
		IdentityInfo:     info,
		IdentityType:     r.IdentityType,
		RootIdentity:     r.RootIdentity,
		IdentityProvider: r.IdentityProvider,
		BinderService:    r.BinderService,
	}, nil
}

// WrappingBindingInfo wraps a driver.IdentityInfo to further register the audit info,
// and binds the new identity to the default FSC node identity
type WrappingBindingInfo struct {
	driver.IdentityInfo
	IdentityType identity.Type

	RootIdentity     driver.Identity
	IdentityProvider driver2.IdentityProvider
	BinderService    driver2.BinderService
}

func (i *WrappingBindingInfo) ID() string {
	return i.IdentityInfo.ID()
}

func (i *WrappingBindingInfo) EnrollmentID() string {
	return i.IdentityInfo.EnrollmentID()
}

func (i *WrappingBindingInfo) Get() (driver.Identity, []byte, error) {
	// get the identity
	id, ai, err := i.IdentityInfo.Get()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get root identity")
	}
	// register the audit info
	if err := i.IdentityProvider.RegisterAuditInfo(id, ai); err != nil {
		return nil, nil, errors.Wrapf(err, "failed to register audit info for identity [%s]", id)
	}
	// bind the identity to the default FSC node identity
	if i.BinderService != nil {
		if err := i.BinderService.Bind(i.RootIdentity, id, false); err != nil {
			return nil, nil, errors.Wrapf(err, "failed to bind identity [%s] to [%s]", id, i.RootIdentity)
		}
	}
	// wrap the backend identity, and bind it
	if len(i.IdentityType) != 0 {
		typedIdentity, err := identity.WrapWithType(i.IdentityType, id)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to wrap identity [%s]", i.IdentityType)
		}
		if i.BinderService != nil {
			if err := i.BinderService.Bind(id, typedIdentity, true); err != nil {
				return nil, nil, errors.Wrapf(err, "failed to bind identity [%s] to [%s]", typedIdentity, id)
			}
			if err := i.BinderService.Bind(i.RootIdentity, typedIdentity, false); err != nil {
				return nil, nil, errors.Wrapf(err, "failed to bind identity [%s] to [%s]", typedIdentity, i.RootIdentity)
			}
		} else {
			// register at the list the audit info
			if err := i.IdentityProvider.RegisterAuditInfo(typedIdentity, ai); err != nil {
				return nil, nil, errors.Wrapf(err, "failed to register audit info for identity [%s]", id)
			}
		}
		id = typedIdentity
	}
	return id, ai, nil
}
