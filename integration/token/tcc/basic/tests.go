/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package basic

import (
	"crypto/rand"
	"math/big"
	"time"

	. "github.com/onsi/gomega"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/tcc/basic/views"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/query"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func TestAll(network *integration.Infrastructure) {
	// registerIssuers(network)
	// registerCertifier(network)
	registerAuditor(network)

	// Rest of the test
	issueCash(network, "", "USD", 110, "alice")
	checkBalance(network, "alice", "", "USD", 110)
	issueCash(network, "", "USD", 10, "alice")
	checkBalance(network, "alice", "", "USD", 120)
	checkBalance(network, "alice", "alice", "USD", 120)

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

	network.StopFSCNode("alice")
	time.Sleep(3 * time.Second)
	network.StartFSCNode("alice")
	time.Sleep(5 * time.Second)

	transferCash(network, "alice", "", "USD", 110, "bob")
	ut := listUnspentTokens(network, "alice", "", "USD")
	Expect(ut.Count() > 0).To(BeTrue())
	Expect(ut.Sum(64).ToBigInt().Cmp(big.NewInt(10))).To(BeEquivalentTo(0))
	Expect(ut.ByType("USD").Count()).To(BeEquivalentTo(ut.Count()))
	ut = listUnspentTokens(network, "bob", "", "USD")
	Expect(ut.Count() > 0).To(BeTrue())
	Expect(ut.Sum(64).ToBigInt().Cmp(big.NewInt(110))).To(BeEquivalentTo(0))
	Expect(ut.ByType("USD").Count()).To(BeEquivalentTo(ut.Count()))

	checkBalance(network, "alice", "", "USD", 10)
	checkBalance(network, "alice", "", "EUR", 0)
	checkBalance(network, "bob", "", "EUR", 30)
	checkBalance(network, "bob", "bob", "EUR", 30)
	checkBalance(network, "bob", "", "USD", 110)

	swapCash(network, "alice", "", "USD", 10, "EUR", 10, "bob")

	checkBalance(network, "alice", "", "USD", 0)
	checkBalance(network, "alice", "", "EUR", 10)
	checkBalance(network, "bob", "", "EUR", 20)
	checkBalance(network, "bob", "", "USD", 120)

	redeemCash(network, "bob", "", "USD", 10)
	checkBalance(network, "bob", "", "USD", 110)

	// Check self endpoints
	issueCash(network, "", "USD", 110, "issuer")
	issueCash(network, "", "EUR", 150, "issuer")
	issueCash(network, "issuer.id1", "EUR", 10, "issuer.owner")

	h = listIssuerHistory(network, "", "USD")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(230))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = listIssuerHistory(network, "", "EUR")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(180))).To(BeEquivalentTo(0))
	Expect(h.ByType("EUR").Count()).To(BeEquivalentTo(h.Count()))

	// Restart the auditor
	network.StopFSCNode("auditor")
	time.Sleep(3 * time.Second)
	network.StartFSCNode("auditor")
	time.Sleep(5 * time.Second)
	registerAuditor(network)

	checkBalance(network, "issuer", "", "USD", 110)
	checkBalance(network, "issuer", "", "EUR", 150)
	checkBalance(network, "issuer", "issuer.owner", "EUR", 10)

	transferCash(network, "issuer", "", "USD", 50, "issuer")
	checkBalance(network, "issuer", "", "USD", 110)
	checkBalance(network, "issuer", "", "EUR", 150)

	transferCash(network, "issuer", "", "USD", 50, "manager")
	transferCash(network, "issuer", "", "EUR", 20, "manager")
	checkBalance(network, "issuer", "", "USD", 60)
	checkBalance(network, "issuer", "", "EUR", 130)
	checkBalance(network, "manager", "", "USD", 50)
	checkBalance(network, "manager", "", "EUR", 20)

	// Play with wallets
	transferCash(network, "manager", "", "USD", 10, "manager.id1")
	transferCash(network, "manager", "", "USD", 10, "manager.id2")
	transferCash(network, "manager", "", "USD", 10, "manager.id3")
	checkBalance(network, "manager", "", "USD", 20)
	checkBalance(network, "manager", "manager.id1", "USD", 10)
	checkBalance(network, "manager", "manager.id2", "USD", 10)
	checkBalance(network, "manager", "manager.id3", "USD", 10)

	transferCash(network, "manager", "manager.id1", "USD", 10, "manager.id2")
	checkBalance(network, "manager", "", "USD", 20)
	checkBalance(network, "manager", "manager.id1", "USD", 0)
	checkBalance(network, "manager", "manager.id2", "USD", 20)
	checkBalance(network, "manager", "manager.id3", "USD", 10)

	// Swap among wallets
	transferCash(network, "manager", "", "EUR", 10, "manager.id1")
	checkBalance(network, "manager", "", "EUR", 10)
	checkBalance(network, "manager", "manager.id1", "EUR", 10)

	swapCash(network, "manager", "manager.id1", "EUR", 10, "USD", 10, "manager.id2")
	checkBalance(network, "manager", "", "USD", 20)
	checkBalance(network, "manager", "", "EUR", 10)
	checkBalance(network, "manager", "manager.id1", "USD", 10)
	checkBalance(network, "manager", "manager.id1", "EUR", 0)
	checkBalance(network, "manager", "manager.id2", "USD", 10)
	checkBalance(network, "manager", "manager.id2", "EUR", 10)
	checkBalance(network, "manager", "manager.id3", "USD", 10)

	// no more USD can be issued, reached quota of 220
	issueCashFail(network, "USD", 10, "alice")

	checkBalance(network, "alice", "", "USD", 0)
	checkBalance(network, "alice", "", "EUR", 10)

	// limits
	checkBalance(network, "alice", "", "USD", 0)
	checkBalance(network, "alice", "", "EUR", 10)
	checkBalance(network, "bob", "", "EUR", 20)
	checkBalance(network, "bob", "", "USD", 110)
	issueCash(network, "", "EUR", 2200, "alice")
	issueCash(network, "", "EUR", 2000, "charlie")
	checkBalance(network, "alice", "", "EUR", 2210)
	checkBalance(network, "charlie", "", "EUR", 2000)
	transferCash(network, "alice", "", "EUR", 210, "bob", "payment limit reached", "alice", "[EUR][210]")

	transferCash(network, "alice", "", "EUR", 200, "bob")
	transferCash(network, "alice", "", "EUR", 200, "bob")
	transferCash(network, "alice", "", "EUR", 200, "bob")
	transferCash(network, "alice", "", "EUR", 200, "bob")
	transferCash(network, "alice", "", "EUR", 200, "bob")
	transferCash(network, "alice", "", "EUR", 200, "bob")
	transferCash(network, "alice", "", "EUR", 200, "bob")
	transferCash(network, "alice", "", "EUR", 200, "bob")
	transferCash(network, "alice", "", "EUR", 200, "bob")
	transferCash(network, "alice", "", "EUR", 200, "bob", "cumulative payment limit reached", "alice", "[EUR][2000]")
	transferCash(network, "charlie", "", "EUR", 200, "bob")
	transferCash(network, "charlie", "", "EUR", 200, "bob")
	transferCash(network, "charlie", "", "EUR", 200, "bob")
	transferCash(network, "charlie", "", "EUR", 200, "bob")
	transferCash(network, "charlie", "", "EUR", 200, "bob")
	transferCash(network, "charlie", "", "EUR", 200, "bob", "holding limit reached", "bob", "[EUR][3020]")
	checkBalance(network, "bob", "", "EUR", 2820)

	// Routing
	issueCash(network, "", "EUR", 10, "alice.id1")
	transferCash(network, "alice", "alice.id1", "EUR", 10, "bob.id1")
	checkBalance(network, "alice", "alice.id1", "EUR", 0)
	checkBalance(network, "bob", "bob.id1", "EUR", 10)

	// Concurrent transfers
	concurrentTransfers := make([]chan error, 5)
	var sum uint64
	for i := range concurrentTransfers {
		concurrentTransfers[i] = make(chan error, 1)

		transfer := concurrentTransfers[i]
		r, err := rand.Int(rand.Reader, big.NewInt(200))
		Expect(err).ToNot(HaveOccurred())
		v := r.Uint64() + 1
		sum += v
		go func() {
			_, err := network.Client("bob").CallView("transferWithSelector", common.JSONMarshall(&views.Transfer{
				Wallet:    "",
				Type:      "EUR",
				Amount:    v,
				Recipient: network.Identity("alice"),
				Retry:     true,
			}))
			if err != nil {
				transfer <- err
				return
			}
			transfer <- nil
		}()
	}
	for _, transfer := range concurrentTransfers {
		err := <-transfer
		Expect(err).ToNot(HaveOccurred())
	}
	checkBalance(network, "bob", "", "EUR", 2820-sum)

	// Transfer With Selector
	issueCash(network, "", "YUAN", 17, "alice")
	transferCashWithSelector(network, "alice", "", "YUAN", 10, "bob")
	checkBalance(network, "alice", "", "YUAN", 7)
	checkBalance(network, "bob", "", "YUAN", 10)
	transferCashWithSelector(network, "alice", "", "YUAN", 10, "bob", "pineapple", "insufficient funds")
	concurrentTransfers = make([]chan error, 2)
	for i := range concurrentTransfers {
		concurrentTransfers[i] = make(chan error, 1)

		transfer := concurrentTransfers[i]
		go func() {
			_, err := network.Client("bob").CallView("transferWithSelector", common.JSONMarshall(&views.Transfer{
				Wallet:    "",
				Type:      "YUAN",
				Amount:    7,
				Recipient: network.Identity("charlie"),
				Retry:     false,
			}))
			if err != nil {
				transfer <- err
				return
			}
			transfer <- nil
		}()
	}
	// one must fail, the other succeeded
	var errors []error
	for _, transfer := range concurrentTransfers {
		errors = append(errors, <-transfer)
	}
	Expect((errors[0] == nil && errors[1] != nil) || (errors[0] != nil && errors[1] == nil)).To(BeTrue())
	if errors[0] == nil {
		Expect(errors[1].Error()).To(ContainSubstring("lemonade"))
	} else {
		Expect(errors[0].Error()).To(ContainSubstring("lemonade"))

	}
	checkBalance(network, "bob", "", "YUAN", 3)
	checkBalance(network, "alice", "", "YUAN", 7)
	checkBalance(network, "charlie", "", "YUAN", 7)

	// Transfer by IDs
	txID := issueCash(network, "", "CHF", 17, "alice")
	transferCashByIDs(network, "alice", "", []*token2.ID{{TxId: txID, Index: 0}}, 17, "bob", true, "test release")
	// the previous call should not keep the token locked if release is successful
	txID = transferCashByIDs(network, "alice", "", []*token2.ID{{TxId: txID, Index: 0}}, 17, "bob", false)
	redeemCashByIDs(network, "bob", "", []*token2.ID{{TxId: txID, Index: 0}}, 17)
}

