/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package dvp

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	. "github.com/onsi/gomega"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp/views"
)

func TestAll(network *integration.Infrastructure) {
	registerAuditor(network)

	_, err := network.Client("cash_issuer").CallView("issue_cash", common.JSONMarshall(views.IssueCash{
		Receiver: network.Identity("buyer"),
		Typ:      "USD",
		Quantity: 10,
		Approver: network.Identity("token_approver"),
	}))
	Expect(err).NotTo(HaveOccurred())
	time.Sleep(5 * time.Second)

	houseIDBoxed, err := network.Client("house_issuer").CallView("issue_house", common.JSONMarshall(views.IssueHouse{
		Owner:     network.Identity("seller"),
		Address:   "5th avenue",
		Valuation: 10,
		Approver:  network.Identity("house_approver"),
	}))
	Expect(err).NotTo(HaveOccurred())
	time.Sleep(5 * time.Second)

	_, err = network.Client("seller").CallView("sell", common.JSONMarshall(views.Sell{
		HouseID: common.JSONUnmarshalString(houseIDBoxed),
		Buyer:   network.Identity("buyer"),
		Approvers: []view.Identity{
			network.Identity("token_approver"),
			network.Identity("house_approver"),
		},
	}))
	Expect(err).NotTo(HaveOccurred())
}

func registerAuditor(network *integration.Infrastructure) {
	_, err := network.Client("auditor").CallView("register", nil)
	Expect(err).NotTo(HaveOccurred())
}
