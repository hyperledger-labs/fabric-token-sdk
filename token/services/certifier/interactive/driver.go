/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interactive

import (
	"context"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc"
)

const (
	ConfigurationKey = "certification.interactive"
)

type Certification struct {
	IDs []string `yaml:"ids,omitempty"`
}

type BackendFactory func(tms *token.ManagementService, wallet string) (Backend, error)

type Resolver interface {
	ResolveIdentities(endpoints ...string) ([]view.Identity, error)
}

type Subscriber = events.Subscriber

type Driver struct {
	BackendFactory    BackendFactory
	Resolver          Resolver
	Subscriber        Subscriber
	ViewManager       ViewManager
	ResponderRegistry ResponderRegistry
	MetricsProvider   metrics.Provider

	Sync                 sync.Mutex
	CertificationClients map[string]*CertificationClient
	CertificationService *CertificationService
}

func NewDriver(backendFactory BackendFactory, resolver Resolver, subscriber Subscriber, viewManager ViewManager, responderRegistry ResponderRegistry, metricsProvider metrics.Provider) *Driver {
	return &Driver{
		BackendFactory:       backendFactory,
		Resolver:             resolver,
		Subscriber:           subscriber,
		ViewManager:          viewManager,
		ResponderRegistry:    responderRegistry,
		MetricsProvider:      metricsProvider,
		CertificationClients: map[string]*CertificationClient{},
	}
}

func (d *Driver) NewCertificationClient(ctx context.Context, tms *token.ManagementService) (driver.CertificationClient, error) {
	d.Sync.Lock()
	defer d.Sync.Unlock()

	k := tms.Channel() + ":" + tms.Namespace()
	cm, ok := d.CertificationClients[k]
	if !ok {
		certification := &Certification{}
		if err := tms.Configuration().UnmarshalKey(ConfigurationKey, &certification); err != nil {
			return nil, errors.Wrap(err, "failed unmarshalling certification config")
		}

		certifiers, err := d.Resolver.ResolveIdentities(certification.IDs...)
		if err != nil {
			return nil, errors.WithMessagef(err, "cannot resolve certifier identities")
		}
		if len(certifiers) == 0 {
			return nil, errors.Errorf("no certifier id configured")
		}

		var certificationClient = NewCertificationClient(
			ctx,
			tms.Channel(),
			tms.Namespace(),
			tms.Vault().NewQueryEngine(),
			tms.Vault().CertificationStorage(),
			d.ViewManager,
			certifiers,
			d.Subscriber,
			3,
			10*time.Second,
		)
		if err := certificationClient.Scan(); err != nil {
			logger.Warnf("failed to scan the vault for tokens to be certified [%s]", err)
		}
		certificationClient.Start()

		d.CertificationClients[k] = certificationClient
		cm = certificationClient
	}
	return cm, nil
}

func (d *Driver) NewCertificationService(tms *token.ManagementService, wallet string) (driver.CertificationService, error) {
	d.Sync.Lock()
	defer d.Sync.Unlock()

	if d.CertificationService == nil {
		backend, err := d.BackendFactory(tms, wallet)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to create backend")
		}
		d.CertificationService = NewCertificationService(d.ResponderRegistry, d.MetricsProvider, backend)
	}
	d.CertificationService.SetWallet(tms, wallet)

	return d.CertificationService, nil
}

type ChaincodeBackend struct{}

func (c *ChaincodeBackend) Load(context view.Context, cr *CertificationRequest) ([][]byte, error) {
	logger.Debugf("invoke chaincode to get commitments for [%v]", cr.IDs)
	// TODO: if the certifier fetches all token transactions, it might have the tokens in its on vault.
	tokensBoxed, err := context.RunView(tcc.NewGetTokensView(cr.Channel, cr.Namespace, cr.IDs...))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting tokens [%s:%s][%v]", cr.Channel, cr.Namespace, cr.IDs)
	}

	tokens, ok := tokensBoxed.([][]byte)
	if !ok {
		return nil, errors.Errorf("expected [][]byte, got [%T]", tokens)
	}
	return tokens, nil
}