/*
func registerIssuers(network *integration.Infrastructure) {
	_, err := network.Client("issuer").CallView("register", common.JSONMarshall(&views.RegisterIssuer{
		TokenTypes: []string{"USD", "EUR"},
	}))
	Expect(err).NotTo(HaveOccurred())
}*/

func registerAuditor(network *integration.Infrastructure) {
	_, err := network.Client("auditor").CallView("register", nil)
	Expect(err).NotTo(HaveOccurred())
}

func registerCertifier(network *integration.Infrastructure) {
	_, err := network.Client("certifier").CallView("register", nil)
	Expect(err).NotTo(HaveOccurred())
}

func issueCash(network *integration.Infrastructure, wallet string, typ string, amount uint64, receiver string) string {
	txid, err := network.Client("issuer").CallView("issue", common.JSONMarshall(&views.IssueCash{
		IssuerWallet: wallet,
		TokenType:    typ,
		Quantity:     amount,
		Recipient:    network.Identity(receiver),
	}))
	Expect(err).NotTo(HaveOccurred())
	Expect(network.Client(receiver).IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())

	return common.JSONUnmarshalString(txid)
}

func listIssuerHistory(network *integration.Infrastructure, wallet string, typ string) *token2.IssuedTokens {
	res, err := network.Client("issuer").CallView("history", common.JSONMarshall(&views.ListIssuedTokens{
		Wallet:    wallet,
		TokenType: typ,
	}))
	Expect(err).NotTo(HaveOccurred())

	issuedTokens := &token2.IssuedTokens{}
	common.JSONUnmarshal(res.([]byte), issuedTokens)
	return issuedTokens
}

