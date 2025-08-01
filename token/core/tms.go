/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"bytes"
	"context"
	"os"
	"runtime/debug"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

type CallbackFunc func(tms driver.TokenManagerService, network, channel, namespace string) error

type PublicParametersStorage interface {
	PublicParams(ctx context.Context, networkID string, channel string, namespace string) ([]byte, error)
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
	logger                  logging.Logger
	configProvider          ConfigProvider
	publicParametersStorage PublicParametersStorage
	callback                CallbackFunc
	tokenDriverService      *TokenDriverService

	lock     sync.RWMutex
	services map[string]driver.TokenManagerService
}

func NewTMSProvider(logger logging.Logger, configProvider ConfigProvider, pps PublicParametersStorage, tokenDriverService *TokenDriverService) *TMSProvider {
	ms := &TMSProvider{
		logger:                  logger,
		configProvider:          configProvider,
		publicParametersStorage: pps,
		services:                map[string]driver.TokenManagerService{},
		tokenDriverService:      tokenDriverService,
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
		// update only if the public params are different from the current
		if bytes.Equal(service.PublicParamsManager().PublicParamsHash(), hash.Hashable(opts.PublicParams).Raw()) {
			m.logger.Debugf("service found, no need to update token management system for [%s:%s:%s] for key [%s], public params are the same", opts.Network, opts.Channel, opts.Namespace, key)
			return nil
		}

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
	return err
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
	ppRaw, err := m.loadPublicParams(opts)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get driver for [%s]", opts)
	}
	opts.PublicParams = ppRaw
	m.logger.Debugf("instantiating token service for [%s]", opts)

	ts, err := m.tokenDriverService.NewTokenService(driver.TMSID{Network: opts.Network, Channel: opts.Channel, Namespace: opts.Namespace}, opts.PublicParams)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to instantiate token service for [%s]", opts)
	}

	return ts, nil
}

func (m *TMSProvider) loadPublicParams(opts *driver.ServiceOptions) ([]byte, error) {
	// priorities:
	// 1. opts.PublicParams
	// 2. publicParametersStorage
	// 3. local configuration
	// 4. public parameters fetcher, if any
	for _, retriever := range []func(options *driver.ServiceOptions) ([]byte, error){m.ppFromOpts, m.ppFromStorage, m.ppFromConfig, m.ppFromFetcher} {
		if ppRaw, err := retriever(opts); err != nil {
			m.logger.Warnf("failed to retrieve params for [%s]: [%s]", opts, err)
		} else if len(ppRaw) != 0 {
			return ppRaw, nil
		}
	}
	m.logger.Errorf("cannot retrieve public params for [%s]: [%s]", opts, string(debug.Stack()))
	return nil, errors.Errorf("cannot retrieve public params for [%s]", opts)
}

func (m *TMSProvider) ppFromOpts(opts *driver.ServiceOptions) ([]byte, error) {
	if len(opts.PublicParams) != 0 {
		return opts.PublicParams, nil
	}
	return nil, errors.Errorf("public parameter not found in options")
}

func (m *TMSProvider) ppFromStorage(opts *driver.ServiceOptions) ([]byte, error) {
	ppRaw, err := m.publicParametersStorage.PublicParams(context.Background(), opts.Network, opts.Channel, opts.Namespace)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load public params from the publicParametersStorage")
	}
	if len(ppRaw) == 0 {
		return nil, errors.Errorf("no public parames found in publicParametersStorage")
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
		ppRaw, err := opts.PublicParamsFetcher.Fetch()
		if err != nil {
			return nil, errors.WithMessage(err, "failed fetching public parameters")
		}
		if len(ppRaw) == 0 {
			return nil, errors.Errorf("no public parames found in publicParametersStorage")
		}
		opts.PublicParams = ppRaw
		return ppRaw, nil
	}
	return nil, errors.Errorf("no public params fetched available")
}

func tmsKey(opts driver.ServiceOptions) string {
	return opts.Network + opts.Channel + opts.Namespace
}
