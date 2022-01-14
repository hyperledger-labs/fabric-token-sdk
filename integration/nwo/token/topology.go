/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"
	fsc2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	. "github.com/onsi/gomega"

	topology2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	token "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
)

const (
	TopologyName = "token"
)

var (
	Drivers = []string{"dlog", "fabtoken"}
)

type Topology struct {
	TopologyName string `yaml:"name,omitempty"`
	TopologyType string `yaml:"type,omitempty"`

	TMSs []*topology2.TMS
}

func NewTopology() *Topology {
	return &Topology{
		TopologyName: TopologyName,
		TopologyType: TopologyName,
		TMSs:         []*topology2.TMS{},
	}
}

func (t *Topology) Name() string {
	return t.TopologyName
}

func (t *Topology) Type() string {
	return t.TopologyType
}

func (t *Topology) AddTMS(fabric *topology.Topology, driver string) *topology2.TMS {
	found := false
	for _, s := range Drivers {
		if driver == s {
			found = true
			break
		}
	}
	if !found {
		Expect(found).To(BeTrue(), "Driver [%s] not recognized", driver)
	}

	tms := &topology2.TMS{
		TargetNetworkTopology: fabric,
		Network:               fabric.Name(),
		Channel:               fabric.Channels[0].Name,
		Namespace:             "zkat",
		Driver:                driver,
		Certifiers:            []string{},
		TokenChaincode:        topology2.Chaincode{},
	}
	t.TMSs = append(t.TMSs, tms)
	return tms
}

func (t *Topology) SetDefaultSDK(fscTopology *fsc2.Topology) {
	t.SetSDK(fscTopology, &token.SDK{})
}

func (t *Topology) SetSDK(fscTopology *fsc2.Topology, sdk api.SDK) {
	for _, node := range fscTopology.Nodes {
		node.AddSDK(sdk)
	}
}
