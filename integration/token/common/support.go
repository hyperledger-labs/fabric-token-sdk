/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	topology2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/views"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	. "github.com/onsi/gomega"
)

func CheckFinality(network *integration.Infrastructure, id *token2.NodeReference, txID string, tmsID *token.TMSID, fail bool) {
	if id == nil || len(id.Id()) == 0 {
		return
	}
	_, err := network.Client(id.ReplicaName()).CallView("TxFinality", common.JSONMarshall(&views.TxFinality{
		TxID:  txID,
		TMSID: tmsID,
	}))
	if fail {
		Expect(err).To(HaveOccurred())
	} else {
		Expect(err).NotTo(HaveOccurred())
	}
}

func CheckEndorserFinality(network *integration.Infrastructure, id *token2.NodeReference, txID string, tmsID *token.TMSID, fail bool) {
	if id == nil || len(id.Id()) == 0 {
		return
	}
	var nw, channel string
	if tmsID != nil {
		nw, channel = tmsID.Network, tmsID.Channel
	} else {
		t := getFabricTopology(network)
		nw, channel = t.Name(), t.Channels[0].Name
	}
	_, err := network.Client(id.ReplicaName()).CallView("EndorserFinality", common.JSONMarshall(&endorser.Finality{
		TxID:    txID,
		Network: nw,
		Channel: channel,
	}))
	if fail {
		Expect(err).To(HaveOccurred())
	} else {
		Expect(err).NotTo(HaveOccurred())
	}
}

func getFabricTopology(network *integration.Infrastructure) *topology2.Topology {
	for _, t := range network.Topologies {
		if t.Type() == "fabric" {
			return t.(*topology2.Topology)
		}
	}
	panic("no fabric topology found")
}
