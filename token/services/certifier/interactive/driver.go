/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package interactive

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
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

func (d *Driver) NewCertificationClient(sp view2.ServiceProvider, network, channel, namespace string) (driver.CertificationClient, error) {
	d.sync.Lock()
	defer d.sync.Unlock()

	k := channel + ":" + namespace
	cm, ok := d.cms[k]
	if !ok {
		ch := fabric.GetChannel(sp, network, channel)
		fabricVault := ch.Vault()
		tokenVault := vault.NewVault(sp, ch, namespace)

		// Load certifier identities
		var tmsConfigs []*token.TMS
		if err := view2.GetConfigService(sp).UnmarshalKey("token.tms", &tmsConfigs); err != nil {
			return nil, errors.WithMessagef(err, "cannot load token-sdk configuration")
		}
		var certifiers []view.Identity
		for _, tms := range tmsConfigs {
			if tms.Channel == channel && tms.Namespace == namespace {
				var err error
				certifiers, err = view2.GetEndpointService(sp).ResolveIdentities(tms.Certification.Interactive.IDs...)
				if err != nil {
					return nil, errors.WithMessagef(err, "cannot resolve certifier identities")
				}
				break
			}
		}
		if len(certifiers) == 0 {
			return nil, errors.Errorf("no certifier id configured")
		}

		inst := NewCertificationClient(
			context.Background(),
			channel,
			namespace,
			fabricVault,
			tokenVault.QueryEngine(),
			tokenVault.CertificationStorage(),
			view2.GetManager(sp),
			certifiers,
		)
		inst.Start()

		d.cms[k] = inst
		cm = inst
	}
	return cm, nil
}

func (d *Driver) NewCertificationService(sp view2.ServiceProvider, network, channel, namespace, wallet string) (driver.CertificationService, error) {
	d.sync.Lock()
	defer d.sync.Unlock()

	if d.certifier == nil {
		d.certifier = NewCertificationService(sp)
	}
	d.certifier.SetWallet(network, channel, namespace, wallet)

	return d.certifier, nil
}
