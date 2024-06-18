/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"context"
	"os"
	"runtime/debug"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

type CallbackFunc func(tms driver.TokenManagerService, network, channel, namespace string) error

type Vault interface {
	PublicParams(networkID string, channel string, namespace string) ([]byte, error)
}

type ConfigProvider interface {
	Configurations() ([]driver.Configuration, error)
	ConfigurationFor(network string, channel string, namespace string) (driver.Configuration, error)
}

type PublicParameters struct {
	Path string `yaml:"path"`
}

// TMSProvider is a token management service provider.
// It is responsible for creating token management services for different networks.
type TMSProvider struct {
	sp             view.ServiceProvider
	logger         logging.Logger
	configProvider ConfigProvider
	vault          Vault
	callback       CallbackFunc

	lock     sync.RWMutex
	services map[string]driver.TokenManagerService
}

func NewTMSProvider(sp view.ServiceProvider, logger logging.Logger, configProvider ConfigProvider, vault Vault) *TMSProvider {
	ms := &TMSProvider{
		sp:             sp,
		logger:         logger,
		configProvider: configProvider,
		vault:          vault,
		services:       map[string]driver.TokenManagerService{},
	}
	return ms
}

// GetTokenManagerService returns a driver.TokenManagerService instance for the passed parameters.
// If a TokenManagerService is not available, it creates one by first fetching the public parameters using the passed driver.PublicParamsFetcher.
// If no driver is registered for the public params identifier, it returns an error.
func (m *TMSProvider) GetTokenManagerService(opts driver.ServiceOptions) (service driver.TokenManagerService, err error) {
	if len(opts.Network) == 0 {
		return nil, errors.Errorf("network not specified")
	}
	if len(opts.Namespace) == 0 {
		return nil, errors.Errorf("namespace not specified")
	}

	key := tmsKey(opts)
	m.logger.Debugf("check existence token manager service for [%s] with key [%s]", opts, key)
	m.lock.RLock()
	service, ok := m.services[key]
	if ok {
		m.lock.RUnlock()
		return service, nil
	}
	m.lock.RUnlock()

	m.logger.Debugf("lock to create token manager service for [%s] with key [%s]", opts, key)

	m.lock.Lock()
	defer m.lock.Unlock()

	service, ok = m.services[key]
	if ok {
		m.logger.Debugf("token manager service for [%s] with key [%s] exists, return it", opts, key)
		return service, nil
	}

	m.logger.Debugf("creating new token manager service for [%s] with key [%s]", opts, key)
	service, err = m.getTokenManagerService(opts)
	if err != nil {
		return nil, err
	}
	m.services[key] = service
	return service, nil
}

func (m *TMSProvider) NewTokenManagerService(opts driver.ServiceOptions) (driver.TokenManagerService, error) {
	if len(opts.Network) == 0 {
		return nil, errors.Errorf("network not specified")
	}
	if len(opts.Namespace) == 0 {
		return nil, errors.Errorf("namespace not specified")
	}
	m.logger.Debugf("creating new token manager service for [%s]", opts)

	service, err := m.newTMS(&opts)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func (m *TMSProvider) Update(opts driver.ServiceOptions) (err error) {
	if len(opts.Network) == 0 {
		return errors.Errorf("network not specified")
	}
	if len(opts.Namespace) == 0 {
		return errors.Errorf("namespace not specified")
	}
	if len(opts.PublicParams) == 0 {
		return errors.Errorf("public params not specified")
	}

	key := tmsKey(opts)
	m.logger.Debugf("update tms for [%s] with key [%s]", opts, key)

	m.lock.Lock()
	defer m.lock.Unlock()
	service, ok := m.services[key]
	if !ok {
		m.logger.Debugf("no service found, instantiate token management system for [%s:%s:%s] for key [%s]", opts.Network, opts.Channel, opts.Namespace, key)
	} else {
		m.logger.Debugf("service found, unload token management system for [%s:%s:%s] for key [%s] and reload it", opts.Network, opts.Channel, opts.Namespace, key)
	}

	// create the service for the new public params
	newService, err := m.getTokenManagerService(opts)
	if err == nil {
		// unload the old service, if set
		if service != nil {
			if err := service.Done(); err != nil {
				return errors.WithMessage(err, "failed to unload token service")
			}
		}
		// register the new service
		m.services[key] = newService
	}
	return
}

func (m *TMSProvider) Configurations() ([]driver.Configuration, error) {
	tmsConfigs, err := m.configProvider.Configurations()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get token managers")
	}
	res := make([]driver.Configuration, len(tmsConfigs))
	copy(res, tmsConfigs)
	return res, nil
}

func (m *TMSProvider) SetCallback(callback CallbackFunc) {
	m.callback = callback
}

