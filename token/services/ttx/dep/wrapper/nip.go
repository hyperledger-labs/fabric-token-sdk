/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wrapper

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/id"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/sig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type NetworkIdentityProvider struct {
	ip         *id.Provider
	sigService *sig.Service
}

func NewNetworkIdentityProvider(ip *id.Provider, sigService *sig.Service) *NetworkIdentityProvider {
	return &NetworkIdentityProvider{ip: ip, sigService: sigService}
}

func (n *NetworkIdentityProvider) DefaultIdentity() view.Identity {
	return n.ip.DefaultIdentity()
}

func (n *NetworkIdentityProvider) GetSigner(identity view.Identity) (driver.Signer, error) {
	return n.sigService.GetSigner(identity)
}
