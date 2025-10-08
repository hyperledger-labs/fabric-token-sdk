/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlogx

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views/fabricx/pp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type LoadPublicParamsInputs struct {
	FSCNodeIDs           []string
	LoadPublicParameters pp.LoadPublicParameters
}

func ReloadPublicParametersForNode(ii *integration.Infrastructure, pps []LoadPublicParamsInputs, id string) error {
	for _, tms := range pps {
		if _, err := ii.Client(id).CallView("LoadPublicParameters", common.JSONMarshall(tms.LoadPublicParameters)); err != nil {
			return err
		}
	}
	return nil
}

func LoadPublicParamsForAll(ii *integration.Infrastructure, pps []LoadPublicParamsInputs) error {
	for _, tms := range pps {
		for _, nodeID := range tms.FSCNodeIDs {
			if _, err := ii.Client(nodeID).CallView("LoadPublicParameters", common.JSONMarshall(tms.LoadPublicParameters)); err != nil {
				return err
			}
		}
	}
	return nil
}

func GetPublicParamsInputs(ii *integration.Infrastructure) ([]LoadPublicParamsInputs, error) {
	tp := find[api.Platform, *token.Platform](ii.NWO.Platforms)
	if tp == nil {
		return nil, errors.New("tp was nil")
	}
	ttp := find[api.Topology, *token.Topology](ii.Topologies)
	if ttp == nil {
		return nil, errors.New("ttp was nil")
	}
	if len(ttp.TMSs) == 0 {
		return nil, errors.New("no tmss found")
	}

	pps := make([]LoadPublicParamsInputs, len(ttp.TMSs))
	for i, tms := range ttp.TMSs {
		nodeIDs := make([]string, len(tms.FSCNodes))
		for j, n := range tms.FSCNodes {
			nodeIDs[j] = n.Name
		}
		pps[i] = LoadPublicParamsInputs{
			FSCNodeIDs: nodeIDs,
			LoadPublicParameters: pp.LoadPublicParameters{
				TMSID: driver.TMSID{Network: tms.Network, Channel: tms.Channel, Namespace: tms.Namespace},
				Raw:   tp.PublicParameters(tms),
			},
		}
	}
	return pps, nil
}

func find[L any, K any](items []L) K {
	for _, item := range items {
		if typed, ok := any(item).(K); ok {
			return typed
		}
	}
	return utils.Zero[K]()
}
