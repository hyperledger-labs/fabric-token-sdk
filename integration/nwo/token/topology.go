/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"
	fsc2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	fsc "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	. "github.com/onsi/gomega"

	token "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
)

const (
	TopologyName = "token"
)

var (
	Drivers = []string{"dlog", "fabtoken"}
)

type Chaincode struct {
	Orgs                []string
	PublicParamsGenArgs []string
}

type TMS struct {
	Fabric *topology.Topology `yaml:"-"`

	Network        string
	Channel        string
	Namespace      string
	Driver         string
	TokenChaincode Chaincode
	Certifiers     []string
}

func (t *TMS) AddCertifier(certifier *fsc.Node) *TMS {
	t.Certifiers = append(t.Certifiers, certifier.Name)
	return t
}

func (t *TMS) SetNamespace(orgs []string, publicParamsGenArgs ...string) {
	t.TokenChaincode.Orgs = orgs
	t.TokenChaincode.PublicParamsGenArgs = publicParamsGenArgs
}

type Topology struct {
	TopologyName string `yaml:"name,omitempty"`
	TMSs         []*TMS
}

func NewTopology() *Topology {
	return &Topology{
		TopologyName: TopologyName,
		TMSs:         []*TMS{},
	}
}

func (t *Topology) Name() string {
	return t.TopologyName
}

func (t *Topology) Type() string {
	return t.TopologyName
}

func (t *Topology) AddTMS(fabric *topology.Topology, driver string) *TMS {
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

	tms := &TMS{
		Fabric:         fabric,
		Network:        fabric.TopologyName,
		Channel:        fabric.Channels[0].Name,
		Namespace:      "zkat",
		Driver:         driver,
		Certifiers:     []string{},
		TokenChaincode: Chaincode{},
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
