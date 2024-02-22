/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"os"
	"runtime/debug"
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
	GetString(key string) string
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

	key := tmsKey(opts)
	var ok bool
	service, ok = m.services[key]
	if !ok {
		logger.Debugf("creating new token manager service for [%s] with key [%s]", opts, key)
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

func tmsKey(opts driver.ServiceOptions) string {
	return opts.Network + opts.Channel + opts.Namespace
}

func (m *TMSProvider) NewTokenManagerService(opts driver.ServiceOptions) (driver.TokenManagerService, error) {
	if len(opts.Network) == 0 {
		return nil, errors.Errorf("network not specified")
	}
	if len(opts.Namespace) == 0 {
		return nil, errors.Errorf("namespace not specified")
	}
	logger.Debugf("creating new token manager service for [%s]", opts)

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
	m.lock.Lock()
	defer func() {
		m.lock.Unlock()

		// instantiate the token management service
		logger.Debugf("retrieve token management system for [%s]", opts)
		_, err = m.GetTokenManagerService(opts)
	}()

	key := tmsKey(opts)
	logger.Debugf("update tms for [%s] with key [%s]", opts, key)
	service, ok := m.services[key]
	if !ok {
		logger.Debugf("no service found, instantiate token management system for [%s:%s:%s] for key [%s]", opts.Network, opts.Channel, opts.Namespace, key)
		return
	}
	// if the public params identifiers are the same, then just pass the new public params
	var newPP *driver.SerializedPublicParameters
	newPP, err = SerializedPublicParametersFromBytes(opts.PublicParams)
	if err != nil {
		err = errors.WithMessage(err, "failed unmarshalling public parameters")
		return
	}
	oldPP := service.PublicParamsManager().PublicParameters()
	if oldPP == nil || (oldPP != nil && newPP.Identifier == oldPP.Identifier()) {
		logger.Debugf("same token driver identifier, update public parameters for [%s:%s:%s] with key [%s]", opts.Network, opts.Channel, opts.Namespace, key)
		return service.PublicParamsManager().SetPublicParameters(opts.PublicParams)
	}

	// if the public params identifiers are NOT the same, then unload the current instance
	// and create the new one with the new public params
	panic("not implemented yet")
}

func (m *TMSProvider) newTMS(opts *driver.ServiceOptions) (driver.TokenManagerService, error) {
	driverName, err := m.driverFor(opts)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get driver for [%s]", opts)
	}
	d, ok := drivers[driverName]
	if !ok {
		return nil, errors.Errorf("failed instantiate token service, driver [%s] not found", driverName)
	}
	logger.Debugf("instantiating token service for [%s], with driver identifier [%s]", opts, driverName)

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
			logger.Warnf("failed to retrieve params for [%s]: [%s]", opts, err)
		}
		if len(ppRaw) != 0 {
			break
		}
	}

	if len(ppRaw) == 0 {
		logger.Errorf("cannot retrive public params for [%s]: [%s]", opts, debug.Stack())
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
	tmsConfig, err := config.NewTokenSDK(m.configProvider).GetTMS(opts.Network, opts.Channel, opts.Namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to identify driver from the configuration of [%s], loading driver from public parameters failed too [%s]", opts, err)
	}
	cPP := tmsConfig.TMS().PublicParameters
	if cPP != nil && len(cPP.Path) != 0 {
		logger.Infof("load public parameters from [%s]...", cPP.Path)
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
			return nil, errors.Errorf("no public parames found in vault")
		}
		return ppRaw, nil
	}
	return nil, errors.Errorf("no public params fetched available")
}
