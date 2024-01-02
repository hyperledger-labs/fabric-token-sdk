/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dvp

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp/views/cash"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp/views/house"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/gomega"
)

func TestAll(network *integration.Infrastructure) {
	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	// Ready to go
	registerAuditor(network)
	issueCash(network)
	checkBalance(network, "buyer", "", "USD", 10)
	checkBalance(network, "seller", "", "USD", 0)
	houseID := issueHouse(network, 4)
	queryHouse(network, "seller", houseID, "5th Avenue")
	queryHouse(network, "buyer", houseID, "5th Avenue", "failed loading house with id")
	sellHouse(network, houseID)
	queryHouse(network, "buyer", houseID, "5th Avenue")
	queryHouse(network, "seller", houseID, "5th Avenue", "failed loading house with id")
	checkBalance(network, "buyer", "", "USD", 6)
	checkBalance(network, "seller", "", "USD", 4)
}

func registerAuditor(network *integration.Infrastructure) {
	_, err := network.Client("auditor").CallView("registerAuditor", nil)
	Expect(err).NotTo(HaveOccurred())
}

func issueCash(network *integration.Infrastructure) {
	_, err := network.Client("cash_issuer").CallView("issue_cash", common.JSONMarshall(cash.IssueCash{
		IssuerWallet: "",
		TokenType:    "USD",
		Quantity:     10,
		Recipient:    "buyer",
	}))
	Expect(err).NotTo(HaveOccurred())
	time.Sleep(5 * time.Second)
}

func issueHouse(network *integration.Infrastructure, valuation uint64) string {
	houseIDBoxed, err := network.Client("house_issuer").CallView("issue_house", common.JSONMarshall(house.IssueHouse{
		IssuerWallet: "",
		Recipient:    "seller",
		Address:      "5th Avenue",
		Valuation:    valuation,
	}))
	Expect(err).NotTo(HaveOccurred())
	time.Sleep(5 * time.Second)
	return common.JSONUnmarshalString(houseIDBoxed)
}

func sellHouse(network *integration.Infrastructure, houseID string) {
	_, err := network.Client("seller").CallView("sell", common.JSONMarshall(views2.Sell{
		HouseID: houseID,
		Buyer:   "buyer",
	}))
	Expect(err).NotTo(HaveOccurred())
}

func checkBalance(network *integration.Infrastructure, id string, wallet string, typ string, expected uint64) {
	res, err := network.Client(id).CallView("balance", common.JSONMarshall(&views2.BalanceQuery{
		Wallet: wallet,
		Type:   typ,
	}))
	Expect(err).NotTo(HaveOccurred())
	b := &views2.Balance{}
	common.JSONUnmarshal(res.([]byte), b)
	Expect(b.Type).To(BeEquivalentTo(typ))
	q, err := token2.ToQuantity(b.Quantity, 64)
	Expect(err).NotTo(HaveOccurred())
	expectedQ := token2.NewQuantityFromUInt64(expected)
	Expect(expectedQ.Cmp(q)).To(BeEquivalentTo(0), "[%s]!=[%s]", expected, q)
}

func queryHouse(network *integration.Infrastructure, clientID string, houseID string, address string, errorMsgs ...string) {
	resBoxed, err := network.Client(clientID).CallView("queryHouse", common.JSONMarshall(house.GetHouse{
		HouseID: houseID,
	}))
	if len(errorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		h := &house.House{}
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
