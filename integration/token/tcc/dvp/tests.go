/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dvp

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/tcc/dvp/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/tcc/dvp/views/cash"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/tcc/dvp/views/house"
	. "github.com/onsi/gomega"
	"time"
)

func TestAll(network *integration.Infrastructure) {
	registerAuditor(network)

	_, err := network.Client("cash_issuer").CallView("issue_cash", common.JSONMarshall(cash.IssueCash{
		IssuerWallet: "",
		TokenType:    "USD",
		Quantity:     10,
		Recipient:    "buyer",
	}))
	Expect(err).NotTo(HaveOccurred())
	time.Sleep(5 * time.Second)

	houseIDBoxed, err := network.Client("house_issuer").CallView("issue_house", common.JSONMarshall(house.IssueHouse{
		IssuerWallet: "",
		Recipient:    "seller",
		Address:      "5th Avenue",
		Valuation:    5,
	}))
	Expect(err).NotTo(HaveOccurred())
	time.Sleep(5 * time.Second)

	_, err = network.Client("seller").CallView("sell", common.JSONMarshall(views2.Sell{
		HouseID: common.JSONUnmarshalString(houseIDBoxed),
		Buyer:   "buyer",
	}))
	Expect(err).NotTo(HaveOccurred())
}

func registerAuditor(network *integration.Infrastructure) {
	_, err := network.Client("auditor").CallView("register", nil)
	Expect(err).NotTo(HaveOccurred())
}
