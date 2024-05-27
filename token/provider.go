/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
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
	logger                      logging.Logger
	tmsProvider                 driver.TokenManagerServiceProvider
	normalizer                  Normalizer
	certificationClientProvider CertificationClientProvider
	selectorManagerProvider     SelectorManagerProvider
	vaultProvider               VaultProvider
}

// NewManagementServiceProvider returns a new instance of ManagementServiceProvider
func NewManagementServiceProvider(
	logger logging.Logger,
	tmsProvider driver.TokenManagerServiceProvider,
	normalizer Normalizer,
	vaultProvider VaultProvider,
	certificationClientProvider CertificationClientProvider,
	selectorManagerProvider SelectorManagerProvider,
) *ManagementServiceProvider {
	return &ManagementServiceProvider{
		logger:                      logger,
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
	opt, err = p.normalize(opt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to normalize options")
	}

	p.logger.Debugf("get tms for [%s,%s,%s]", opt.Network, opt.Channel, opt.Namespace)

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

	p.logger.Debugf("returning tms for [%s,%s,%s]", opt.Network, opt.Channel, opt.Namespace)

	ms := &ManagementService{
		logger:                      logging.DeriveDriverLogger(p.logger, "", opt.Network, opt.Channel, opt.Namespace),
		network:                     opt.Network,
		channel:                     opt.Channel,
		namespace:                   opt.Namespace,
		tms:                         tokenService,
		vaultProvider:               p.vaultProvider,
		certificationClientProvider: p.certificationClientProvider,
		selectorManagerProvider:     p.selectorManagerProvider,
		signatureService: &SignatureService{
			deserializer: tokenService.Deserializer(),
			ip:           tokenService.IdentityProvider(),
		},
	}
	if err := ms.init(); err != nil {
		return nil, errors.WithMessagef(err, "failed to initialize token management service")
	}
	return ms, nil
}

func (p *ManagementServiceProvider) normalize(opt *ServiceOptions) (*ServiceOptions, error) {
	// lookup configurations
	configs, err := p.tmsProvider.Configs()
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting tms configs")
	}
	if len(configs) == 0 {
		return nil, errors.Errorf("no token management service configs found")
	}
	var config *config.TMS
	if len(opt.Network) == 0 {
		config = configs[0].TMS()
		opt.Network = config.Network
	} else {
		// search
		found := false
		for _, manager := range configs {
			if manager.TMS().Network == opt.Network {
				found = true
				config = manager.TMS()
			}
		}
		if !found {
			return nil, errors.Errorf("no token management service config found for network [%s]", opt.Network)
		}
	}

	if len(opt.Channel) == 0 {
		opt.Channel = config.Channel
	}
	if opt.Channel != config.Channel {
		// is there another TMS with the same network, but different channel? If yes, don't fail
		found := false
		for _, manager := range configs {
			tms := manager.TMS()
			if tms.Network == opt.Network && tms.Channel == opt.Channel {
				found = true
				config = tms
			}
		}
		if !found {
			return nil, errors.Errorf("invalid channel [%s], expected [%s]", opt.Channel, config.Channel)
		}
	}

	if len(opt.Namespace) == 0 {
		opt.Namespace = config.Namespace
	}
	if opt.Namespace != config.Namespace {
		// is there another TMS with the same network and channel, but different namespace? If yes, don't fail
		found := false
		for _, manager := range configs {
			tms := manager.TMS()
			if tms.Network == opt.Network && tms.Channel == opt.Channel && tms.Namespace == opt.Namespace {
				found = true
				config = tms
			}
		}
		if !found {
			return nil, errors.Errorf("invalid namespace [%s], expected [%s]", opt.Namespace, config.Namespace)
		}
	}

	// last pass
	return p.normalizer.Normalize(opt)
}

func (p *ManagementServiceProvider) Update(tmsID TMSID, val []byte) error {
	p.logger.Debugf("update tms [%s] with public params [%s]", tmsID, Hashable(val))
	err := p.tmsProvider.Update(driver.ServiceOptions{
		Network:      tmsID.Network,
		Channel:      tmsID.Channel,
		Namespace:    tmsID.Namespace,
		PublicParams: val,
	})
	if err != nil {
		return errors.Wrapf(err, "failed updating tms [%s]", tmsID)
	}
	p.logger.Debugf("update tms [%s] with public params [%s]...done", tmsID, Hashable(val))
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
