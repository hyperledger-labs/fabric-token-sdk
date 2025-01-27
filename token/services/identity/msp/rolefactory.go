/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/common"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
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
var RoleToMSPID = map[identity.RoleType]string{
	identity.OwnerRole:     OwnerMSPID,
	identity.IssuerRole:    IssuerMSPID,
	identity.AuditorRole:   AuditorMSPID,
	identity.CertifierRole: CertifierMSPID,
}

type StorageProvider interface {
	OpenIdentityDB(tmsID token.TMSID) (driver2.IdentityDB, error)
	NewKeystore() (identity.Keystore, error)
}

// RoleFactory is the factory for creating wallets, idemix and x509
type RoleFactory struct {
	Logger                 logging.Logger
	TMSID                  token.TMSID
	Config                 idriver.Config
	FSCIdentity            driver.Identity
	NetworkDefaultIdentity driver.Identity
	IdentityProvider       idriver.IdentityProvider
	SignerService          idriver.SigService
	BinderService          idriver.BinderService
	StorageProvider        StorageProvider
	DeserializerManager    idriver.DeserializerManager
	ignoreRemote           bool
}

// NewRoleFactory creates a new RoleFactory
func NewRoleFactory(
	logger logging.Logger,
	TMSID token.TMSID,
	config idriver.Config,
	fscIdentity driver.Identity,
	networkDefaultIdentity driver.Identity,
	identityProvider idriver.IdentityProvider,
	signerService idriver.SigService,
	binderService idriver.BinderService,
	storageProvider StorageProvider,
	deserializerManager idriver.DeserializerManager,
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
func (f *RoleFactory) NewIdemix(role identity.RoleType, cacheSize int, issuerPublicKey *crypto.IdemixIssuerPublicKey, additionalKMPs ...common.KeyManagerProvider) (identity.Role, error) {
	f.Logger.Debugf("create idemix role for [%s]", identity.RoleTypeStrings[role])
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
		true,
		f.IdentityProvider,
		kmps...,
	)
	identities, err := f.IdentitiesForRole(role)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identities for role [%d]", role)
	}
	if err := lm.Load(identities, nil); err != nil {
		return nil, errors.WithMessage(err, "failed to load identities")
	}
	return common.NewRole(f.Logger, role, f.TMSID.Network, f.FSCIdentity, lm), nil
}

// NewX509 creates a new X509-based role
func (f *RoleFactory) NewX509(role identity.RoleType, targets ...driver.Identity) (identity.Role, error) {
	return f.newX509WithType(role, "", false, targets...)
}

func (f *RoleFactory) NewWrappedX509(role identity.RoleType, ignoreRemote bool) (identity.Role, error) {
	return f.newX509WithType(role, X509Identity, ignoreRemote)
}

func (f *RoleFactory) newX509WithType(role identity.RoleType, identityType string, ignoreRemote bool, targets ...driver.Identity) (identity.Role, error) {
	f.Logger.Debugf("create x509 role for [%s]", identity.RoleTypeStrings[role])

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
		false,
		f.IdentityProvider,
		kmp,
	)
	identities, err := f.IdentitiesForRole(role)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identities for role [%d]", role)
	}
	if err := lm.Load(identities, targets); err != nil {
		return nil, errors.WithMessage(err, "failed to load identities")
	}

	return common.NewRole(f.Logger, role, f.TMSID.Network, f.FSCIdentity, lm), nil
}

// IdentitiesForRole returns the configured identities for the passed role
func (f *RoleFactory) IdentitiesForRole(role identity.RoleType) ([]*idriver.ConfiguredIdentity, error) {
	return f.Config.IdentitiesForRole(role)
}
