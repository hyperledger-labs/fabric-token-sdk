/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

var (
	managementServiceProviderIndex = &ManagementServiceProvider{}
)

// Normalizer is used to set default values of ServiceOptions struct, if needed.
type Normalizer interface {
	// Normalize normalizes the given ServiceOptions struct.
	Normalize(opt *ServiceOptions) (*ServiceOptions, error)
}

// VaultProvider provides token vault instances
type VaultProvider interface {
	// Vault returns a token vault instance for the passed inputs
	Vault(network string, channel string, namespace string) (driver.Vault, error)
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
	SelectorManager(tms *ManagementService) (SelectorManager, error)
}

// CertificationClientProvider provides instances of CertificationClient
type CertificationClientProvider interface {
	// New returns a new CertificationClient instance for the passed inputs
	New(tms *ManagementService) (driver.CertificationClient, error)
}

// ManagementServiceProvider provides instances of the management service
type ManagementServiceProvider struct {
	tmsProvider                 driver.TokenManagerServiceProvider
	normalizer                  Normalizer
	certificationClientProvider CertificationClientProvider
	selectorManagerProvider     SelectorManagerProvider
	vaultProvider               VaultProvider
}

// NewManagementServiceProvider returns a new instance of ManagementServiceProvider
func NewManagementServiceProvider(
	tmsProvider driver.TokenManagerServiceProvider,
	normalizer Normalizer,
	vaultProvider VaultProvider,
	certificationClientProvider CertificationClientProvider,
	selectorManagerProvider SelectorManagerProvider,
) *ManagementServiceProvider {
	return &ManagementServiceProvider{
		tmsProvider:                 tmsProvider,
		normalizer:                  normalizer,
		vaultProvider:               vaultProvider,
		certificationClientProvider: certificationClientProvider,
		selectorManagerProvider:     selectorManagerProvider,
	}
}

// GetManagementService returns an instance of the management service for the passed options.
// If the management service has not been created yet, it will be created.
func (p *ManagementServiceProvider) GetManagementService(opts ...ServiceOption) (*ManagementService, error) {
	return p.managementService(false, opts...)
}

// NewManagementService returns a new instance of the management service for the passed options.
func (p *ManagementServiceProvider) NewManagementService(opts ...ServiceOption) (*ManagementService, error) {
	return p.managementService(true, opts...)
}

func (p *ManagementServiceProvider) managementService(aNew bool, opts ...ServiceOption) (*ManagementService, error) {
	opt, err := CompileServiceOptions(opts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile options")
	}
	opt, err = p.normalizer.Normalize(opt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to normalize options")
	}

	logger.Debugf("get tms for [%s,%s,%s]", opt.Network, opt.Channel, opt.Namespace)

	var tokenService driver.TokenManagerService
	driverOpts := driver.ServiceOptions{
		Network:             opt.Network,
		Channel:             opt.Channel,
		Namespace:           opt.Namespace,
		PublicParamsFetcher: opt.PublicParamsFetcher,
		PublicParams:        opt.PublicParams,
		Params:              opt.Params,
	}
	if aNew {
		tokenService, err = p.tmsProvider.NewTokenManagerService(driverOpts)
	} else {
		tokenService, err = p.tmsProvider.GetTokenManagerService(driverOpts)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting TMS for [%s]", opt)
	}

	logger.Debugf("returning tms for [%s,%s,%s]", opt.Network, opt.Channel, opt.Namespace)

	ms := &ManagementService{
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
	if err := ms.init(); err != nil {
		return nil, errors.WithMessagef(err, "failed to initialize token management service")
	}
	return ms, nil
}

func (p *ManagementServiceProvider) Update(tmsID TMSID, val []byte) error {
	logger.Debugf("update tms [%s] with public params [%s]", tmsID, hash.Hashable(val))
	err := p.tmsProvider.Update(driver.ServiceOptions{
		Network:      tmsID.Network,
		Channel:      tmsID.Channel,
		Namespace:    tmsID.Namespace,
		PublicParams: val,
	})
	if err != nil {
		return errors.Wrapf(err, "failed updating tms [%s]", tmsID)
	}
	logger.Debugf("update tms [%s] with public params [%s]...done", tmsID, hash.Hashable(val))
	return nil
}

// GetManagementServiceProvider returns the management service provider from the passed service provider.
// The function panics if an error occurs.
// An alternative way is to use `s, err := sp.GetService(&ManagementServiceProvider{}) and catch the error manually.`
func GetManagementServiceProvider(sp ServiceProvider) *ManagementServiceProvider {
	s, err := sp.GetService(managementServiceProviderIndex)
	if err != nil {
		panic(err)
	}
	return s.(*ManagementServiceProvider)
}