func issueCashFail(network *integration.Infrastructure, typ string, amount uint64, receiver string) {
	_, err := network.Client("issuer").CallView("issue", common.JSONMarshall(&views.IssueCash{
		TokenType: typ,
		Quantity:  amount,
		Recipient: network.Identity(receiver),
	}))
	Expect(err).To(HaveOccurred())
}

func transferCash(network *integration.Infrastructure, id string, wallet string, typ string, amount uint64, receiver string, errorMsgs ...string) {
	txid, err := network.Client(id).CallView("transfer", common.JSONMarshall(&views.Transfer{
		Wallet:    wallet,
		Type:      typ,
		Amount:    amount,
		Recipient: network.Identity(receiver),
	}))
	if len(errorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Client(receiver).IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
		Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range errorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg))
		}
		time.Sleep(5 * time.Second)
	}
}

func transferCashByIDs(network *integration.Infrastructure, id string, wallet string, ids []*token2.ID, amount uint64, receiver string, failToRelease bool, errorMsgs ...string) string {
	txid, err := network.Client(id).CallView("transfer", common.JSONMarshall(&views.Transfer{
		Wallet:        wallet,
		Type:          "",
		TokenIDs:      ids,
		Amount:        amount,
		Recipient:     network.Identity(receiver),
		FailToRelease: failToRelease,
	}))
	if len(errorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Client(receiver).IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
		Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
		return common.JSONUnmarshalString(txid)
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range errorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg))
		}
		time.Sleep(5 * time.Second)
		return ""
	}
}

