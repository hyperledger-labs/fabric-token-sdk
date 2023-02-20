/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"github.com/IBM/idemix/common/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
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

// RoleToMSPID maps the role to the MSP ID
var RoleToMSPID = map[driver.IdentityRole]string{
	driver.OwnerRole:     OwnerMSPID,
	driver.IssuerRole:    IssuerMSPID,
	driver.AuditorRole:   AuditorMSPID,
	driver.CertifierRole: CertifierMSPID,
}

// WalletFactory is the factory for creating wallets, idemix and x509
type WalletFactory struct {
	SP                     view.ServiceProvider
	NetworkID              string
	ConfigManager          config.Manager
	FSCIdentity            view2.Identity
	NetworkDefaultIdentity view2.Identity
	SignerService          common.SignerService
	BinderService          common.BinderService
}

// NewWalletFactory creates a new WalletFactory
func NewWalletFactory(
	sp view.ServiceProvider,
	networkID string,
	configManager config.Manager,
	fscIdentity view2.Identity,
	networkDefaultIdentity view2.Identity,
	signerService common.SignerService,
	binderService common.BinderService,
) *WalletFactory {
	return &WalletFactory{
		SP:                     sp,
		NetworkID:              networkID,
		ConfigManager:          configManager,
		FSCIdentity:            fscIdentity,
		NetworkDefaultIdentity: networkDefaultIdentity,
		SignerService:          signerService,
		BinderService:          binderService,
	}
}

// NewIdemixWallet creates a new Idemix wallet
func (f *WalletFactory) NewIdemixWallet(role driver.IdentityRole, cacheSize int) (identity.Wallet, error) {
	identities, err := f.ConfigFor(role)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identities for role [%d]", role)
	}

	dm, err := common.GetDeserializerManager(f.SP)
	if err != nil {
		return nil, err
	}
	lm := idemix.NewLocalMembership(
		f.SP,
		f.ConfigManager,
		f.NetworkDefaultIdentity,
		f.SignerService,
		f.BinderService,
		dm,
		kvs.GetService(f.SP),
		RoleToMSPID[role],
		cacheSize,
	)
	if err := lm.Load(identities); err != nil {
		return nil, errors.WithMessage(err, "failed to load owners")
	}
	return idemix.NewWallet(f.NetworkID, f.FSCIdentity, lm), nil
}

// NewX509Wallet creates a new X509 wallet
func (f *WalletFactory) NewX509Wallet(role driver.IdentityRole) (identity.Wallet, error) {
	identities, err := f.ConfigFor(role)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identities for role [%d]", role)
	}
	dm, err := common.GetDeserializerManager(f.SP)
	if err != nil {
		return nil, err
	}
	lm := x509.NewLocalMembership(
		f.ConfigManager,
		f.NetworkDefaultIdentity,
		f.SignerService,
		f.BinderService,
		dm,
		kvs.GetService(f.SP),
		RoleToMSPID[role],
	)
	if err := lm.Load(identities); err != nil {
		return nil, errors.WithMessage(err, "failed to load owners")
	}
	return x509.NewWallet(f.NetworkID, f.FSCIdentity, lm), nil
}

// ConfigFor returns the configured identities for the passed role
func (f *WalletFactory) ConfigFor(role driver.IdentityRole) ([]*config.Identity, error) {
	tmsConfig := f.ConfigManager.TMS()
	if tmsConfig.Wallets == nil {
		logger.Warnf("No wallets found in config")
		tmsConfig.Wallets = &config.Wallets{}
	}

	switch role {
	case driver.IssuerRole:
		return tmsConfig.Wallets.Issuers, nil
	case driver.AuditorRole:
		return tmsConfig.Wallets.Auditors, nil
	case driver.OwnerRole:
		return tmsConfig.Wallets.Owners, nil
	case driver.CertifierRole:
		return tmsConfig.Wallets.Certifiers, nil
	default:
		return nil, errors.Errorf("unknown role [%d]", role)
	}
}
