/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.driver.identity.tms")

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

// IdentityType is the type of identity
type IdentityType int

const (
	// LongTermIdentity is the type of the long term identity (x509)
	LongTermIdentity IdentityType = iota
	// AnonymousIdentity is the type of the long term anonymous identity from
	// which anonymous and unlikable pseudonyms are derived (Idemix)
	AnonymousIdentity
)

// LocalMembership is the interface for local membership
type LocalMembership interface {
	Load(owners []*config.Identity) error
	DefaultNetworkIdentity() view.Identity
	IsMe(id view.Identity) bool
	GetIdentityInfo(label string, auditInfo []byte) (driver.IdentityInfo, error)
	GetIdentifier(id view.Identity) (string, error)
	GetDefaultIdentifier() string
	RegisterIdentity(id string, path string) error
}

type NewLocalMembershipFunc = func(
	sp view2.ServiceProvider,
	configManager config.Manager,
	defaultNetworkIdentity view.Identity,
	signerService common.SignerService,
	binderService common.BinderService,
	deserializerManager common.DeserializerManager,
	mspID string,
) (LocalMembership, error)

// Role contains information about an identity role
type Role struct {
	IdentityType    IdentityType
	LocalMembership LocalMembership
}

// WalletManager is a manager of wallets.
// The first thing is to assign identity types to roles using SetRoleIdentityType.
// Then, the wallets can be loaded from the configuration using Load.
// Finally, the wallets can be obtained using Wallets.
type WalletManager struct {
	sp                     view2.ServiceProvider
	networkID              string
	configManager          config.Manager
	fscIdentity            view.Identity
	networkDefaultIdentity view.Identity
	signerService          common.SignerService
	binderService          common.BinderService

	roles       map[driver.IdentityRole]*Role
	lmProviders map[IdentityType]NewLocalMembershipFunc
}

// NewWalletManager creates a new wallet manager
func NewWalletManager(
	sp view2.ServiceProvider,
	networkID string,
	configManager config.Manager,
	fscIdentity view.Identity,
	networkDefaultIdentity view.Identity,
	signerService common.SignerService,
	binderService common.BinderService,
) *WalletManager {
	return &WalletManager{
		sp:                     sp,
		networkID:              networkID,
		configManager:          configManager,
		fscIdentity:            fscIdentity,
		networkDefaultIdentity: networkDefaultIdentity,
		signerService:          signerService,
		binderService:          binderService,
		roles:                  map[driver.IdentityRole]*Role{},
		lmProviders: map[IdentityType]NewLocalMembershipFunc{
			LongTermIdentity: func(sp view2.ServiceProvider, configManager config.Manager, defaultNetworkIdentity view.Identity, signerService common.SignerService, binderService common.BinderService, deserializerManager common.DeserializerManager, mspID string) (LocalMembership, error) {
				return x509.NewLocalMembership(
					configManager,
					networkDefaultIdentity,
					signerService,
					binderService,
					deserializerManager,
					mspID,
				), nil
			},
			AnonymousIdentity: func(sp view2.ServiceProvider, configManager config.Manager, defaultNetworkIdentity view.Identity, signerService common.SignerService, binderService common.BinderService, deserializerManager common.DeserializerManager, mspID string) (LocalMembership, error) {
				return idemix.NewLocalMembership(
					sp,
					configManager,
					networkDefaultIdentity,
					signerService,
					binderService,
					deserializerManager,
					mspID,
				), nil
			},
		},
	}

}

// Load loads the wallets defined in the configuration
func (wm *WalletManager) Load() error {
	logger.Debugf("load wallets...")
	if len(wm.roles) < 4 {
		return errors.New("missing roles")
	}
	defer logger.Debugf("load wallets...done")

	tmsConfig := wm.configManager.TMS()
	if tmsConfig.Wallets == nil {
		logger.Warnf("No wallets found in config")
		tmsConfig.Wallets = &config.Wallets{}
	}

	if err := wm.load(driver.OwnerRole, OwnerMSPID, wm.configManager.TMS().Wallets.Owners); err != nil {
		return errors.Wrap(err, "failed to load owners")
	}

	if err := wm.load(driver.IssuerRole, IssuerMSPID, wm.configManager.TMS().Wallets.Issuers); err != nil {
		return errors.Wrap(err, "failed to load issuers")
	}

	if err := wm.load(driver.AuditorRole, AuditorMSPID, wm.configManager.TMS().Wallets.Auditors); err != nil {
		return errors.Wrap(err, "failed to load auditors")
	}

	if err := wm.load(driver.CertifierRole, CertifierMSPID, wm.configManager.TMS().Wallets.Certifiers); err != nil {
		return errors.Wrap(err, "failed to load certifiers")
	}

	return nil
}

