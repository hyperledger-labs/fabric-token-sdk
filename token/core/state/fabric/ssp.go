/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	weaver2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/weaver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/pkg/errors"
)

const (
	QueryPublicParamsFunction = "queryPublicParams"
)

var logger = flogging.MustGetLogger("token-sdk.state")

type StateServiceProvider struct {
	sp             driver.ServiceProvider
	mu             sync.RWMutex
	queryExecutors map[string]driver.StateQueryExecutor
	verifiers      map[string]driver.StateVerifier
}

func NewStateServiceProvider(sp driver.ServiceProvider) *StateServiceProvider {
	return &StateServiceProvider{
		sp:             sp,
		mu:             sync.RWMutex{},
		queryExecutors: map[string]driver.StateQueryExecutor{},
		verifiers:      map[string]driver.StateVerifier{},
	}
}

func (f *StateServiceProvider) QueryExecutor(url string) (driver.StateQueryExecutor, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	qe, ok := f.queryExecutors[url]
	if ok {
		return qe, nil
	}

	// Fetch public parameters, if not fetched already
	ppRaw, err := f.fetchPublicParameters(url)
	if err != nil {
		return nil, errors.Wrapf(err, "failed fetching public parameters from [%s]", url)
	}
	pp, err := core.PublicParametersFromBytes(ppRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling public parameters from [%s]", url)
	}

	driver, ok := drivers[pp.Identifier()]
	if !ok {
		return nil, errors.Errorf("invalid public parameters type, got [%s]", pp.Identifier())
	}
	qe, err = driver.NewStateQueryExecutor(f.sp, url)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating state query executor from [%s]", url)
	}
	v, err := driver.NewStateVerifier(f.sp, url)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating state verifier from [%s]", url)
	}
	f.queryExecutors[url] = qe
	f.verifiers[url] = v

	return qe, nil
}

func (f *StateServiceProvider) Verifier(url string) (driver.StateVerifier, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	v, ok := f.verifiers[url]
	if ok {
		return v, nil
	}

	var identifier string

	// Check if the url refers to a TMS known by this node, then create and return just a verifier
	tmsID, err := pledge.FabricURLToTMSID(url)
	if err != nil {
		return nil, errors.Wrapf(err, "failed parsing url [%s]", url)
	}
	tms := token.GetManagementService(f.sp, token.WithTMSID(tmsID))
	if tms != nil {
		identifier = tms.PublicParametersManager().PublicParameters().Identifier()
	} else {
		// If not, fetch public parameters, if not fetched already
		ppRaw, err := f.fetchPublicParameters(url)
		if err != nil {
			return nil, errors.Wrapf(err, "failed fetching public parameters from [%s]", url)
		}
		pp, err := core.PublicParametersFromBytes(ppRaw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed unmarshalling public parameters from [%s]", url)
		}
		identifier = pp.Identifier()
	}

	driver, ok := drivers[identifier]
	if !ok {
		return nil, errors.Errorf("invalid public parameters type, got [%s]", identifier)
	}
	v, err = driver.NewStateVerifier(f.sp, url)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating state verifier from [%s]", url)
	}
	f.verifiers[url] = v

	return v, nil
}

func (f *StateServiceProvider) URLToTMSID(url string) (driver.TMSID, error) {
	//TODO implement me
	panic("implement me")
}

func (f *StateServiceProvider) fetchPublicParameters(url string) ([]byte, error) {
	relay := weaver2.GetProvider(f.sp).Relay(fabric.GetDefaultFNS(f.sp))
	logger.Debugf("Query [%s] for the public parameters", url)

	query, err := relay.ToFabric().Query(url, QueryPublicParamsFunction)
	if err != nil {
		return nil, err
	}
	res, err := query.Call()
	if err != nil {
		return nil, err
	}
	return res.Result(), nil
}

type SSPDriver struct {
	mu             sync.RWMutex
	queryExecutors map[string]driver.StateQueryExecutor
	verifiers      map[string]driver.StateVerifier
}

func NewSSPDriver() *SSPDriver {
	return &SSPDriver{
		mu:             sync.RWMutex{},
		queryExecutors: map[string]driver.StateQueryExecutor{},
		verifiers:      map[string]driver.StateVerifier{},
	}
}

func (f *SSPDriver) New(sp driver.ServiceProvider) (driver.StateServiceProvider, error) {
	return NewStateServiceProvider(sp), nil
}

func init() {
	core.RegisterSSPDriver("fabric", NewSSPDriver())
}
