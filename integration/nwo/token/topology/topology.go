/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package topology

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
)

type Chaincode struct {
	Orgs        []string
	Private     bool
	DockerImage string
}

type TMS struct {
	Network             string
	Channel             string
	Namespace           string
	Driver              string
	PublicParamsGenArgs []string
	TokenChaincode      Chaincode
	Certifiers          []string

	TargetNetworkTopology *topology.Topology `yaml:"-"`
}

func (t *TMS) AddCertifier(certifier *node.Node) *TMS {
	t.Certifiers = append(t.Certifiers, certifier.Name)
	return t
}

func (t *TMS) SetNamespace(orgs []string, publicParamsGenArgs ...string) {
	t.TokenChaincode.Orgs = orgs
	t.PublicParamsGenArgs = publicParamsGenArgs
}

func (t *TMS) Private(dockerImage string) {
	t.TargetNetworkTopology.EnableFPC()
	t.TargetNetworkTopology.AddChaincode(&topology.ChannelChaincode{
		Chaincode: topology.Chaincode{
			Name: t.Namespace,
		},
		PrivateChaincode: topology.PrivateChaincode{
			Image: "",
		},
		Channel: t.Channel,
		Private: true,
	})

	t.TokenChaincode.Private = true
	t.TokenChaincode.DockerImage = dockerImage
}

func (t *TMS) ID() string {
	return fmt.Sprintf("%s-%s-%s", t.Network, t.Channel, t.Network)
}
