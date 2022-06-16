/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interop

import (
	"math/big"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	. "github.com/onsi/gomega"
)

func TestExchangeSingleFabricNetwork(network *integration.Infrastructure) {
	registerAuditor(network)

	issueCash(network, "", "USD", 110, "alice")
	checkBalance(network, "alice", "", "USD", 110)
	issueCash(network, "", "USD", 10, "alice")
	checkBalance(network, "alice", "", "USD", 120)

	h := listIssuerHistory(network, "", "USD")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(120))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = listIssuerHistory(network, "", "EUR")
	Expect(h.Count()).To(BeEquivalentTo(0))

	issueCash(network, "", "EUR", 10, "bob")
	checkBalance(network, "bob", "", "EUR", 10)
	issueCash(network, "", "EUR", 10, "bob")
	checkBalance(network, "bob", "", "EUR", 20)
	issueCash(network, "", "EUR", 10, "bob")
	checkBalance(network, "bob", "", "EUR", 30)

	h = listIssuerHistory(network, "", "USD")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(120))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = listIssuerHistory(network, "", "EUR")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(30))).To(BeEquivalentTo(0))
	Expect(h.ByType("EUR").Count()).To(BeEquivalentTo(h.Count()))

	checkBalance(network, "alice", "", "USD", 120)
	checkBalance(network, "bob", "", "EUR", 30)
}
