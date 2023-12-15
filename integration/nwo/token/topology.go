/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	. "github.com/onsi/gomega"
)

const (
	TopologyName = "token"
)

var (
	Drivers = []string{"dlog", "fabtoken"}
)

type BackedTopology interface {
	Name() string
}

type Topology struct {
	TopologyName string `yaml:"name,omitempty"`
	TopologyType string `yaml:"type,omitempty"`

	TMSs     []*topology.TMS
	SqlTTXDB bool
}

func NewTopology() *Topology {
	return &Topology{
		TopologyName: TopologyName,
		TopologyType: TopologyName,
		TMSs:         []*topology.TMS{},
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
		Expect(found).To(BeTrue(), "Driver [%s] not recognized", driver)
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

func (t *Topology) SetSDK(fscTopology *fsc.Topology, sdk api.SDK) {
	for _, node := range fscTopology.Nodes {
		node.AddSDK(sdk)
	}
}

func (t *Topology) GetTMSs() []*topology.TMS {
	return t.TMSs
}

func (t *Topology) EnableSqlTTXDB() {
	t.SqlTTXDB = true
}
