/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interactive

import (
	"context"
	"sync"
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc"
	"github.com/pkg/errors"
)

type Driver struct {
	sync      sync.Mutex
	cms       map[string]*CertificationClient
	certifier *CertificationService
}

func NewDriver() *Driver {
	return &Driver{
		sync: sync.Mutex{},
		cms:  map[string]*CertificationClient{},
	}
}

func (d *Driver) NewCertificationClient(sp view2.ServiceProvider, networkID, channel, namespace string) (driver.CertificationClient, error) {
	d.sync.Lock()
	defer d.sync.Unlock()

	k := channel + ":" + namespace
	cm, ok := d.cms[k]
	if !ok {
		n := network.GetInstance(sp, networkID, channel)
		if n == nil {
			return nil, errors.Errorf("network [%s] not found", networkID)
		}
		v, err := n.Vault(namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get vault for network [%s:%s]", networkID, channel)
		}

		// Load certifier identities
		tms := token.GetManagementService(sp, token.WithTMS(networkID, channel, namespace))
		if tms == nil {
			return nil, errors.Errorf("failed to get token management service for network [%s:%s:%s]", networkID, channel, namespace)
		}
		var certifiers []view.Identity
		certifiers, err = view2.GetEndpointService(sp).ResolveIdentities(tms.ConfigManager().Certifiers()...)
		if err != nil {
			return nil, errors.WithMessagef(err, "cannot resolve certifier identities")
		}
		if len(certifiers) == 0 {
			return nil, errors.Errorf("no certifier id configured")
		}

		sub, err := events.GetSubscriber(sp)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to get global subscriber")
		}

		certificationClient := NewCertificationClient(
			context.Background(),
			channel,
			namespace,
			v,
			v,
			view2.GetManager(sp),
			certifiers,
			sub,
			3,
			10*time.Second,
		)
		if err := certificationClient.Scan(); err != nil {
			logger.Warnf("failed to scan the vault for tokens to be certified [%s]", err)
		}
		certificationClient.Start()

		d.cms[k] = certificationClient
		cm = certificationClient
	}
	return cm, nil
}

func (d *Driver) NewCertificationService(sp view2.ServiceProvider, network, channel, namespace, wallet string) (driver.CertificationService, error) {
	d.sync.Lock()
	defer d.sync.Unlock()

	if d.certifier == nil {
		d.certifier = NewCertificationService(sp, &ChaincodeBackend{})
	}
	d.certifier.SetWallet(network, channel, namespace, wallet)

	return d.certifier, nil
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
