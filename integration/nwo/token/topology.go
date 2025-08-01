/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	nodepkg "github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/crypto/fabtokenv1"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/crypto/zkatdlognoghv1"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/onsi/gomega"
)

const (
	TopologyName = "token"
)

var (
	Drivers = []string{zkatdlognoghv1.DriverIdentifier, fabtokenv1.DriverIdentifier}
)

type BackedTopology interface {
	Name() string
}

type Topology struct {
	TopologyName string `yaml:"name,omitempty"`
	TopologyType string `yaml:"type,omitempty"`

	TokenSelector string
	FinalityType  config.ManagerType
	TMSs          []*topology.TMS
}

func NewTopology() *Topology {
	return &Topology{
		TopologyName:  TopologyName,
		TopologyType:  TopologyName,
		TokenSelector: "",
		FinalityType:  "delivery",
		TMSs:          []*topology.TMS{},
	}
}

func (t *Topology) Name() string {
	return t.TopologyName
}

func (t *Topology) Type() string {
	return t.TopologyType
}

func (t *Topology) DefaultChannel() string {
	return ""
}

func (t *Topology) AddTMS(fscNodes []*node.Node, backend BackedTopology, channel string, driver string) *topology.TMS {
	found := false
	for _, s := range Drivers {
		if driver == s {
			found = true
			break
		}
	}
	if !found {
		gomega.Expect(found).To(gomega.BeTrue(), "Driver [%s] not recognized", driver)
	}

	var nodes []*node.Node
	nodes = append(nodes, fscNodes...)
	tms := &topology.TMS{
		BackendTopology: backend,
		Network:         backend.Name(),
		Channel:         channel,
		Namespace:       ttx.TokenNamespace,
		Driver:          driver,
		Certifiers:      []string{},
		BackendParams:   map[string]interface{}{},
		TokenTopology:   t,
		FSCNodes:        nodes,
	}
	t.TMSs = append(t.TMSs, tms)
	return tms
}

func (t *Topology) SetSDK(fscTopology *fsc.Topology, sdk nodepkg.SDK) {
	for _, node := range fscTopology.Nodes {
		node.AddSDK(sdk)
	}
}

func (t *Topology) GetTMSs() []*topology.TMS {
	return t.TMSs
}
