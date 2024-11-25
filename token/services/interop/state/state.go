/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	url2 "net/url"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/state/driver"
	"github.com/pkg/errors"
)

var (
	logger = flogging.MustGetLogger("token-sdk.services.interop.state")
	// TokenExistsError is returned when the token already exists
	TokenExistsError = errors.New("token exists")
	// TokenDoesNotExistError is returned when the token does not exist
	TokenDoesNotExistError = errors.New("token does not exists")
)

type ServiceProvider struct {
	sspsMu     sync.RWMutex
	ssps       map[string]driver.StateServiceProvider
	sspDrivers map[driver.SSPDriverName]driver.NamedSSPDriver
}

func NewServiceProvider() *ServiceProvider {
	return &ServiceProvider{
		ssps:       map[string]driver.StateServiceProvider{},
		sspDrivers: map[driver.SSPDriverName]driver.NamedSSPDriver{},
	}
}

func (p *ServiceProvider) RegisterDriver(driver driver.NamedSSPDriver) {
	logger.Debugf("register driver [%s]", driver.Name)
	p.sspDrivers[driver.Name] = driver
}

func (p *ServiceProvider) QueryExecutor(url string) (driver.StateQueryExecutor, error) {
	ssp, err := p.ssp(url)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get ssp for url [%s]", url)
	}
	return ssp.QueryExecutor(url)
}

func (p *ServiceProvider) Verifier(url string) (driver.StateVerifier, error) {
	ssp, err := p.ssp(url)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get ssp for url [%s]", url)
	}
	return ssp.Verifier(url)
}

func (p *ServiceProvider) URLToTMSID(url string) (token.TMSID, error) {
	ssp, err := p.ssp(url)
	if err != nil {
		return token.TMSID{}, errors.WithMessagef(err, "failed to get ssp for url [%s]", url)
	}
	return ssp.URLToTMSID(url)
}

func (p *ServiceProvider) ssp(url string) (driver.StateServiceProvider, error) {
	p.sspsMu.Lock()
	defer p.sspsMu.Unlock()

	ssp, ok := p.ssps[url]
	if !ok {
		u, err := url2.Parse(url)
		if err != nil {
			return nil, errors.Wrapf(err, "failed parsing url")
		}
		provider, ok := p.sspDrivers[driver.SSPDriverName(u.Scheme)]
		if !ok {
			return nil, errors.Errorf("invalid scheme, expected fabric, got [%s]", u.Scheme)
		}
		ssp, err = provider.Driver.New()
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting state service provider for [%s]", u.Scheme)
		}
		p.ssps[url] = ssp
	}
	return ssp, nil
}

// GetServiceProvider returns an instance of a state service provider
func GetServiceProvider(sp token.ServiceProvider) (*ServiceProvider, error) {
	s, err := sp.GetService(&ServiceProvider{})
	if err != nil {
		return nil, errors.Wrap(err, "failed getting state service provider")
	}
	return s.(*ServiceProvider), nil
}