// SetRoleIdentityType sets the identity type for the given role
func (wm *WalletManager) SetRoleIdentityType(role driver.IdentityRole, identityType IdentityType) {
	wm.roles[role] = &Role{IdentityType: identityType}
}

// Wallets returns the wallets for each role
func (wm *WalletManager) Wallets() (identity.Wallets, error) {
	wallets := identity.NewWallets()

	for _, role := range []driver.IdentityRole{driver.IssuerRole, driver.AuditorRole, driver.OwnerRole, driver.CertifierRole} {
		m, err := wm.newWallet(role)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create wallet for role [%d]", role)
		}
		wallets.Put(role, m)
	}
	return wallets, nil
}

func (wm *WalletManager) SetLocalMembershipProvider(it IdentityType, provider NewLocalMembershipFunc) {
	// update the provider for the given identity type
	wm.lmProviders[it] = provider
}

func (wm *WalletManager) GetLocalMembership(role driver.IdentityRole) (LocalMembership, error) {
	roleInfo, ok := wm.roles[role]
	if !ok {
		return nil, errors.Errorf("role [%d] not found", role)
	}
	if roleInfo.LocalMembership == nil {
		return nil, errors.Errorf("local membership for role [%d] not found", role)
	}
	return roleInfo.LocalMembership, nil
}

func (wm *WalletManager) newWallet(role driver.IdentityRole) (identity.Wallet, error) {
	r, ok := wm.roles[role]
	if !ok {
		return nil, errors.Errorf("missing role %d", role)
	}
	if r.LocalMembership == nil {
		return nil, errors.Errorf("missing local membership for role %d", role)
	}

	var m identity.Wallet
	switch r.IdentityType {
	case AnonymousIdentity:
		m = idemix.NewWallet(wm.networkID, wm.fscIdentity, r.LocalMembership)
	case LongTermIdentity:
		m = x509.NewWallet(wm.networkID, wm.fscIdentity, r.LocalMembership)
	default:
		return nil, errors.Errorf("unknown identity type %d", r.IdentityType)
	}
	return m, nil
}

func (wm *WalletManager) newLocalMembership(role driver.IdentityRole, mspID string, identities []*config.Identity) (LocalMembership, error) {
	r, ok := wm.roles[role]
	if !ok {
		return nil, errors.Errorf("missing role %d", role)
	}

	localMembershipProvider, ok := wm.lmProviders[r.IdentityType]
	if !ok {
		return nil, errors.Errorf("missing local membership provider for identity type %d", r.IdentityType)
	}
	lm, err := localMembershipProvider(
		wm.sp,
		wm.configManager,
		wm.networkDefaultIdentity,
		wm.signerService,
		wm.binderService,
		common.GetDeserializerManager(wm.sp),
		mspID,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create local membership for role [%d]", role)
	}
	logger.Debugf("Load local membership for role [%d] and MSP ID [%s] with identities [%+q]", role, mspID, identities)
	if err := lm.Load(identities); err != nil {
		return nil, errors.WithMessage(err, "failed to load owners")
	}
	return lm, nil
}

func (wm *WalletManager) load(role driver.IdentityRole, mspID string, identities []*config.Identity) error {
	logger.Debugf("load [%d] identities for role [%d]", len(identities), role)
	r, ok := wm.roles[role]
	if !ok {
		return errors.Errorf("missing role %d", role)
	}
	lm, err := wm.newLocalMembership(role, mspID, identities)
	if err != nil {
		return errors.WithMessage(err, "failed to load local manager")
	}
	r.LocalMembership = lm
	return nil
}
