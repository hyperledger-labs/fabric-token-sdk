/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
)

type NetworkProvider struct {
	sp view.ServiceProvider
}

func NewNetworkProvider(sp view.ServiceProvider) *NetworkProvider {
	return &NetworkProvider{sp: sp}
}

func (n *NetworkProvider) Network(networkID string) (core.Network, error) {
	fns := fabric.GetFabricNetworkService(n.sp, networkID)
	if fns == nil {
		return nil, errors.Errorf("network [%s] not found", networkID)
	}
	return fns, nil
}