func (m *TMSProvider) getTokenManagerService(opts driver.ServiceOptions) (service driver.TokenManagerService, err error) {
	m.logger.Debugf("creating new token manager service for [%s]", opts)
	service, err = m.newTMS(&opts)
	if err != nil {
		return nil, err
	}
	// invoke callback
	if m.callback != nil {
		err = m.callback(service, opts.Network, opts.Channel, opts.Namespace)
		if err != nil {
			m.logger.Fatalf("failed to initialize tms for [%s]: [%s]", opts, err)
		}
	}
	return service, nil
}

func (m *TMSProvider) newTMS(opts *driver.ServiceOptions) (driver.TokenManagerService, error) {
	driverName, err := m.driverFor(opts)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get driver for [%s]", opts)
	}
	d, ok := holder.Drivers[driverName]
	if !ok {
		return nil, errors.Errorf("failed instantiate token service, driver [%s] not found", driverName)
	}
	m.logger.Debugf("instantiating token service for [%s], with driver identifier [%s]", opts, driverName)

	ts, err := d.NewTokenService(m.sp, opts.Network, opts.Channel, opts.Namespace, opts.PublicParams)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to instantiate token service for [%s]", opts)
	}

	return ts, nil
}

func (m *TMSProvider) driverFor(opts *driver.ServiceOptions) (string, error) {
	pp, err := m.loadPublicParams(opts)
	if err != nil {
		return "", errors.WithMessagef(err, "failed to identify driver for [%s]", opts)
	}
	return pp.Identifier, nil
}

func (m *TMSProvider) loadPublicParams(opts *driver.ServiceOptions) (*driver.SerializedPublicParameters, error) {
	// priorities:
	// 1. opts.PublicParams
	// 2. vault
	// 3. local configuration
	// 4. public parameters fetcher, if any
	var ppRaw []byte
	var err error

	for _, retriever := range []func(options *driver.ServiceOptions) ([]byte, error){m.ppFromOpts, m.ppFromVault, m.ppFromConfig, m.ppFromFetcher} {
		ppRaw, err = retriever(opts)
		if err != nil {
			m.logger.Warnf("failed to retrieve params for [%s]: [%s]", opts, err)
		}
		if len(ppRaw) != 0 {
			break
		}
	}

	if len(ppRaw) == 0 {
		m.logger.Errorf("cannot retrieve public params for [%s]: [%s]", opts, debug.Stack())
		return nil, errors.Errorf("cannot retrive public params for [%s]", opts)
	}

	// deserialize public params
	pp, err := SerializedPublicParametersFromBytes(ppRaw)
	if err != nil {
		return nil, errors.WithMessage(err, "failed unmarshalling public parameters")
	}

	return pp, nil
}

func (m *TMSProvider) ppFromOpts(opts *driver.ServiceOptions) ([]byte, error) {
	if len(opts.PublicParams) != 0 {
		return opts.PublicParams, nil
	}
	return nil, errors.Errorf("public parameter not found in options")
}

func (m *TMSProvider) ppFromVault(opts *driver.ServiceOptions) ([]byte, error) {
	ppRaw, err := m.vault.PublicParams(opts.Network, opts.Channel, opts.Namespace)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load public params from the vault")
	}
	if len(ppRaw) == 0 {
		return nil, errors.Errorf("no public parames found in vault")
	}
	return ppRaw, nil
}

func (m *TMSProvider) ppFromConfig(opts *driver.ServiceOptions) ([]byte, error) {
	tmsConfig, err := m.configProvider.ConfigurationFor(opts.Network, opts.Channel, opts.Namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to identify driver from the configuration of [%s], loading driver from public parameters failed too [%s]", opts, err)
	}
	cPP := &PublicParameters{}
	if err := tmsConfig.UnmarshalKey("publicParameters", cPP); err != nil {
		return nil, errors.WithMessage(err, "failed to unmarshal public parameters")
	}
	if len(cPP.Path) != 0 {
		m.logger.Infof("load public parameters from [%s]...", cPP.Path)
		ppRaw, err := os.ReadFile(cPP.Path)
		if err != nil {
			return nil, errors.Errorf("failed to load public parameters from [%s]: [%s]", cPP.Path, err)
		}
		return ppRaw, nil
	}
	return nil, errors.Errorf("no public params found in configuration")
}

func (m *TMSProvider) ppFromFetcher(opts *driver.ServiceOptions) ([]byte, error) {
	if opts.PublicParamsFetcher != nil {
		ppRaw, err := opts.PublicParamsFetcher.Fetch(context.TODO())
		if err != nil {
			return nil, errors.WithMessage(err, "failed fetching public parameters")
		}
		if len(ppRaw) == 0 {
			return nil, errors.Errorf("no public parames found in vault")
		}
		opts.PublicParams = ppRaw
		return ppRaw, nil
	}
	return nil, errors.Errorf("no public params fetched available")
}

func tmsKey(opts driver.ServiceOptions) string {
	return opts.Network + opts.Channel + opts.Namespace
}
