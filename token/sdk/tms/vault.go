/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tms

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/mailman"
	"github.com/pkg/errors"
)

type NetworkProvider interface {
	GetNetwork(network string, channel string) (*network.Network, error)
}

type VaultProvider struct {
	np NetworkProvider
}

func NewVaultProvider(np NetworkProvider) *VaultProvider {
	return &VaultProvider{np: np}
}

func (v *VaultProvider) Vault(tms *token.ManagementService) (mailman.Vault, mailman.QueryService, error) {
	net, err := v.np.GetNetwork(tms.Network(), tms.Channel())
	if err != nil {
		return nil, nil, errors.Errorf("cannot get network for TMS [%s]", tms.ID())
	}
	vault, err := net.Vault(tms.Namespace())
	if err != nil {
		return nil, nil, errors.Errorf("cannot get network vault for TMS [%s]", tms.ID())
	}
	return vault, tms.Vault().NewQueryEngine(), nil
}
