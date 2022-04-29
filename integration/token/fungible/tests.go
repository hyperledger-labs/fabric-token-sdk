/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fungible

import (
	"crypto/rand"
	"math/big"
	"strings"
	"time"

	. "github.com/onsi/gomega"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/query"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func TestAll(network *integration.Infrastructure) {
	RegisterAuditor(network)

	// Rest of the test
	IssueCash(network, "", "USD", 110, "alice")
	CheckBalance(network, "alice", "", "USD", 110)
	IssueCash(network, "", "USD", 10, "alice")
	CheckBalance(network, "alice", "", "USD", 120)
	CheckBalance(network, "alice", "alice", "USD", 120)

	h := ListIssuerHistory(network, "", "USD")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(120))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = ListIssuerHistory(network, "", "EUR")
	Expect(h.Count()).To(BeEquivalentTo(0))

	IssueCash(network, "", "EUR", 10, "bob")
	CheckBalance(network, "bob", "", "EUR", 10)
	IssueCash(network, "", "EUR", 10, "bob")
	CheckBalance(network, "bob", "", "EUR", 20)
	IssueCash(network, "", "EUR", 10, "bob")
	CheckBalance(network, "bob", "", "EUR", 30)

	h = ListIssuerHistory(network, "", "USD")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(120))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = ListIssuerHistory(network, "", "EUR")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(30))).To(BeEquivalentTo(0))
	Expect(h.ByType("EUR").Count()).To(BeEquivalentTo(h.Count()))

	CheckBalance(network, "alice", "", "USD", 120)
	CheckBalance(network, "bob", "", "EUR", 30)

	network.StopFSCNode("alice")
	time.Sleep(3 * time.Second)
	network.StartFSCNode("alice")
	time.Sleep(5 * time.Second)

	TransferCash(network, "alice", "", "USD", 110, "bob")
	ut := ListUnspentTokens(network, "alice", "", "USD")
	Expect(ut.Count() > 0).To(BeTrue())
	Expect(ut.Sum(64).ToBigInt().Cmp(big.NewInt(10))).To(BeEquivalentTo(0))
	Expect(ut.ByType("USD").Count()).To(BeEquivalentTo(ut.Count()))
	ut = ListUnspentTokens(network, "bob", "", "USD")
	Expect(ut.Count() > 0).To(BeTrue())
	Expect(ut.Sum(64).ToBigInt().Cmp(big.NewInt(110))).To(BeEquivalentTo(0))
	Expect(ut.ByType("USD").Count()).To(BeEquivalentTo(ut.Count()))

	CheckBalance(network, "alice", "", "USD", 10)
	CheckBalance(network, "alice", "", "EUR", 0)
	CheckBalance(network, "bob", "", "EUR", 30)
	CheckBalance(network, "bob", "bob", "EUR", 30)
	CheckBalance(network, "bob", "", "USD", 110)

	SwapCash(network, "alice", "", "USD", 10, "EUR", 10, "bob")

	CheckBalance(network, "alice", "", "USD", 0)
	CheckBalance(network, "alice", "", "EUR", 10)
	CheckBalance(network, "bob", "", "EUR", 20)
	CheckBalance(network, "bob", "", "USD", 120)

	RedeemCash(network, "bob", "", "USD", 10)
	CheckBalance(network, "bob", "", "USD", 110)

	// Check self endpoints
	IssueCash(network, "", "USD", 110, "issuer")
	IssueCash(network, "", "EUR", 150, "issuer")
	IssueCash(network, "issuer.id1", "EUR", 10, "issuer.owner")

	h = ListIssuerHistory(network, "", "USD")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(230))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = ListIssuerHistory(network, "", "EUR")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(180))).To(BeEquivalentTo(0))
	Expect(h.ByType("EUR").Count()).To(BeEquivalentTo(h.Count()))

	// Restart the auditor
	network.StopFSCNode("auditor")
	time.Sleep(3 * time.Second)
	network.StartFSCNode("auditor")
	time.Sleep(5 * time.Second)
	RegisterAuditor(network)

	CheckBalance(network, "issuer", "", "USD", 110)
	CheckBalance(network, "issuer", "", "EUR", 150)
	CheckBalance(network, "issuer", "issuer.owner", "EUR", 10)

	TransferCash(network, "issuer", "", "USD", 50, "issuer")
	CheckBalance(network, "issuer", "", "USD", 110)
	CheckBalance(network, "issuer", "", "EUR", 150)

	TransferCash(network, "issuer", "", "USD", 50, "manager")
	TransferCash(network, "issuer", "", "EUR", 20, "manager")
	CheckBalance(network, "issuer", "", "USD", 60)
	CheckBalance(network, "issuer", "", "EUR", 130)
	CheckBalance(network, "manager", "", "USD", 50)
	CheckBalance(network, "manager", "", "EUR", 20)

	// Play with wallets
	TransferCash(network, "manager", "", "USD", 10, "manager.id1")
	TransferCash(network, "manager", "", "USD", 10, "manager.id2")
	TransferCash(network, "manager", "", "USD", 10, "manager.id3")
	CheckBalance(network, "manager", "", "USD", 20)
	CheckBalance(network, "manager", "manager.id1", "USD", 10)
	CheckBalance(network, "manager", "manager.id2", "USD", 10)
	CheckBalance(network, "manager", "manager.id3", "USD", 10)

	TransferCash(network, "manager", "manager.id1", "USD", 10, "manager.id2")
	CheckBalance(network, "manager", "", "USD", 20)
	CheckBalance(network, "manager", "manager.id1", "USD", 0)
	CheckBalance(network, "manager", "manager.id2", "USD", 20)
	CheckBalance(network, "manager", "manager.id3", "USD", 10)

	// Swap among wallets
	TransferCash(network, "manager", "", "EUR", 10, "manager.id1")
	CheckBalance(network, "manager", "", "EUR", 10)
	CheckBalance(network, "manager", "manager.id1", "EUR", 10)

	SwapCash(network, "manager", "manager.id1", "EUR", 10, "USD", 10, "manager.id2")
	CheckBalance(network, "manager", "", "USD", 20)
	CheckBalance(network, "manager", "", "EUR", 10)
	CheckBalance(network, "manager", "manager.id1", "USD", 10)
	CheckBalance(network, "manager", "manager.id1", "EUR", 0)
	CheckBalance(network, "manager", "manager.id2", "USD", 10)
	CheckBalance(network, "manager", "manager.id2", "EUR", 10)
	CheckBalance(network, "manager", "manager.id3", "USD", 10)

	// no more USD can be issued, reached quota of 220
	IssueCashFail(network, "USD", 10, "alice")

	CheckBalance(network, "alice", "", "USD", 0)
	CheckBalance(network, "alice", "", "EUR", 10)

	// limits
	CheckBalance(network, "alice", "", "USD", 0)
	CheckBalance(network, "alice", "", "EUR", 10)
	CheckBalance(network, "bob", "", "EUR", 20)
	CheckBalance(network, "bob", "", "USD", 110)
	IssueCash(network, "", "EUR", 2200, "alice")
	IssueCash(network, "", "EUR", 2000, "charlie")
	CheckBalance(network, "alice", "", "EUR", 2210)
	CheckBalance(network, "charlie", "", "EUR", 2000)
	TransferCash(network, "alice", "", "EUR", 210, "bob", "payment limit reached", "alice", "[EUR][210]")

	TransferCash(network, "alice", "", "EUR", 200, "bob")
	TransferCash(network, "alice", "", "EUR", 200, "bob")
	TransferCash(network, "alice", "", "EUR", 200, "bob")
	TransferCash(network, "alice", "", "EUR", 200, "bob")
	TransferCash(network, "alice", "", "EUR", 200, "bob")
	TransferCash(network, "alice", "", "EUR", 200, "bob")
	TransferCash(network, "alice", "", "EUR", 200, "bob")
	TransferCash(network, "alice", "", "EUR", 200, "bob")
	TransferCash(network, "alice", "", "EUR", 200, "bob")
	TransferCash(network, "alice", "", "EUR", 200, "bob", "cumulative payment limit reached", "alice", "[EUR][2000]")
	TransferCash(network, "charlie", "", "EUR", 200, "bob")
	TransferCash(network, "charlie", "", "EUR", 200, "bob")
	TransferCash(network, "charlie", "", "EUR", 200, "bob")
	TransferCash(network, "charlie", "", "EUR", 200, "bob")
	TransferCash(network, "charlie", "", "EUR", 200, "bob")
	TransferCash(network, "charlie", "", "EUR", 200, "bob", "holding limit reached", "bob", "[EUR][3020]")
	CheckBalance(network, "bob", "", "EUR", 2820)

	// Routing
	IssueCash(network, "", "EUR", 10, "alice.id1")
	TransferCash(network, "alice", "alice.id1", "EUR", 10, "bob.id1")
	CheckBalance(network, "alice", "alice.id1", "EUR", 0)
	CheckBalance(network, "bob", "bob.id1", "EUR", 10)

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
	CheckBalance(network, "bob", "", "EUR", 2820-sum)

	// Transfer With Selector
	IssueCash(network, "", "YUAN", 17, "alice")
	TransferCashWithSelector(network, "alice", "", "YUAN", 10, "bob")
	CheckBalance(network, "alice", "", "YUAN", 7)
	CheckBalance(network, "bob", "", "YUAN", 10)
	TransferCashWithSelector(network, "alice", "", "YUAN", 10, "bob", "pineapple", "insufficient funds")
	concurrentTransfers = make([]chan error, 2)
	for i := range concurrentTransfers {
		concurrentTransfers[i] = make(chan error, 1)

		transfer := concurrentTransfers[i]
		go func() {
			txid, err := network.Client("bob").CallView("transferWithSelector", common.JSONMarshall(&views.Transfer{
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
			Expect(network.Client("charlie").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
			transfer <- nil
		}()
	}
	// one must fail, the other succeeded
	var errors []error
	for _, transfer := range concurrentTransfers {
		errors = append(errors, <-transfer)
	}
	Expect((errors[0] == nil && errors[1] != nil) || (errors[0] != nil && errors[1] == nil)).To(BeTrue())
	var errStr string
	if errors[0] == nil {
		errStr = errors[1].Error()
	} else {
		errStr = errors[0].Error()
	}
	v := strings.Contains(errStr, "pineapple") || strings.Contains(errStr, "lemonade")
	Expect(v).To(BeEquivalentTo(true))

	CheckBalance(network, "bob", "", "YUAN", 3)
	CheckBalance(network, "alice", "", "YUAN", 7)
	CheckBalance(network, "charlie", "", "YUAN", 7)

	// Transfer by IDs
	txID := IssueCash(network, "", "CHF", 17, "alice")
	TransferCashByIDs(network, "alice", "", []*token2.ID{{TxId: txID, Index: 0}}, 17, "bob", true, "test release")
	// the previous call should not keep the token locked if release is successful
	txID = TransferCashByIDs(network, "alice", "", []*token2.ID{{TxId: txID, Index: 0}}, 17, "bob", false)
	RedeemCashByIDs(network, "bob", "", []*token2.ID{{TxId: txID, Index: 0}}, 17)
}

func RegisterAuditor(network *integration.Infrastructure) {
	_, err := network.Client("auditor").CallView("register", nil)
	Expect(err).NotTo(HaveOccurred())
}

func RegisterCertifier(network *integration.Infrastructure) {
	_, err := network.Client("certifier").CallView("register", nil)
	Expect(err).NotTo(HaveOccurred())
}

func IssueCash(network *integration.Infrastructure, wallet string, typ string, amount uint64, receiver string) string {
	txid, err := network.Client("issuer").CallView("issue", common.JSONMarshall(&views.IssueCash{
		IssuerWallet: wallet,
		TokenType:    typ,
		Quantity:     amount,
		Recipient:    network.Identity(receiver),
	}))
	Expect(err).NotTo(HaveOccurred())
	Expect(network.Client(receiver).IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
	Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())

	return common.JSONUnmarshalString(txid)
}

func IssueCashFail(network *integration.Infrastructure, typ string, amount uint64, receiver string) {
	_, err := network.Client("issuer").CallView("issue", common.JSONMarshall(&views.IssueCash{
		TokenType: typ,
		Quantity:  amount,
		Recipient: network.Identity(receiver),
	}))
	Expect(err).To(HaveOccurred())
}

func CheckBalance(network *integration.Infrastructure, id string, wallet string, typ string, expected uint64) {
	b, err := query.NewClient(network.Client(id)).WalletBalance(wallet, typ)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(b)).To(BeEquivalentTo(1))
	Expect(b[0].Type).To(BeEquivalentTo(typ))
	q, err := token2.ToQuantity(b[0].Quantity, 64)
	Expect(err).NotTo(HaveOccurred())
	expectedQ := token2.NewQuantityFromUInt64(expected)
	Expect(expectedQ.Cmp(q)).To(BeEquivalentTo(0), "[%s]!=[%s]", expected, q)
}

func ListIssuerHistory(network *integration.Infrastructure, wallet string, typ string) *token2.IssuedTokens {
	res, err := network.Client("issuer").CallView("history", common.JSONMarshall(&views.ListIssuedTokens{
		Wallet:    wallet,
		TokenType: typ,
	}))
	Expect(err).NotTo(HaveOccurred())

	issuedTokens := &token2.IssuedTokens{}
	common.JSONUnmarshal(res.([]byte), issuedTokens)
	return issuedTokens
}

func ListUnspentTokens(network *integration.Infrastructure, id string, wallet string, typ string) *token2.UnspentTokens {
	res, err := network.Client(id).CallView("history", common.JSONMarshall(&views.ListUnspentTokens{
		Wallet:    wallet,
		TokenType: typ,
	}))
	Expect(err).NotTo(HaveOccurred())

	unspentTokens := &token2.UnspentTokens{}
	common.JSONUnmarshal(res.([]byte), unspentTokens)
	return unspentTokens
}

func TransferCash(network *integration.Infrastructure, id string, wallet string, typ string, amount uint64, receiver string, errorMsgs ...string) {
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

func TransferCashByIDs(network *integration.Infrastructure, id string, wallet string, ids []*token2.ID, amount uint64, receiver string, failToRelease bool, errorMsgs ...string) string {
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

func TransferCashWithSelector(network *integration.Infrastructure, id string, wallet string, typ string, amount uint64, receiver string, errorMsgs ...string) {
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

func RedeemCash(network *integration.Infrastructure, id string, wallet string, typ string, amount uint64) {
	txid, err := network.Client(id).CallView("redeem", common.JSONMarshall(&views.Redeem{
		Wallet: wallet,
		Type:   typ,
		Amount: amount,
	}))
	Expect(err).NotTo(HaveOccurred())
	Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
}

func RedeemCashByIDs(network *integration.Infrastructure, id string, wallet string, ids []*token2.ID, amount uint64) {
	txid, err := network.Client(id).CallView("redeem", common.JSONMarshall(&views.Redeem{
		Wallet:   wallet,
		Type:     "",
		TokenIDs: ids,
		Amount:   amount,
	}))
	Expect(err).NotTo(HaveOccurred())
	Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
}

func SwapCash(network *integration.Infrastructure, id string, wallet string, typeLeft string, amountLeft uint64, typRight string, amountRight uint64, receiver string) {
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
