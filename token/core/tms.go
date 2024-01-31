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
func (m *TMSProvider) GetTokenManagerService(opts driver.ServiceOptions) (service driver.TokenManagerService, err error) {
	if len(opts.Network) == 0 {
		return nil, errors.Errorf("network not specified")
	}
	if len(opts.Namespace) == 0 {
		return nil, errors.Errorf("namespace not specified")
	}
	m.lock.Lock()
	invokeCallback := false
	defer func() {
		// unlock
		m.lock.Unlock()
		// invoke callback
		if invokeCallback && m.callbackFunc != nil {
			err = m.callbackFunc(service, opts.Network, opts.Channel, opts.Namespace)
			if err != nil {
				logger.Fatalf("failed to initialize tms for [%s]: [%s]", opts, err)
			}
		}
	}()

	key := opts.Network + opts.Channel + opts.Namespace
	var ok bool
	service, ok = m.services[key]
	if !ok {
		logger.Debugf("creating new token manager service for [%s:%s:%s] with key [%s]", opts.Network, opts.Channel, opts.Namespace, key)
		var err error
		service, err = m.newTMS(&opts)
		if err != nil {
			return nil, err
		}
		m.services[key] = service
		invokeCallback = true
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
	logger.Debugf("creating new token manager service for [%s:%s:%s]", opts.Network, opts.Channel, opts.Namespace)

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

	key := opts.Network + opts.Channel + opts.Namespace
	logger.Debugf("update tms for [%s:%s:%s] with key [%s]", opts.Network, opts.Channel, opts.Namespace, key)
	service, ok := m.services[key]
	if ok {
		// if the public params identifiers are the same, then just pass the new public params
		newPP, err := SerializedPublicParametersFromBytes(opts.PublicParams)
		if err != nil {
			return errors.WithMessage(err, "failed unmarshalling public parameters")
		}
		oldPP := service.PublicParamsManager().PublicParameters()
		if oldPP == nil || (oldPP != nil && newPP.Identifier == oldPP.Identifier()) {
			return service.PublicParamsManager().SetPublicParameters(opts.PublicParams)
		}

		// if the public params identifiers are NOT the same, then unload the current instance
		// and create the new one with the new public params
		panic("not implemented yet")
	}

	// instantiate the token management service
	panic("not implemented yet")
}

func (m *TMSProvider) newTMS(opts *driver.ServiceOptions) (driver.TokenManagerService, error) {
	driverName, err := m.driverFor(opts)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get driver for [%s:%s:%s]", opts.Network, opts.Channel, opts.Namespace)
	}
	d, ok := drivers[driverName]
	if !ok {
		return nil, errors.Errorf("failed instantiate token service, driver [%s] not found", driverName)
	}
	logger.Debugf("instantiating token service for [%s:%s:%s], with driver identifier [%s]", opts.Network, opts.Channel, opts.Namespace, driverName)

	ts, err := d.NewTokenService(m.sp, opts.Network, opts.Channel, opts.Namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to instantiate token service for [%s:%s:%s]", opts.Network, opts.Channel, opts.Namespace)
	}

	return ts, nil
}

func (m *TMSProvider) driverFor(opts *driver.ServiceOptions) (string, error) {
	pp, err := m.loadPublicParams(opts)
	if err != nil {
		// resort to configuration
		tmsConfig, err2 := config.NewTokenSDK(m.configProvider).GetTMS(opts.Network, opts.Channel, opts.Namespace)
		if err2 != nil {
			return "", errors.WithMessagef(err, "failed to identify driver from the configuration of [%s:%s:%s], loading driver from public parameters failed too [%s]", opts.Network, opts.Channel, opts.Namespace, err)
		}

		driverName := tmsConfig.TMS().Driver
		if len(driverName) != 0 {
			return driverName, nil
		}
		return "", errors.WithMessagef(err, "failed to identify driver for [%s:%s:%s]", opts.Network, opts.Channel, opts.Namespace)
	}
	return pp.Identifier, nil
}

func (m *TMSProvider) loadPublicParams(opts *driver.ServiceOptions) (*driver.SerializedPublicParameters, error) {
	var ppRaw []byte
	var err error
	if len(opts.PublicParams) != 0 {
		ppRaw = opts.PublicParams
	} else {
		ppRaw, err = m.vault.PublicParams(opts.Network, opts.Channel, opts.Namespace)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to load public params from the vault")
		}
		if len(ppRaw) == 0 {
			ppRaw, err = opts.PublicParamsFetcher.Fetch()
			if err != nil {
				return nil, errors.WithMessage(err, "failed fetching public parameters")
			}
		}
	}

	// deserialize public params
	pp, err := SerializedPublicParametersFromBytes(ppRaw)
	if err != nil {
		return nil, errors.WithMessage(err, "failed unmarshalling public parameters")
	}
	return pp, nil
}
