/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabricx

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"
	tokentopology "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
)

type Backend struct {
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
}