func transferCashWithSelector(network *integration.Infrastructure, id string, wallet string, typ string, amount uint64, receiver string, errorMsgs ...string) {
	txid, err := network.Client(id).CallView("transferWithSelector", common.JSONMarshall(&views.Transfer{
		Wallet:    wallet,
		Type:      typ,
		Amount:    amount,
		Recipient: network.Identity(receiver),
	}))
	if len(errorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Client(receiver).IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
		Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range errorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg))
		}
		time.Sleep(5 * time.Second)
	}
}

func redeemCash(network *integration.Infrastructure, id string, wallet string, typ string, amount uint64) {
	txid, err := network.Client(id).CallView("redeem", common.JSONMarshall(&views.Redeem{
		Wallet: wallet,
		Type:   typ,
		Amount: amount,
	}))
	Expect(err).NotTo(HaveOccurred())
	Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
}

func redeemCashByIDs(network *integration.Infrastructure, id string, wallet string, ids []*token2.ID, amount uint64) {
	txid, err := network.Client(id).CallView("redeem", common.JSONMarshall(&views.Redeem{
		Wallet:   wallet,
		Type:     "",
		TokenIDs: ids,
		Amount:   amount,
	}))
	Expect(err).NotTo(HaveOccurred())
	Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
}

func swapCash(network *integration.Infrastructure, id string, wallet string, typeLeft string, amountLeft uint64, typRight string, amountRight uint64, receiver string) {
	txid, err := network.Client(id).CallView("swap", common.JSONMarshall(&views.Swap{
		AliceWallet:     wallet,
		FromAliceType:   typeLeft,
		FromAliceAmount: amountLeft,
		FromBobType:     typRight,
		FromBobAmount:   amountRight,
		Bob:             network.Identity(receiver),
	}))
	Expect(err).NotTo(HaveOccurred())
	Expect(network.Client(receiver).IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
	Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
}

func checkBalance(network *integration.Infrastructure, id string, wallet string, typ string, expected uint64) {
	b, err := query.NewClient(network.Client(id)).WalletBalance(wallet, typ)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(b)).To(BeEquivalentTo(1))
	Expect(b[0].Type).To(BeEquivalentTo(typ))
	q, err := token2.ToQuantity(b[0].Quantity, 64)
	Expect(err).NotTo(HaveOccurred())
	expectedQ := token2.NewQuantityFromUInt64(expected)
	Expect(expectedQ.Cmp(q)).To(BeEquivalentTo(0), "[%s]!=[%s]", expected, q)
}

func listUnspentTokens(network *integration.Infrastructure, id string, wallet string, typ string) *token2.UnspentTokens {
	res, err := network.Client(id).CallView("history", common.JSONMarshall(&views.ListUnspentTokens{
		Wallet:    wallet,
		TokenType: typ,
	}))
	Expect(err).NotTo(HaveOccurred())

	unspentTokens := &token2.UnspentTokens{}
	common.JSONUnmarshal(res.([]byte), unspentTokens)
	return unspentTokens
}
