/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interop

import (
	"crypto"
	"math/big"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	. "github.com/onsi/gomega"
)

func TestHTLCSingleFabricNetwork(network *integration.Infrastructure) {
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
	checkBalance(network, "alice", "", "EUR", 0)
	checkBalance(network, "bob", "", "EUR", 30)
	checkBalance(network, "bob", "", "USD", 0)

	// htlc (lock, reclaim)
	htlcLock(network, token.TMSID{}, "alice", "", "USD", 10, "bob", 10*time.Second, nil, crypto.SHA512)
	time.Sleep(15 * time.Second)
	checkBalance(network, "alice", "", "USD", 110)
	checkBalance(network, "alice", "", "EUR", 0)
	checkBalance(network, "bob", "", "EUR", 30)
	checkBalance(network, "bob", "", "USD", 0)

	htlcReclaimAll(network, "alice", "")
	checkBalance(network, "alice", "", "USD", 120)
	checkBalance(network, "alice", "", "EUR", 0)
	checkBalance(network, "bob", "", "EUR", 30)
	checkBalance(network, "bob", "", "USD", 0)

	// htlc (lock, claim)
	defaultTMSID := token.TMSID{}
	preImage, _ := htlcLock(network, defaultTMSID, "alice", "", "USD", 20, "bob", 1*time.Hour, nil, crypto.SHA3_256)
	checkBalance(network, "alice", "", "USD", 100, token.WithTMSID(defaultTMSID))
	checkBalance(network, "alice", "", "EUR", 0)
	checkBalance(network, "bob", "", "EUR", 30)
	checkBalance(network, "bob", "", "USD", 0)

	htlcClaim(network, defaultTMSID, "bob", "", preImage)
	checkBalance(network, "alice", "", "USD", 100, token.WithTMSID(defaultTMSID))
	checkBalance(network, "alice", "", "EUR", 0)
	checkBalance(network, "bob", "", "EUR", 30)
	checkBalance(network, "bob", "", "USD", 20, token.WithTMSID(defaultTMSID))

	// payment limit reached
	htlcLock(network, defaultTMSID, "alice", "", "USD", uint64(views.Limit+10), "bob", 1*time.Hour, nil, crypto.SHA3_256, "payment limit reached")
	checkBalance(network, "alice", "", "USD", 100, token.WithTMSID(defaultTMSID))
	checkBalance(network, "alice", "", "EUR", 0)
	checkBalance(network, "bob", "", "EUR", 30)
	checkBalance(network, "bob", "", "USD", 20, token.WithTMSID(defaultTMSID))
}

func TestHTLCTwoFabricNetworks(network *integration.Infrastructure) {
	alpha := token.TMSID{Network: "alpha"}
	beta := token.TMSID{Network: "beta"}

	registerAuditor(network, token.WithTMSID(alpha))
	registerAuditor(network, token.WithTMSID(beta))

	tmsIssueCash(network, alpha, "issuer", "", "EUR", 30, "alice")
	checkBalance(network, "alice", "", "EUR", 30, token.WithTMSID(alpha))

	tmsIssueCash(network, beta, "issuer", "", "USD", 30, "bob")
	checkBalance(network, "bob", "", "USD", 30, token.WithTMSID(beta))

	preImage, hash := htlcLock(network, alpha, "alice", "", "EUR", 10, "bob", 1*time.Hour, nil, 0)
	htlcLock(network, beta, "bob", "", "USD", 10, "alice", 1*time.Hour, hash, 0)

	htlcClaim(network, beta, "alice", "", preImage)
	htlcClaim(network, alpha, "bob", "", preImage)

	checkBalance(network, "alice", "", "EUR", 20, token.WithTMSID(alpha))
	checkBalance(network, "bob", "", "EUR", 10, token.WithTMSID(alpha))
	checkBalance(network, "alice", "", "USD", 10, token.WithTMSID(beta))
	checkBalance(network, "bob", "", "USD", 20, token.WithTMSID(beta))

	// Try to claim again and get an error

	htlcClaim(network, beta, "alice", "", preImage, "expected only one htlc script to match")
	htlcClaim(network, alpha, "bob", "", preImage, "expected only one htlc script to match")

	checkBalance(network, "alice", "", "EUR", 20, token.WithTMSID(alpha))
	checkBalance(network, "bob", "", "EUR", 10, token.WithTMSID(alpha))
	checkBalance(network, "alice", "", "USD", 10, token.WithTMSID(beta))
	checkBalance(network, "bob", "", "USD", 20, token.WithTMSID(beta))

	// Try to claim without locking

	htlcClaim(network, beta, "alice", "", nil, "expected only one htlc script to match")
	htlcClaim(network, alpha, "bob", "", nil, "expected only one htlc script to match")

	checkBalance(network, "alice", "", "EUR", 20, token.WithTMSID(alpha))
	checkBalance(network, "bob", "", "EUR", 10, token.WithTMSID(alpha))
	checkBalance(network, "alice", "", "USD", 10, token.WithTMSID(beta))
	checkBalance(network, "bob", "", "USD", 20, token.WithTMSID(beta))
}

