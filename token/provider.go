/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

var (
	managementServiceProviderIndex = &ManagementServiceProvider{}
)

// Normalizer is used to set default values of ServiceOptions struct, if needed.
type Normalizer interface {
	// Normalize normalizes the given ServiceOptions struct.
	Normalize(opt *ServiceOptions) *ServiceOptions
}

// VaultProvider provides token vault instances
type VaultProvider interface {
	// Vault returns a token vault instance for the passed inputs
	Vault(network string, channel string, namespace string) driver.Vault
}

// SelectorManager handles token selection operations
type SelectorManager interface {
	// NewSelector returns a new Selector instance bound the passed id.
	NewSelector(id string) (Selector, error)
	// Unlock unlocks the tokens bound to the passed id, if any
	Unlock(id string) error
}

// SelectorManagerProvider provides instances of SelectorManager
type SelectorManagerProvider interface {
	// SelectorManager returns a SelectorManager instance for the passed inputs.
	SelectorManager(network string, channel string, namespace string) SelectorManager
}

// CertificationClientProvider provides instances of CertificationClient
type CertificationClientProvider interface {
	// New returns a new CertificationClient instance for the passed inputs
	New(network string, channel string, namespace string, driver string) (driver.CertificationClient, error)
}

// ManagementServiceProvider provides instances of the management service
type ManagementServiceProvider struct {
	sp                          ServiceProvider
	tmsProvider                 driver.TokenManagerServiceProvider
	normalizer                  Normalizer
	certificationClientProvider CertificationClientProvider
	selectorManagerProvider     SelectorManagerProvider
	vaultProvider               VaultProvider
}

// NewManagementServiceProvider returns a new instance of ManagementServiceProvider
func NewManagementServiceProvider(
	sp ServiceProvider,
	tmsProvider driver.TokenManagerServiceProvider,
	normalizer Normalizer,
	vaultProvider VaultProvider,
	certificationClientProvider CertificationClientProvider,
	selectorManagerProvider SelectorManagerProvider,
) *ManagementServiceProvider {
	return &ManagementServiceProvider{
		sp:                          sp,
		tmsProvider:                 tmsProvider,
		normalizer:                  normalizer,
		vaultProvider:               vaultProvider,
		certificationClientProvider: certificationClientProvider,
		selectorManagerProvider:     selectorManagerProvider,
	}
}

// GetManagementService returns an instance of the management service for the passed options.
// If the management service has not been created yet, it will be created.
func (p *ManagementServiceProvider) GetManagementService(opts ...ServiceOption) *ManagementService {
	opt, err := CompileServiceOptions(opts...)
	if err != nil {
		panic(err)
	}
	opt = p.normalizer.Normalize(opt)

	logger.Debugf("get tms for [%s,%s,%s]", opt.Network, opt.Channel, opt.Namespace)
	tokenService, err := p.tmsProvider.GetTokenManagerService(
		opt.Network,
		opt.Channel,
		opt.Namespace,
		opt.PublicParamsFetcher,
	)
	if err != nil {
		logger.Errorf("failed getting TMS for [%s]: [%s]", opt.TMSID(), err)
		return nil
	}

	logger.Debugf("returning tms for [%s,%s,%s]", opt.Network, opt.Channel, opt.Namespace)

	return &ManagementService{
		sp:                          p.sp,
		network:                     opt.Network,
		channel:                     opt.Channel,
		namespace:                   opt.Namespace,
		tms:                         tokenService,
		vaultProvider:               p.vaultProvider,
		certificationClientProvider: p.certificationClientProvider,
		selectorManagerProvider:     p.selectorManagerProvider,
		signatureService: &SignatureService{
			deserializer: tokenService,
			ip:           tokenService.IdentityProvider(),
		},
	}
}

// GetManagementServiceProvider returns the management service provider from the passed service provider
func GetManagementServiceProvider(sp ServiceProvider) *ManagementServiceProvider {
	s, err := sp.GetService(managementServiceProviderIndex)
	if err != nil {
		panic(err)
	}
	return s.(*ManagementServiceProvider)
}
