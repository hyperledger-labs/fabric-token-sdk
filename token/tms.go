/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"context"
	"fmt"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

var logger = logging.MustGetLogger()

// TMSID models a TMS identifier
type TMSID = driver.TMSID

// ServiceProvider is used to return instances of a given type
type ServiceProvider interface {
	// GetService returns an instance of the given type
	GetService(v interface{}) (interface{}, error)
}

// ManagementService (TMS, for short) is the entry point for the Token API. A TMS is uniquely
// identified by a network, channel, namespace, and public parameters.
// The TMS gives access, among other things, to the wallet manager, the public parameters,
// the token selector, and so on.
type ManagementService struct {
	network   string
	channel   string
	namespace string
	tms       driver.TokenManagerService

	vaultProvider               VaultProvider
	certificationClientProvider CertificationClientProvider
	selectorManagerProvider     SelectorManagerProvider
	signatureService            *SignatureService
	vault                       *Vault
	logger                      logging.Logger
}

// String returns a string representation of the TMS
func (t *ManagementService) String() string {
	return fmt.Sprintf("TMS[%s:%s:%s]", t.Network(), t.Channel(), t.Network())
}

// Network returns the network identifier
func (t *ManagementService) Network() string {
	return t.network
}

// Channel returns the channel identifier
func (t *ManagementService) Channel() string {
	return t.channel
}

// Namespace returns the namespace identifier, empty if not defined
func (t *ManagementService) Namespace() string {
	return t.namespace
}

// NewRequest returns a new Token Request whose anchor is the passed id
func (t *ManagementService) NewRequest(id RequestAnchor) (*Request, error) {
	return NewRequest(t, id), nil
}

// NewRequestFromBytes returns a new Token Request for the passed anchor, and whose actions and metadata are
// unmarshalled from the passed bytes
func (t *ManagementService) NewRequestFromBytes(anchor RequestAnchor, actions []byte, meta []byte) (*Request, error) {
	return NewRequestFromBytes(t, anchor, actions, meta)
}

// NewFullRequestFromBytes returns a new Token Request for the serialized version
func (t *ManagementService) NewFullRequestFromBytes(tr []byte) (*Request, error) {
	return NewFullRequestFromBytes(t, tr)
}

// NewMetadataFromBytes unmarshals the passed bytes into a Metadata object
func (t *ManagementService) NewMetadataFromBytes(raw []byte) (*Metadata, error) {
	tokenRequestMetadata := &driver.TokenRequestMetadata{}
	if err := tokenRequestMetadata.FromBytes(raw); err != nil {
		return nil, err
	}
	return &Metadata{
		TokenService:         t.tms.TokensService(),
		WalletService:        t.tms.WalletService(),
		TokenRequestMetadata: tokenRequestMetadata,
		Logger:               t.logger,
	}, nil
}

// Validator returns a new token validator for this TMS
func (t *ManagementService) Validator() (*Validator, error) {
	v, err := t.tms.Validator()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get validator")
	}
	return &Validator{backend: v}, nil
}

// Vault returns the Token Vault for this TMS
func (t *ManagementService) Vault() *Vault {
	return t.vault
}

// WalletManager returns the wallet manager for this TMS
func (t *ManagementService) WalletManager() *WalletManager {
	return &WalletManager{managementService: t, walletService: t.tms.WalletService()}
}

// CertificationManager returns the certification manager for this TMS.
// It returns nil if certification is not supported.
func (t *ManagementService) CertificationManager() *CertificationManager {
	cs := t.tms.CertificationService()
	if cs == nil {
		return nil
	}
	return &CertificationManager{c: cs}
}

// CertificationClient returns the certification client for this TMS
func (t *ManagementService) CertificationClient(ctx context.Context) (*CertificationClient, error) {
	certificationClient, err := t.certificationClientProvider.New(ctx, nil)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create ceritifacation client")
	}
	return &CertificationClient{cc: certificationClient}, nil
}

// PublicParametersManager returns a manager that gives access to the public parameters
// governing this TMS.
func (t *ManagementService) PublicParametersManager() *PublicParametersManager {
	return &PublicParametersManager{ppm: t.tms.PublicParamsManager()}
}

// SelectorManager returns a manager that gives access to the token selectors
func (t *ManagementService) SelectorManager() (SelectorManager, error) {
	return t.selectorManagerProvider.SelectorManager(t)
}

// SigService returns the signature service for this TMS
func (t *ManagementService) SigService() *SignatureService {
	return t.signatureService
}

// ID returns the TMS identifier
func (t *ManagementService) ID() TMSID {
	return TMSID{
		Network:   t.Network(),
		Channel:   t.Channel(),
		Namespace: t.Namespace(),
	}
}

// Configuration returns the configuration for this TMS
func (t *ManagementService) Configuration() *Configuration {
	return &Configuration{cm: t.tms.Configuration()}
}

func (t *ManagementService) Authorization() *Authorization {
	return &Authorization{Authorization: t.tms.Authorization()}
}

func (t *ManagementService) init() error {
	v, err := t.vaultProvider.Vault(t.network, t.channel, t.namespace)
	if err != nil {
		return errors.WithMessagef(err, "failed to get vault for [%s:%s:%s]", t.namespace, t.channel, t.namespace)
	}
	t.vault = &Vault{v: v, logger: t.logger}
	return nil
}

func (t *ManagementService) TokensService() *TokensService {
	return &TokensService{ts: t.tms.TokensService(), tus: t.tms.TokensUpgradeService()}
}

// GetManagementService returns the management service for the passed options. If no options are passed,
// the default management service is returned.
// Options: WithNetwork, WithChannel, WithNamespace, WithPublicParameterFetcher, WithTMS, WithTMSID
// The function panics if an error occurs. Use GetManagementServiceProvider(sp).GetManagementService(opts...) to handle any error directly
func GetManagementService(sp ServiceProvider, opts ...ServiceOption) *ManagementService {
	ms, err := GetManagementServiceProvider(sp).GetManagementService(opts...)
	if err != nil {
		logger.Warnf("failed to get token manager service [%s]", err)
		return nil
	}
	return ms
}

func NewWalletManager(walletService driver.WalletService) *WalletManager {
	return &WalletManager{walletService: walletService}
}
