/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabricx

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	common2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	tokentopology "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views/fabricx/tmsdeploy"
)

type ClientProvider interface {
	Client(string) api.GRPCClient
}

type Backend struct {
	ClientProvider ClientProvider
}

func (b *Backend) PrepareNamespace(t *tokentopology.TMS) {
	switch n := t.BackendTopology.(type) {
	case *topology.Topology:
		n.AddNamespaceWithUnanimity(t.Namespace)
	default:
		panic(fmt.Sprintf("unknown backend network type %T", n))
	}
}

func (b *Backend) UpdatePublicParams(tms *tokentopology.TMS, ppRaw []byte) {
	endorsers := fabric.Endorsers(tms)
	if len(endorsers) == 0 {
		panic("no endorsers found")
	}
	_, err := b.ClientProvider.Client(endorsers[0]).CallView("TMSDeploy", common2.JSONMarshall(
		&tmsdeploy.Deploy{
			Network:         tms.Network,
			Channel:         tms.Channel,
			Namespace:       tms.Namespace,
			PublicParamsRaw: ppRaw,
		},
	))
	if err != nil {
		panic("failed updating pps: " + err.Error())
	}
}
