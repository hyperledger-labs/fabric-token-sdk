/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.core")

type CallbackFunc func(tms driver.TokenManagerService, network, channel, namespace string) error

type Vault interface {
	PublicParams(networkID string, channel string, namespace string) ([]byte, error)
}

type ConfigProvider interface {
	UnmarshalKey(key string, rawVal interface{}) error
	IsSet(key string) bool
	TranslatePath(path string) string
}

// TMSProvider is a token management service provider.
// It is responsible for creating token management services for different networks.
type TMSProvider struct {
	sp             view.ServiceProvider
	configProvider ConfigProvider
	vault          Vault
	callbackFunc   CallbackFunc

	lock     sync.Mutex
	services map[string]driver.TokenManagerService
}

func NewTMSProvider(sp view.ServiceProvider, configProvider ConfigProvider, vault Vault, callbackFunc CallbackFunc) *TMSProvider {
	ms := &TMSProvider{
		sp:             sp,
		configProvider: configProvider,
		vault:          vault,
		callbackFunc:   callbackFunc,
		services:       map[string]driver.TokenManagerService{},
	}
	return ms
}

// GetTokenManagerService returns a driver.TokenManagerService instance for the passed parameters.
// If a TokenManagerService is not available, it creates one by first fetching the public parameters using the passed driver.PublicParamsFetcher.
// If no driver is registered for the public params identifier, it returns an error.
func (m *TMSProvider) GetTokenManagerService(opts driver.ServiceOptions) (driver.TokenManagerService, error) {
	if len(opts.Network) == 0 {
		return nil, errors.Errorf("network not specified")
	}
	if len(opts.Namespace) == 0 {
		return nil, errors.Errorf("namespace not specified")
	}
	m.lock.Lock()
	defer m.lock.Unlock()

	key := opts.Network + opts.Channel + opts.Network
	service, ok := m.services[key]
	if !ok {
		if opts.PublicParamsFetcher == nil {
			return nil, errors.Errorf("public params fetcher not specified")
		}
		logger.Debugf("creating new token manager service for network %s, channel %s, namespace %s", opts.Network, opts.Channel, opts.Network)

		var err error
		service, err = m.newTMS(&opts)
		if err != nil {
			return nil, err
		}
		m.services[key] = service
	}
	return service, nil
}

func (m *TMSProvider) NewTokenManagerService(opts driver.ServiceOptions) (driver.TokenManagerService, error) {
	if len(opts.Network) == 0 {
		return nil, errors.Errorf("network not specified")
	}
	if len(opts.Namespace) == 0 {
		return nil, errors.Errorf("namespace not specified")
	}
	if opts.PublicParamsFetcher == nil {
		return nil, errors.Errorf("public params fetcher not specified")
	}
	logger.Debugf("creating new token manager service for network %s, channel %s, namespace %s", opts.Network, opts.Channel, opts.Network)

	service, err := m.newTMS(&opts)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func (m *TMSProvider) Update(opts driver.ServiceOptions) error {
	if len(opts.Network) == 0 {
		return errors.Errorf("network not specified")
	}
	if len(opts.Namespace) == 0 {
		return errors.Errorf("namespace not specified")
	}
	if len(opts.PublicParams) == 0 {
		return errors.Errorf("public params not specified")
	}
	m.lock.Lock()
	defer m.lock.Unlock()

	key := opts.Network + opts.Channel + opts.Network
	service, ok := m.services[key]
	if ok {
		return service.PublicParamsManager().SetPublicParameters(opts.PublicParams)
	}
	panic("not implemented yet")
}

func (m *TMSProvider) newTMS(opts *driver.ServiceOptions) (driver.TokenManagerService, error) {
	driverName, err := m.driverFor(opts)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get driver for [%s:%s:%s]", opts.Network, opts.Channel, opts.Network)
	}
	d, ok := drivers[driverName]
	if !ok {
		return nil, errors.Errorf("failed instantiate token service, driver [%s] not found", driverName)
	}
	logger.Debugf("instantiating token service for network [%s], channel [%s], namespace [%s], with driver identifier [%s]", opts.Network, opts.Channel, opts.Network, driverName)

	ts, err := d.NewTokenService(m.sp, opts.PublicParamsFetcher, opts.Network, opts.Channel, opts.Network)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to instantiate token service for [%s:%s:%s]", opts.Network, opts.Channel, opts.Network)
	}

	if m.callbackFunc != nil {
		if err := m.callbackFunc(ts, opts.Network, opts.Channel, opts.Network); err != nil {
			return nil, err
		}
	}
	return ts, nil
}

func (m *TMSProvider) driverFor(opts *driver.ServiceOptions) (string, error) {
	pp, err := m.loadPublicParams(opts)
	if err != nil {
		// resort to configuration
		tmsConfig, err2 := config.NewTokenSDK(m.configProvider).GetTMS(opts.Network, opts.Channel, opts.Network)
		if err2 != nil {
			return "", errors.WithMessagef(err, "failed to identify driver from the configuration of [%s:%s:%s], loading driver from public parameters failed too [%s]", opts.Network, opts.Channel, opts.Network, err)
		}

		driverName := tmsConfig.TMS().Driver
		if len(driverName) != 0 {
			return driverName, nil
		}
		return "", errors.WithMessagef(err, "failed to identify driver for [%s:%s:%s]", opts.Network, opts.Channel, opts.Network)
	}
	return pp.Identifier, nil
}

func (m *TMSProvider) loadPublicParams(opts *driver.ServiceOptions) (*driver.SerializedPublicParameters, error) {
	ppRaw, err := m.vault.PublicParams(opts.Network, opts.Channel, opts.Network)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load public params from the vault")
	}
	if len(ppRaw) == 0 {
		ppRaw, err = opts.PublicParamsFetcher.Fetch()
		if err != nil {
			return nil, errors.WithMessage(err, "failed fetching public parameters")
		}
	}
	pp, err := SerializedPublicParametersFromBytes(ppRaw)
	if err != nil {
		return nil, errors.WithMessage(err, "failed unmarshalling public parameters")
	}
	return pp, nil
}
