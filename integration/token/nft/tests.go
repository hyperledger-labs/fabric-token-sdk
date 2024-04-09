/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nft

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/nft/views"
	. "github.com/onsi/gomega"
)

func NewTestSuiteLibP2P(backend string, startPort func() int, tokenSDKDriver string) *token.TestSuite {
	return token.NewTestSuite(nil, startPort, Topology(backend, fsc.LibP2P, tokenSDKDriver, integration.NoReplication))
}

func NewTestSuiteWebsocket(backend string, startPort func() int, tokenSDKDriver string, replicationOpts *integration.ReplicationOptions) *token.TestSuite {
	return token.NewTestSuite(replicationOpts.SQLConfigs, startPort, Topology(backend, fsc.WebSocket, tokenSDKDriver, &token.ReplicationOptions{ReplicationOptions: replicationOpts}))
}

func TestAll(network *integration.Infrastructure) {
	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	registerAuditor(network)
	houseID := issueHouse(network, "issuer", "alice", 4)
	queryHouse(network, "alice", houseID, "5th Avenue")
	queryHouse(network, "bob", houseID, "5th Avenue", "failed loading house with id")
	transferHouse(network, houseID, "alice", "bob", "bob")
	queryHouse(network, "bob", houseID, "5th Avenue")
	queryHouse(network, "alice", houseID, "5th Avenue", "failed loading house with id")
}

func TestAllWithReplicas(network *integration.Infrastructure) {
	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	registerAuditor(network)
	houseID := issueHouse(network, "issuer", "alice", 4)
	queryHouse(network, "alice", houseID, "5th Avenue")
	queryHouse(network, "fsc.alice.0", houseID, "5th Avenue")
	queryHouse(network, "fsc.alice.1", houseID, "5th Avenue")
	queryHouse(network, "fsc.alice.2", houseID, "5th Avenue")
	queryHouse(network, "bob", houseID, "5th Avenue", "failed loading house with id")
	queryHouse(network, "fsc.bob.0", houseID, "5th Avenue", "failed loading house with id")
	queryHouse(network, "fsc.bob.1", houseID, "5th Avenue", "failed loading house with id")
	transferHouse(network, houseID, "fsc.alice.1", "bob", "fsc.bob.1")
	queryHouse(network, "bob", houseID, "5th Avenue")
	queryHouse(network, "fsc.bob.0", houseID, "5th Avenue")
	queryHouse(network, "fsc.bob.1", houseID, "5th Avenue")
	queryHouse(network, "alice", houseID, "5th Avenue", "failed loading house with id")
	queryHouse(network, "fsc.alice.0", houseID, "5th Avenue", "failed loading house with id")
	queryHouse(network, "fsc.alice.1", houseID, "5th Avenue", "failed loading house with id")
	queryHouse(network, "fsc.alice.2", houseID, "5th Avenue", "failed loading house with id")
}

func registerAuditor(network *integration.Infrastructure) {
	_, err := network.Client("auditor").CallView("registerAuditor", nil)
	Expect(err).NotTo(HaveOccurred())
}

func issueHouse(network *integration.Infrastructure, issuer, recipient string, valuation uint64) string {
	houseIDBoxed, err := network.Client(issuer).CallView("issue", common.JSONMarshall(views.IssueHouse{
		IssuerWallet: "",
		Recipient:    recipient,
		Address:      "5th Avenue",
		Valuation:    valuation,
	}))
	Expect(err).NotTo(HaveOccurred())
	time.Sleep(5 * time.Second)
	return common.JSONUnmarshalString(houseIDBoxed)
}

func transferHouse(network *integration.Infrastructure, houseID string, from, toID, to string) {
	txIDBoxed, err := network.Client(from).CallView("transfer", common.JSONMarshall(views.Transfer{
		HouseID:   houseID,
		Recipient: toID,
	}))
	Expect(err).NotTo(HaveOccurred())
	common2.CheckFinality(network, to, common.JSONUnmarshalString(txIDBoxed), nil, false)
}

func queryHouse(network *integration.Infrastructure, clientID string, houseID string, address string, errorMsgs ...string) {
	resBoxed, err := network.Client(clientID).CallView("queryHouse", common.JSONMarshall(views.GetHouse{
		HouseID: houseID,
	}))
	if len(errorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		h := &views.House{}
		Expect(json.Unmarshal(resBoxed.([]byte), h)).NotTo(HaveOccurred())
		Expect(h.LinearID).To(BeEquivalentTo(houseID))
		Expect(h.Address).To(BeEquivalentTo(address))
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range errorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg))
		}
	}
}
