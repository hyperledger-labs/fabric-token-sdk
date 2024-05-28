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
	token3 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/nft/views"
	. "github.com/onsi/gomega"
)

func TestAll(network *integration.Infrastructure, sel *token3.ReplicaSelector) {
	issuer := sel.Get("issuer")
	alice := sel.Get("alice")
	bob := sel.Get("bob")
	auditor := sel.Get("auditor")

	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	_, err := network.Client(auditor.ReplicaName()).CallView("registerAuditor", nil)
	Expect(err).NotTo(HaveOccurred())
	houseID := issueHouse(network, issuer, alice, 4)
	queryHouse(network, alice, houseID, "5th Avenue")
	queryHouse(network, bob, houseID, "5th Avenue", "failed loading house with id")
	transferHouse(network, houseID, alice, bob)
	queryHouse(network, bob, houseID, "5th Avenue")
	queryHouse(network, alice, houseID, "5th Avenue", "failed loading house with id")
}

func issueHouse(network *integration.Infrastructure, issuer, recipient *token3.NodeReference, valuation uint64) string {
	houseIDBoxed, err := network.Client(issuer.ReplicaName()).CallView("issue", common.JSONMarshall(views.IssueHouse{
		IssuerWallet: "",
		Recipient:    recipient.Id(),
		Address:      "5th Avenue",
		Valuation:    valuation,
	}))
	Expect(err).NotTo(HaveOccurred())
	time.Sleep(5 * time.Second)
	return common.JSONUnmarshalString(houseIDBoxed)
}

func transferHouse(network *integration.Infrastructure, houseID string, from, to *token3.NodeReference) {
	txIDBoxed, err := network.Client(from.ReplicaName()).CallView("transfer", common.JSONMarshall(views.Transfer{
		HouseID:   houseID,
		Recipient: to.Id(),
	}))
	Expect(err).NotTo(HaveOccurred())
	common2.CheckFinality(network, to, common.JSONUnmarshalString(txIDBoxed), nil, false)
}

func queryHouse(network *integration.Infrastructure, client *token3.NodeReference, houseID string, address string, errorMsgs ...string) {
	resBoxed, err := network.Client(client.ReplicaName()).CallView("queryHouse", common.JSONMarshall(views.GetHouse{
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