func TestHTLCNoCrossClaimTwoFabricNetworks(network *integration.Infrastructure) {
	alpha := token.TMSID{Network: "alpha"}
	beta := token.TMSID{Network: "beta"}

	registerAuditor(network, token.WithTMSID(alpha))
	registerAuditor(network, token.WithTMSID(beta))

	tmsIssueCash(network, alpha, "issuer", "", "EUR", 30, "alice.id1")
	checkBalance(network, "alice", "alice.id1", "EUR", 30, token.WithTMSID(alpha))

	tmsIssueCash(network, beta, "issuer", "", "USD", 30, "bob.id1")
	checkBalance(network, "bob", "bob.id1", "USD", 30, token.WithTMSID(beta))

	preImage, hash := htlcLock(network, alpha, "alice", "alice.id1", "EUR", 10, "alice.id2", 30*time.Second, nil, 0)
	htlcLock(network, beta, "bob", "bob.id1", "USD", 10, "bob.id2", 30*time.Second, hash, 0)

	go func() { htlcClaim(network, alpha, "alice", "alice.id2", preImage) }()
	go func() { htlcClaim(network, beta, "bob", "bob.id2", preImage) }()
	scan(network, "alice", hash, crypto.SHA256, token.WithTMSID(alpha))
	scan(network, "bob", hash, crypto.SHA256, token.WithTMSID(beta))

	checkBalance(network, "alice", "alice.id1", "EUR", 20, token.WithTMSID(alpha))
	checkBalance(network, "alice", "alice.id2", "EUR", 10, token.WithTMSID(alpha))
	checkBalance(network, "bob", "bob.id1", "USD", 20, token.WithTMSID(beta))
	checkBalance(network, "bob", "bob.id2", "USD", 10, token.WithTMSID(beta))
}

func TestFastExchange(network *integration.Infrastructure) {
	alpha := token.TMSID{Network: "alpha"}
	beta := token.TMSID{Network: "beta"}

	registerAuditor(network, token.WithTMSID(alpha))
	registerAuditor(network, token.WithTMSID(beta))

	tmsIssueCash(network, alpha, "issuer", "", "EUR", 30, "alice")
	checkBalance(network, "alice", "", "EUR", 30, token.WithTMSID(alpha))

	tmsIssueCash(network, beta, "issuer", "", "USD", 30, "bob")
	checkBalance(network, "bob", "", "USD", 30, token.WithTMSID(beta))

	fastExchange(network, "alice", "bob", alpha, "EUR", 10, beta, "USD", 10, 1*time.Hour)

	checkBalance(network, "alice", "", "EUR", 20, token.WithTMSID(alpha))
	checkBalance(network, "bob", "", "EUR", 10, token.WithTMSID(alpha))

	checkBalance(network, "alice", "", "USD", 10, token.WithTMSID(beta))
	checkBalance(network, "bob", "", "USD", 20, token.WithTMSID(beta))
}
