/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/weaver"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/state/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
	"go.uber.org/dig"
)

const (
	QueryPublicParamsFunction = "queryPublicParams"

	DefaultFabricNetworkName = ""
)

var logger = logging.MustGetLogger("token-sdk.state")

type RelayProvider interface {
	Relay(fns *fabric.NetworkService) *weaver.Relay
}

type StateServiceProvider struct {
	drivers       map[StateDriverName]NamedStateDriver
	fnsProvider   *fabric.NetworkServiceProvider
	relayProvider RelayProvider
	tmsProvider   *token.ManagementServiceProvider

	mu             sync.RWMutex
	queryExecutors map[string]driver.StateQueryExecutor
	verifiers      map[string]driver.StateVerifier
}

func NewStateServiceProvider(
	drivers map[StateDriverName]NamedStateDriver,
	fnsProvider *fabric.NetworkServiceProvider,
	weaverProvider RelayProvider,
	tmsProvider *token.ManagementServiceProvider,
) *StateServiceProvider {
	return &StateServiceProvider{
		drivers:        drivers,
		fnsProvider:    fnsProvider,
		relayProvider:  weaverProvider,
		tmsProvider:    tmsProvider,
		mu:             sync.RWMutex{},
		queryExecutors: map[string]driver.StateQueryExecutor{},
		verifiers:      map[string]driver.StateVerifier{},
	}
}

func (f *StateServiceProvider) RegisterDriver(driver NamedStateDriver) {
	f.drivers[driver.Name] = driver
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
	pp, err := f.tmsProvider.PublicParametersFromBytes(ppRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling public parameters from [%s]", url)
	}

	driver, ok := f.drivers[StateDriverName(pp.Identifier())]
	if !ok {
		return nil, errors.Errorf("invalid public parameters type, got [%s]", pp.Identifier())
	}
	qe, err = driver.Driver.NewStateQueryExecutor(url)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating state query executor from [%s]", url)
	}
	v, err := driver.Driver.NewStateVerifier(url)
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
	tmsID, err := FabricURLToTMSID(url)
	if err != nil {
		return nil, errors.Wrapf(err, "failed parsing url [%s]", url)
	}
	tms, err := f.tmsProvider.GetManagementService(token.WithTMSID(tmsID))
	if err != nil {
		// If not, fetch public parameters, if not fetched already
		ppRaw, err := f.fetchPublicParameters(url)
		if err != nil {
			return nil, errors.Wrapf(err, "failed fetching public parameters from [%s]", url)
		}
		pp, err := f.tmsProvider.PublicParametersFromBytes(ppRaw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed unmarshalling public parameters from [%s]", url)
		}
		identifier = pp.Identifier()
	} else {
		identifier = tms.PublicParametersManager().PublicParameters().Identifier()
	}

	driver, ok := f.drivers[StateDriverName(identifier)]
	if !ok {
		return nil, errors.Errorf("invalid public parameters type, got [%s]", identifier)
	}
	v, err = driver.Driver.NewStateVerifier(url)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating state verifier from [%s]", url)
	}
	f.verifiers[url] = v

	return v, nil
}

func (f *StateServiceProvider) URLToTMSID(url string) (token.TMSID, error) {
	return FabricURLToTMSID(url)
}

func (f *StateServiceProvider) fetchPublicParameters(url string) ([]byte, error) {
	fns, err := f.fnsProvider.FabricNetworkService(DefaultFabricNetworkName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting default FNS for [%s]", url)
	}
	relay := f.relayProvider.Relay(fns)
	logger.Debugf("query [%s] for the public parameters", url)

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
	stateDrivers   map[StateDriverName]NamedStateDriver
	fnsProvider    *fabric.NetworkServiceProvider
	weaverProvider RelayProvider
	tmsProvider    *token.ManagementServiceProvider
}

func NewSSPDriver(in struct {
	dig.In
	StateDriversList []NamedStateDriver `group:"fabric-ssp-state-drivers"`
	FNSProvider      *fabric.NetworkServiceProvider
	WeaverProvider   RelayProvider
	TMSProvider      *token.ManagementServiceProvider
}) driver.NamedSSPDriver {
	stateDrivers := map[StateDriverName]NamedStateDriver{}
	for _, stateDriver := range in.StateDriversList {
		stateDrivers[stateDriver.Name] = stateDriver
	}
	return driver.NamedSSPDriver{
		Name: "fabric",
		Driver: &SSPDriver{
			stateDrivers:   stateDrivers,
			fnsProvider:    in.FNSProvider,
			weaverProvider: in.WeaverProvider,
			tmsProvider:    in.TMSProvider,
		},
	}
}

func (f *SSPDriver) New() (driver.StateServiceProvider, error) {
	return NewStateServiceProvider(f.stateDrivers, f.fnsProvider, f.weaverProvider, f.tmsProvider), nil
}
