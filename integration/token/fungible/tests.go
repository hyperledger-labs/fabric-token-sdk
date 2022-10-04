/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fungible

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/query"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/gomega"
)

var AuditedTransactions = []*ttxdb.TransactionRecord{
	{
		TxID:         "",
		ActionType:   ttxdb.Issue,
		SenderEID:    "",
		RecipientEID: "alice",
		TokenType:    "USD",
		Amount:       big.NewInt(110),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Issue,
		SenderEID:    "",
		RecipientEID: "alice",
		TokenType:    "USD",
		Amount:       big.NewInt(10),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Issue,
		SenderEID:    "",
		RecipientEID: "bob",
		TokenType:    "EUR",
		Amount:       big.NewInt(10),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Issue,
		SenderEID:    "",
		RecipientEID: "bob",
		TokenType:    "EUR",
		Amount:       big.NewInt(10),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Issue,
		SenderEID:    "",
		RecipientEID: "bob",
		TokenType:    "EUR",
		Amount:       big.NewInt(10),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Transfer,
		SenderEID:    "alice",
		RecipientEID: "bob",
		TokenType:    "USD",
		Amount:       big.NewInt(111),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Transfer,
		SenderEID:    "alice",
		RecipientEID: "alice",
		TokenType:    "USD",
		Amount:       big.NewInt(9),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Transfer,
		SenderEID:    "bob",
		RecipientEID: "bob",
		TokenType:    "USD",
		Amount:       big.NewInt(100),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Redeem,
		SenderEID:    "bob",
		RecipientEID: "",
		TokenType:    "USD",
		Amount:       big.NewInt(11),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Issue,
		SenderEID:    "",
		RecipientEID: "bob",
		TokenType:    "USD",
		Amount:       big.NewInt(10),
		Status:       ttxdb.Confirmed,
	},
}

var AliceAcceptedTransactions = []*ttxdb.TransactionRecord{
	{
		TxID:         "",
		ActionType:   ttxdb.Issue,
		SenderEID:    "",
		RecipientEID: "alice",
		TokenType:    "USD",
		Amount:       big.NewInt(110),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Issue,
		SenderEID:    "",
		RecipientEID: "alice",
		TokenType:    "USD",
		Amount:       big.NewInt(10),
		Status:       ttxdb.Confirmed,
	},
}

var AliceID1AcceptedTransactions = []*ttxdb.TransactionRecord{
	{
		TxID:         "",
		ActionType:   ttxdb.Issue,
		SenderEID:    "",
		RecipientEID: "alice",
		TokenType:    "EUR",
		Amount:       big.NewInt(10),
		Status:       ttxdb.Confirmed,
	},
}

var BobAcceptedTransactions = []*ttxdb.TransactionRecord{
	{
		TxID:         "",
		ActionType:   ttxdb.Issue,
		SenderEID:    "",
		RecipientEID: "bob",
		TokenType:    "EUR",
		Amount:       big.NewInt(10),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Issue,
		SenderEID:    "",
		RecipientEID: "bob",
		TokenType:    "EUR",
		Amount:       big.NewInt(10),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Issue,
		SenderEID:    "",
		RecipientEID: "bob",
		TokenType:    "EUR",
		Amount:       big.NewInt(10),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Transfer,
		SenderEID:    "alice",
		RecipientEID: "bob",
		TokenType:    "USD",
		Amount:       big.NewInt(111),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Transfer,
		SenderEID:    "bob",
		RecipientEID: "bob",
		TokenType:    "USD",
		Amount:       big.NewInt(100),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Redeem,
		SenderEID:    "bob",
		RecipientEID: "",
		TokenType:    "USD",
		Amount:       big.NewInt(11),
		Status:       ttxdb.Confirmed,
	},
}

func TestAll(network *integration.Infrastructure, auditor string) {
	RegisterAuditor(network, auditor)

	CheckPublicParamsViewFactory(network, "issuer", "auditor", "alice", "bob", "charlie", "manager")

	t0 := time.Now()
	// Rest of the test
	IssueCash(network, "", "USD", 110, "alice", auditor, true)
	t1 := time.Now()
	CheckBalanceAndHolding(network, "alice", "", "USD", 110)
	CheckAuditedTransactions(network, AuditedTransactions[:1], nil, nil)
	CheckAuditedTransactions(network, AuditedTransactions[:1], &t0, &t1)
	CheckAcceptedTransactions(network, "alice", "", AliceAcceptedTransactions[:1], nil, nil, nil, ttxdb.Issue)
	CheckAcceptedTransactions(network, "alice", "", AliceAcceptedTransactions[:1], nil, nil, []ttxdb.TxStatus{ttxdb.Confirmed}, ttxdb.Issue)
	CheckAcceptedTransactions(network, "alice", "", AliceAcceptedTransactions[:1], nil, nil, nil)
	CheckAcceptedTransactions(network, "alice", "", AliceAcceptedTransactions[:1], &t0, &t1, nil)

	t2 := time.Now()
	IssueCash(network, "", "USD", 10, "alice", auditor, false)
	t3 := time.Now()
	CheckBalanceAndHolding(network, "alice", "", "USD", 120)
	CheckBalanceAndHolding(network, "alice", "alice", "USD", 120)
	CheckAuditedTransactions(network, AuditedTransactions[:2], nil, nil)
	CheckAuditedTransactions(network, AuditedTransactions[:2], &t0, &t3)
	CheckAuditedTransactions(network, AuditedTransactions[1:2], &t2, &t3)
	CheckAcceptedTransactions(network, "alice", "", AliceAcceptedTransactions[:2], nil, nil, nil)
	CheckAcceptedTransactions(network, "alice", "", AliceAcceptedTransactions[:2], &t0, &t3, nil)
	CheckAcceptedTransactions(network, "alice", "", AliceAcceptedTransactions[1:2], &t2, &t3, nil)

	h := ListIssuerHistory(network, "", "USD")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(120))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = ListIssuerHistory(network, "", "EUR")
	Expect(h.Count()).To(BeEquivalentTo(0))

	t4 := time.Now()
	IssueCash(network, "", "EUR", 10, "bob", auditor, false)
	//t5 := time.Now()
	CheckBalanceAndHolding(network, "bob", "", "EUR", 10)
	IssueCash(network, "", "EUR", 10, "bob", auditor, true)
	//t6 := time.Now()
	CheckBalanceAndHolding(network, "bob", "", "EUR", 20)
	IssueCash(network, "", "EUR", 10, "bob", auditor, false)
	t7 := time.Now()
	CheckBalanceAndHolding(network, "bob", "", "EUR", 30)
	CheckAuditedTransactions(network, AuditedTransactions[:5], nil, nil)
	CheckAuditedTransactions(network, AuditedTransactions[:5], &t0, &t7)
	CheckAuditedTransactions(network, AuditedTransactions[2:5], &t4, &t7)
	CheckAcceptedTransactions(network, "bob", "", BobAcceptedTransactions[:3], nil, nil, nil)
	CheckAcceptedTransactions(network, "bob", "", BobAcceptedTransactions[:3], &t4, &t7, nil)

	h = ListIssuerHistory(network, "", "USD")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(120))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = ListIssuerHistory(network, "", "EUR")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(30))).To(BeEquivalentTo(0))
	Expect(h.ByType("EUR").Count()).To(BeEquivalentTo(h.Count()))

	CheckBalanceAndHolding(network, "alice", "", "USD", 120)
	CheckBalanceAndHolding(network, "bob", "", "EUR", 30)

	network.StopFSCNode("alice")
	time.Sleep(3 * time.Second)
	network.StartFSCNode("alice")
	time.Sleep(5 * time.Second)

	t8 := time.Now()
	TransferCash(network, "alice", "", "USD", 111, "bob", auditor)
	t9 := time.Now()
	CheckAuditedTransactions(network, AuditedTransactions[5:7], &t8, &t9)
	CheckSpending(network, "alice", "", "USD", 111)
	CheckAcceptedTransactions(network, "bob", "", BobAcceptedTransactions[3:4], &t8, &t9, nil)
	CheckAcceptedTransactions(network, "bob", "", BobAcceptedTransactions[3:4], &t8, &t9, nil, ttxdb.Transfer)
	CheckAcceptedTransactions(network, "bob", "", BobAcceptedTransactions[3:4], &t8, &t9, []ttxdb.TxStatus{ttxdb.Confirmed}, ttxdb.Transfer)
	CheckAcceptedTransactions(network, "bob", "", BobAcceptedTransactions[:4], &t0, &t9, nil)
	CheckAcceptedTransactions(network, "bob", "", BobAcceptedTransactions[:4], nil, nil, nil)
	ut := ListUnspentTokens(network, "alice", "", "USD")
	Expect(ut.Count() > 0).To(BeTrue())
	Expect(ut.Sum(64).ToBigInt().Cmp(big.NewInt(9))).To(BeEquivalentTo(0))
	Expect(ut.ByType("USD").Count()).To(BeEquivalentTo(ut.Count()))
	ut = ListUnspentTokens(network, "bob", "", "USD")
	Expect(ut.Count() > 0).To(BeTrue())
	Expect(ut.Sum(64).ToBigInt().Cmp(big.NewInt(111))).To(BeEquivalentTo(0))
	Expect(ut.ByType("USD").Count()).To(BeEquivalentTo(ut.Count()))

	RedeemCash(network, "bob", "", "USD", 11, auditor)
	t10 := time.Now()
	CheckAcceptedTransactions(network, "bob", "", BobAcceptedTransactions[:6], nil, nil, nil)
	CheckAcceptedTransactions(network, "bob", "", BobAcceptedTransactions[5:6], nil, nil, nil, ttxdb.Redeem)
	CheckAcceptedTransactions(network, "bob", "", BobAcceptedTransactions[5:6], nil, nil, []ttxdb.TxStatus{ttxdb.Confirmed}, ttxdb.Redeem)
	CheckAuditedTransactions(network, AuditedTransactions[7:9], &t9, &t10)

	t11 := time.Now()
	IssueCash(network, "", "USD", 10, "bob", auditor, true)
	t12 := time.Now()
	CheckAuditedTransactions(network, AuditedTransactions[9:10], &t11, &t12)
	CheckAuditedTransactions(network, AuditedTransactions[:], &t0, &t12)
	CheckSpending(network, "bob", "", "USD", 11)

	IssueCash(network, "", "USD", 1, "alice", auditor, true)

	CheckBalanceAndHolding(network, "alice", "", "USD", 10)
	CheckBalanceAndHolding(network, "alice", "", "EUR", 0)
	CheckBalanceAndHolding(network, "bob", "", "EUR", 30)
	CheckBalanceAndHolding(network, "bob", "bob", "EUR", 30)
	CheckBalanceAndHolding(network, "bob", "", "USD", 110)

	SwapCash(network, "alice", "", "USD", 10, "EUR", 10, "bob", auditor)

	CheckBalanceAndHolding(network, "alice", "", "USD", 0)
	CheckBalanceAndHolding(network, "alice", "", "EUR", 10)
	CheckBalanceAndHolding(network, "bob", "", "EUR", 20)
	CheckBalanceAndHolding(network, "bob", "", "USD", 120)
	CheckSpending(network, "alice", "", "USD", 121)
	CheckSpending(network, "bob", "", "EUR", 10)

	RedeemCash(network, "bob", "", "USD", 10, auditor)
	CheckBalanceAndHolding(network, "bob", "", "USD", 110)
	CheckSpending(network, "bob", "", "USD", 21)

	// Check self endpoints
	IssueCash(network, "", "USD", 110, "issuer", auditor, true)
	IssueCash(network, "", "EUR", 150, "issuer", auditor, true)
	IssueCash(network, "issuer.id1", "EUR", 10, "issuer.owner", auditor, true)

	h = ListIssuerHistory(network, "", "USD")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(241))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = ListIssuerHistory(network, "", "EUR")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(180))).To(BeEquivalentTo(0))
	Expect(h.ByType("EUR").Count()).To(BeEquivalentTo(h.Count()))

	CheckBalanceAndHolding(network, "issuer", "", "USD", 110)
	CheckBalanceAndHolding(network, "issuer", "", "EUR", 150)
	CheckBalanceAndHolding(network, "issuer", "issuer.owner", "EUR", 10)

	// Restart the auditor
	network.StopFSCNode("auditor")
	time.Sleep(3 * time.Second)
	network.StartFSCNode("auditor")
	time.Sleep(5 * time.Second)
	RegisterAuditor(network, auditor)

	CheckBalanceAndHolding(network, "issuer", "", "USD", 110)
	CheckBalanceAndHolding(network, "issuer", "", "EUR", 150)
	CheckBalanceAndHolding(network, "issuer", "issuer.owner", "EUR", 10)

	// Check double spending
	txIDPine := IssueCash(network, "", "PINE", 55, "alice", auditor, true)
	tokenIDPine := &token2.ID{
		TxId:  txIDPine,
		Index: 0,
	}
	txID1, tx1 := PrepareTransferCash(network, "alice", "", "PINE", 55, "bob", auditor, tokenIDPine)
	CheckBalance(network, "alice", "", "PINE", 55)
	CheckHolding(network, "alice", "", "PINE", 0)
	CheckBalance(network, "bob", "", "PINE", 0)
	CheckHolding(network, "bob", "", "PINE", 55)
	txID2, tx2 := PrepareTransferCash(network, "alice", "", "PINE", 55, "bob", auditor, tokenIDPine)
	CheckBalance(network, "alice", "", "PINE", 55)
	CheckHolding(network, "alice", "", "PINE", -55)
	CheckBalance(network, "bob", "", "PINE", 0)
	CheckHolding(network, "bob", "", "PINE", 110)
	BroadcastPreparedTransferCash(network, "alice", tx1, true)
	Expect(network.Client("bob").IsTxFinal(txID1)).NotTo(HaveOccurred())
	Expect(network.Client("auditor").IsTxFinal(txID1)).NotTo(HaveOccurred())
	CheckBalance(network, "alice", "", "PINE", 0)
	CheckHolding(network, "alice", "", "PINE", -55)
	CheckBalance(network, "bob", "", "PINE", 55)
	CheckHolding(network, "bob", "", "PINE", 110)
	BroadcastPreparedTransferCash(network, "alice", tx2, true, "is not valid")
	Expect(network.Client("bob").IsTxFinal(txID2)).To(HaveOccurred())
	Expect(network.Client("auditor").IsTxFinal(txID2)).To(HaveOccurred())
	CheckBalanceAndHolding(network, "alice", "", "PINE", 0)
	CheckBalanceAndHolding(network, "bob", "", "PINE", 55)

	// Test Auditor ability to override transaction state
	txID3, tx3 := PrepareTransferCash(network, "bob", "", "PINE", 10, "alice", auditor, nil)
	CheckBalance(network, "alice", "", "PINE", 0)
	CheckHolding(network, "alice", "", "PINE", 10)
	CheckBalance(network, "bob", "", "PINE", 55)
	CheckHolding(network, "bob", "", "PINE", 45)
	SetTransactionAuditStatus(network, auditor, txID3, ttx.Deleted)
	CheckBalanceAndHolding(network, "alice", "", "PINE", 0)
	CheckBalanceAndHolding(network, "bob", "", "PINE", 55)
	TokenSelectorUnlock(network, "bob", txID3)
	FinalityWithTimeout(network, "bob", tx3, 20*time.Second)

	// Addition transfers

	TransferCash(network, "issuer", "", "USD", 50, "issuer", auditor)
	CheckBalanceAndHolding(network, "issuer", "", "USD", 110)
	CheckBalanceAndHolding(network, "issuer", "", "EUR", 150)

	TransferCash(network, "issuer", "", "USD", 50, "manager", auditor)
	TransferCash(network, "issuer", "", "EUR", 20, "manager", auditor)
	CheckBalanceAndHolding(network, "issuer", "", "USD", 60)
	CheckBalanceAndHolding(network, "issuer", "", "EUR", 130)
	CheckBalanceAndHolding(network, "manager", "", "USD", 50)
	CheckBalanceAndHolding(network, "manager", "", "EUR", 20)

	// Play with wallets
	TransferCash(network, "manager", "", "USD", 10, "manager.id1", auditor)
	TransferCash(network, "manager", "", "USD", 10, "manager.id2", auditor)
	TransferCash(network, "manager", "", "USD", 10, "manager.id3", auditor)
	CheckBalanceAndHolding(network, "manager", "", "USD", 20)
	CheckBalanceAndHolding(network, "manager", "manager.id1", "USD", 10)
	CheckBalanceAndHolding(network, "manager", "manager.id2", "USD", 10)
	CheckBalanceAndHolding(network, "manager", "manager.id3", "USD", 10)

	TransferCash(network, "manager", "manager.id1", "USD", 10, "manager.id2", auditor)
	CheckSpending(network, "manager", "manager.id1", "USD", 10)
	CheckBalanceAndHolding(network, "manager", "", "USD", 20)
	CheckBalanceAndHolding(network, "manager", "manager.id1", "USD", 0)
	CheckBalanceAndHolding(network, "manager", "manager.id2", "USD", 20)
	CheckBalanceAndHolding(network, "manager", "manager.id3", "USD", 10)

	// Swap among wallets
	TransferCash(network, "manager", "", "EUR", 10, "manager.id1", auditor)
	CheckBalanceAndHolding(network, "manager", "", "EUR", 10)
	CheckBalanceAndHolding(network, "manager", "manager.id1", "EUR", 10)

	SwapCash(network, "manager", "manager.id1", "EUR", 10, "USD", 10, "manager.id2", auditor)
	CheckBalanceAndHolding(network, "manager", "", "USD", 20)
	CheckBalanceAndHolding(network, "manager", "", "EUR", 10)
	CheckBalanceAndHolding(network, "manager", "manager.id1", "USD", 10)
	CheckBalanceAndHolding(network, "manager", "manager.id1", "EUR", 0)
	CheckBalanceAndHolding(network, "manager", "manager.id2", "USD", 10)
	CheckBalanceAndHolding(network, "manager", "manager.id2", "EUR", 10)
	CheckBalanceAndHolding(network, "manager", "manager.id3", "USD", 10)

	// no more USD can be issued, reached quota of 220
	IssueCashFail(network, "USD", 10, "alice")

	CheckBalanceAndHolding(network, "alice", "", "USD", 0)
	CheckBalanceAndHolding(network, "alice", "", "EUR", 10)

	// limits
	CheckBalanceAndHolding(network, "alice", "", "USD", 0)
	CheckBalanceAndHolding(network, "alice", "", "EUR", 10)
	CheckBalanceAndHolding(network, "bob", "", "EUR", 20)
	CheckBalanceAndHolding(network, "bob", "", "USD", 110)
	IssueCash(network, "", "EUR", 2200, "alice", auditor, true)
	IssueCash(network, "", "EUR", 2000, "charlie", auditor, true)
	CheckBalanceAndHolding(network, "alice", "", "EUR", 2210)
	CheckBalanceAndHolding(network, "charlie", "", "EUR", 2000)
	TransferCash(network, "alice", "", "EUR", 210, "bob", auditor, "payment limit reached", "alice", "[EUR][210]")
	CheckBalanceAndHolding(network, "bob", "", "USD", 110)
	CheckBalanceAndHolding(network, "bob", "", "EUR", 20)

	TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	CheckBalanceAndHolding(network, "bob", "", "EUR", 1820)
	CheckSpending(network, "alice", "", "EUR", 1800)
	TransferCash(network, "alice", "", "EUR", 200, "bob", auditor, "cumulative payment limit reached", "alice", "[EUR][2000]")
	TransferCash(network, "charlie", "", "EUR", 200, "bob", auditor)
	TransferCash(network, "charlie", "", "EUR", 200, "bob", auditor)
	TransferCash(network, "charlie", "", "EUR", 200, "bob", auditor)
	TransferCash(network, "charlie", "", "EUR", 200, "bob", auditor)
	TransferCash(network, "charlie", "", "EUR", 200, "bob", auditor)
	CheckBalanceAndHolding(network, "bob", "", "EUR", 2820)
	TransferCash(network, "charlie", "", "EUR", 200, "bob", auditor, "holding limit reached", "bob", "[EUR][3020]")
	CheckBalanceAndHolding(network, "bob", "", "EUR", 2820)

	// Routing
	IssueCash(network, "", "EUR", 10, "alice.id1", auditor, true)
	CheckAcceptedTransactions(network, "alice", "alice.id1", AliceID1AcceptedTransactions[:], nil, nil, nil)
	TransferCash(network, "alice", "alice.id1", "EUR", 10, "bob.id1", auditor)
	CheckBalanceAndHolding(network, "alice", "alice.id1", "EUR", 0)
	CheckBalanceAndHolding(network, "bob", "bob.id1", "EUR", 10)

	// Concurrent transfers
	transferErrors := make([]chan error, 5)
	var sum uint64
	for i := range transferErrors {
		transferErrors[i] = make(chan error, 1)

		transfer := transferErrors[i]
		r, err := rand.Int(rand.Reader, big.NewInt(200))
		Expect(err).ToNot(HaveOccurred())
		v := r.Uint64() + 1
		sum += v
		go func() {
			_, err := network.Client("bob").CallView("transferWithSelector", common.JSONMarshall(&views.Transfer{
				Auditor:   auditor,
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
	for _, transfer := range transferErrors {
		err := <-transfer
		Expect(err).ToNot(HaveOccurred())
	}
	CheckBalanceAndHolding(network, "bob", "", "EUR", 2820-sum)

	// Transfer With Selector
	IssueCash(network, "", "YUAN", 17, "alice", auditor, true)
	TransferCashWithSelector(network, "alice", "", "YUAN", 10, "bob", auditor)
	CheckBalanceAndHolding(network, "alice", "", "YUAN", 7)
	CheckBalanceAndHolding(network, "bob", "", "YUAN", 10)
	TransferCashWithSelector(network, "alice", "", "YUAN", 10, "bob", auditor, "pineapple", "insufficient funds")

	// Now, the tests asks Bob to transfer to Charlie 14 YUAN split in two parallel transactions each one transferring 7 YUAN.
	// Notice that Bob has only 10 YUAN, therefore bob will be able to assemble only one transfer.
	// We use two channels to collect the results of the two transfers.
	transferErrors = make([]chan error, 2)
	for i := range transferErrors {
		transferErrors[i] = make(chan error, 1)

		transferError := transferErrors[i]
		go func() {
			txid, err := network.Client("bob").CallView("transferWithSelector", common.JSONMarshall(&views.Transfer{
				Auditor:   auditor,
				Wallet:    "",
				Type:      "YUAN",
				Amount:    7,
				Recipient: network.Identity("charlie"),
				Retry:     false,
			}))
			if err != nil {
				// The transaction failed, we return the error to the caller.
				transferError <- err
				return
			}
			// The transaction didn't fail, let's wait for it to be confirmed, and return no error
			Expect(network.Client("charlie").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
			transferError <- nil
		}()
	}
	// collect the errors, and check that they are all nil, and one of them is the error we expect.
	var errors []error
	for _, transfer := range transferErrors {
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

	CheckBalanceAndHolding(network, "bob", "", "YUAN", 3)
	CheckBalanceAndHolding(network, "alice", "", "YUAN", 7)
	CheckBalanceAndHolding(network, "charlie", "", "YUAN", 7)

	// Transfer by IDs
	txID := IssueCash(network, "", "CHF", 17, "alice", auditor, true)
	TransferCashByIDs(network, "alice", "", []*token2.ID{{TxId: txID, Index: 0}}, 17, "bob", auditor, true, "test release")
	// the previous call should not keep the token locked if release is successful
	txID = TransferCashByIDs(network, "alice", "", []*token2.ID{{TxId: txID, Index: 0}}, 17, "bob", auditor, false)
	RedeemCashByIDs(network, "bob", "", []*token2.ID{{TxId: txID, Index: 0}}, 17, auditor)

	CheckPublicParamsViewFactory(network, "issuer", "auditor", "alice", "bob", "charlie", "manager")
}

func RegisterAuditor(network *integration.Infrastructure, id string) {
	_, err := network.Client(id).CallView("register", nil)
	Expect(err).NotTo(HaveOccurred())
}

func RegisterCertifier(network *integration.Infrastructure) {
	_, err := network.Client("certifier").CallView("register", nil)
	Expect(err).NotTo(HaveOccurred())
}

func IssueCash(network *integration.Infrastructure, wallet string, typ string, amount uint64, receiver string, auditor string, anonymous bool) string {
	if auditor == "issuer" {
		// the issuer is the auditor, choose default identity
		auditor = ""
	}
	txid, err := network.Client("issuer").CallView("issue", common.JSONMarshall(&views.IssueCash{
		Anonymous:    anonymous,
		Auditor:      auditor,
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

func CheckAuditedTransactions(network *integration.Infrastructure, expected []*ttxdb.TransactionRecord, start *time.Time, end *time.Time) {
	txsBoxed, err := network.Client("auditor").CallView("historyAuditing", common.JSONMarshall(&views.ListAuditedTransactions{
		From: start,
		To:   end,
	}))
	Expect(err).NotTo(HaveOccurred())
	var txs []*ttxdb.TransactionRecord
	common.JSONUnmarshal(txsBoxed.([]byte), &txs)
	Expect(len(txs)).To(Equal(len(expected)), "expected [%v] transactions, got [%v]", expected, txs)
	for i, tx := range txs {
		fmt.Printf("tx %d: %+v\n", i, tx)
		fmt.Printf("expected %d: %+v\n", i, expected[i])
		txExpected := expected[i]
		Expect(tx.TokenType).To(Equal(txExpected.TokenType), "tx [%d] expected token type [%v], got [%v]", i, txExpected.TokenType, tx.TokenType)
		Expect(strings.HasPrefix(tx.SenderEID, txExpected.SenderEID)).To(BeTrue(), "tx [%d] expected sender [%v], got [%v]", i, txExpected.SenderEID, tx.SenderEID)
		Expect(strings.HasPrefix(tx.RecipientEID, txExpected.RecipientEID)).To(BeTrue(), "tx [%d] tx.RecipientEID: %s, txExpected.RecipientEID: %s", i, tx.RecipientEID, txExpected.RecipientEID)
		Expect(tx.Status).To(Equal(txExpected.Status), "tx [%d] expected status [%v], got [%v]", i, txExpected.Status, tx.Status)
		Expect(tx.ActionType).To(Equal(txExpected.ActionType), "tx [%d] expected transaction type [%v], got [%v]", i, txExpected.ActionType, tx.ActionType)
		Expect(tx.Amount).To(Equal(txExpected.Amount), "tx [%d] expected amount [%v], got [%v]", i, txExpected.Amount, tx.Amount)
	}
}

func CheckAcceptedTransactions(network *integration.Infrastructure, id string, wallet string, expected []*ttxdb.TransactionRecord, start *time.Time, end *time.Time, statuses []driver.TxStatus, actionTypes ...ttxdb.ActionType) {
	eIDBoxed, err := network.Client(id).CallView("GetEnrollmentID", common.JSONMarshall(&views.GetEnrollmentID{
		Wallet: wallet,
	}))
	Expect(err).NotTo(HaveOccurred())
	eID := common.JSONUnmarshalString(eIDBoxed)

	txsBoxed, err := network.Client(id).CallView("acceptedTransactionHistory", common.JSONMarshall(&views.ListAcceptedTransactions{
		SenderWallet:    eID,
		RecipientWallet: eID,
		From:            start,
		To:              end,
		ActionTypes:     actionTypes,
		Statuses:        statuses,
	}))
	Expect(err).NotTo(HaveOccurred())
	var txs []*ttxdb.TransactionRecord
	common.JSONUnmarshal(txsBoxed.([]byte), &txs)
	Expect(len(txs)).To(Equal(len(expected)), "expected [%v] transactions, got [%v]", expected, txs)
	for i, tx := range txs {
		fmt.Printf("tx %d: %+v\n", i, tx)
		fmt.Printf("expected %d: %+v\n", i, expected[i])
		txExpected := expected[i]
		Expect(tx.TokenType).To(Equal(txExpected.TokenType), "tx [%d] tx.TokenType: %s, txExpected.TokenType: %s", i, tx.TokenType, txExpected.TokenType)
		Expect(strings.HasPrefix(tx.SenderEID, txExpected.SenderEID)).To(BeTrue(), "tx [%d] tx.SenderEID: %s, txExpected.SenderEID: %s", i, tx.SenderEID, txExpected.SenderEID)
		Expect(strings.HasPrefix(tx.RecipientEID, txExpected.RecipientEID)).To(BeTrue(), "tx [%d] tx.RecipientEID: %s, txExpected.RecipientEID: %s", i, tx.RecipientEID, txExpected.RecipientEID)
		Expect(tx.Status).To(Equal(txExpected.Status), "tx [%d] tx.Status: %s, txExpected.Status: %s", i, tx.Status, txExpected.Status)
		Expect(tx.ActionType).To(Equal(txExpected.ActionType), "tx [%d] tx.ActionType: %s, txExpected.ActionType: %s", i, tx.ActionType, txExpected.ActionType)
		Expect(tx.Amount).To(Equal(txExpected.Amount), "tx [%d] tx.Amount: %d, txExpected.Amount: %d", i, tx.Amount, txExpected.Amount)
	}
}

func CheckBalanceAndHolding(network *integration.Infrastructure, id string, wallet string, typ string, expected uint64) {
	CheckBalance(network, id, wallet, typ, expected)
	CheckHolding(network, id, wallet, typ, int64(expected))
}

func CheckBalance(network *integration.Infrastructure, id string, wallet string, typ string, expected uint64) {
	// check balance
	b, err := query.NewClient(network.Client(id)).WalletBalance(wallet, typ)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(b)).To(BeEquivalentTo(1))
	Expect(b[0].Type).To(BeEquivalentTo(typ))
	q, err := token2.ToQuantity(b[0].Quantity, 64)
	Expect(err).NotTo(HaveOccurred())
	expectedQ := token2.NewQuantityFromUInt64(expected)
	Expect(expectedQ.Cmp(q)).To(BeEquivalentTo(0), "[%s]!=[%s]", expected, q)
}

func CheckHolding(network *integration.Infrastructure, id string, wallet string, typ string, expected int64) {
	eIDBoxed, err := network.Client(id).CallView("GetEnrollmentID", common.JSONMarshall(&views.GetEnrollmentID{
		Wallet: wallet,
	}))
	Expect(err).NotTo(HaveOccurred())
	eID := common.JSONUnmarshalString(eIDBoxed)
	holdingBoxed, err := network.Client("auditor").CallView("holding", common.JSONMarshall(&views.CurrentHolding{
		EnrollmentID: eID,
		TokenType:    typ,
	}))
	Expect(err).NotTo(HaveOccurred())
	holding, err := strconv.Atoi(common.JSONUnmarshalString(holdingBoxed))
	Expect(err).NotTo(HaveOccurred())
	Expect(holding).To(Equal(int(expected)))
}

func CheckSpending(network *integration.Infrastructure, id string, wallet string, tokenType string, expected uint64) {
	// check spending
	// first get the enrollment id
	eIDBoxed, err := network.Client(id).CallView("GetEnrollmentID", common.JSONMarshall(&views.GetEnrollmentID{
		Wallet: wallet,
	}))
	Expect(err).NotTo(HaveOccurred())
	eID := common.JSONUnmarshalString(eIDBoxed)
	spendingBoxed, err := network.Client("auditor").CallView("spending", common.JSONMarshall(&views.CurrentSpending{
		EnrollmentID: eID,
		TokenType:    tokenType,
	}))
	Expect(err).NotTo(HaveOccurred())
	spending, err := strconv.ParseUint(common.JSONUnmarshalString(spendingBoxed), 10, 64)
	Expect(err).NotTo(HaveOccurred())
	Expect(spending).To(Equal(expected))
}

func ListIssuerHistory(network *integration.Infrastructure, wallet string, typ string) *token2.IssuedTokens {
	res, err := network.Client("issuer").CallView("historyIssuedToken", common.JSONMarshall(&views.ListIssuedTokens{
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

func TransferCash(network *integration.Infrastructure, id string, wallet string, typ string, amount uint64, receiver string, auditor string, expectedErrorMsgs ...string) {
	txidBoxed, err := network.Client(id).CallView("transfer", common.JSONMarshall(&views.Transfer{
		Auditor:   auditor,
		Wallet:    wallet,
		Type:      typ,
		Amount:    amount,
		Recipient: network.Identity(receiver),
	}))
	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Client(receiver).IsTxFinal(common.JSONUnmarshalString(txidBoxed))).NotTo(HaveOccurred())
		Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txidBoxed))).NotTo(HaveOccurred())

		signers := []string{auditor}
		if !strings.HasPrefix(receiver, id) {
			signers = append(signers, strings.Split(receiver, ".")[0])
		}
		txInfo := GetTransactionInfo(network, id, common.JSONUnmarshalString(txidBoxed))
		for _, identity := range signers {
			sigma, ok := txInfo.EndorsementAcks[network.Identity(identity).UniqueID()]
			Expect(ok).To(BeTrue(), "identity %s not found in txInfo.EndorsementAcks", identity)
			Expect(sigma).ToNot(BeNil(), "endorsement ack sigma is nil for identity %s", identity)
		}
		Expect(len(txInfo.EndorsementAcks)).To(BeEquivalentTo(len(signers)))
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range expectedErrorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg))
		}
		time.Sleep(5 * time.Second)
	}
}

func PrepareTransferCash(network *integration.Infrastructure, id string, wallet string, typ string, amount uint64, receiver string, auditor string, tokenID *token2.ID, expectedErrorMsgs ...string) (string, []byte) {
	transferInput := &views.Transfer{
		Auditor:   auditor,
		Wallet:    wallet,
		Type:      typ,
		Amount:    amount,
		Recipient: network.Identity(receiver),
	}
	if tokenID != nil {
		transferInput.TokenIDs = []*token2.ID{tokenID}
	}
	txBoxed, err := network.Client(id).CallView("prepareTransfer", common.JSONMarshall(transferInput))
	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range expectedErrorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg))
		}
		time.Sleep(5 * time.Second)
	}
	res := &views.PrepareTransferResult{}
	common.JSONUnmarshal(txBoxed.([]byte), res)
	return res.TxID, res.TXRaw
}

func SetTransactionAuditStatus(network *integration.Infrastructure, id string, txID string, txStatus ttx.TxStatus) {
	_, err := network.Client(id).CallView("SetTransactionAuditStatus", common.JSONMarshall(views.SetTransactionAuditStatus{
		TxID:   txID,
		Status: txStatus,
	}))
	Expect(err).NotTo(HaveOccurred())
}

func TokenSelectorUnlock(network *integration.Infrastructure, id string, txID string) {
	_, err := network.Client(id).CallView("TokenSelectorUnlock", common.JSONMarshall(views.TokenSelectorUnlock{
		TxID: txID,
	}))
	Expect(err).NotTo(HaveOccurred())
}

func BroadcastPreparedTransferCash(network *integration.Infrastructure, id string, tx []byte, finality bool, expectedErrorMsgs ...string) {
	_, err := network.Client(id).CallView("broadcastPreparedTransfer", common.JSONMarshall(&views.BroadcastPreparedTransfer{
		Tx:       tx,
		Finality: finality,
	}))
	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	Expect(err).To(HaveOccurred())
	fmt.Println("Failed to broadcast ", err)
	for _, msg := range expectedErrorMsgs {
		Expect(err.Error()).To(ContainSubstring(msg))
	}
	time.Sleep(5 * time.Second)
}

func FinalityWithTimeout(network *integration.Infrastructure, id string, tx []byte, timeout time.Duration) {
	elapsedBoxed, err := network.Client(id).CallView("FinalityWithTimeout", common.JSONMarshall(&views.FinalityWithTimeout{
		Tx:      tx,
		Timeout: timeout,
	}))
	Expect(err).NotTo(HaveOccurred())

	elapsed := JSONUnmarshalFloat64(elapsedBoxed)
	Expect(elapsed > timeout.Seconds()).To(BeTrue())
	Expect(elapsed < timeout.Seconds()*2).To(BeTrue())
}

func GetTransactionInfo(network *integration.Infrastructure, id string, txnId string) *ttx.TransactionInfo {
	boxed, err := network.Client(id).CallView("transactionInfo", common.JSONMarshall(&views.TransactionInfo{
		TransactionID: txnId,
	}))
	Expect(err).NotTo(HaveOccurred())
	info := &ttx.TransactionInfo{}
	common.JSONUnmarshal(boxed.([]byte), info)
	return info
}

func TransferCashByIDs(network *integration.Infrastructure, id string, wallet string, ids []*token2.ID, amount uint64, receiver string, auditor string, failToRelease bool, expectedErrorMsgs ...string) string {
	txid, err := network.Client(id).CallView("transfer", common.JSONMarshall(&views.Transfer{
		Auditor:       auditor,
		Wallet:        wallet,
		Type:          "",
		TokenIDs:      ids,
		Amount:        amount,
		Recipient:     network.Identity(receiver),
		FailToRelease: failToRelease,
	}))
	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Client(receiver).IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
		Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
		return common.JSONUnmarshalString(txid)
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range expectedErrorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg))
		}
		time.Sleep(5 * time.Second)
		return ""
	}
}

func TransferCashWithSelector(network *integration.Infrastructure, id string, wallet string, typ string, amount uint64, receiver string, auditor string, expectedErrorMsgs ...string) {
	txid, err := network.Client(id).CallView("transferWithSelector", common.JSONMarshall(&views.Transfer{
		Auditor:   auditor,
		Wallet:    wallet,
		Type:      typ,
		Amount:    amount,
		Recipient: network.Identity(receiver),
	}))
	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Client(receiver).IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
		Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range expectedErrorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg))
		}
		time.Sleep(5 * time.Second)
	}
}

func RedeemCash(network *integration.Infrastructure, id string, wallet string, typ string, amount uint64, auditor string) {
	txid, err := network.Client(id).CallView("redeem", common.JSONMarshall(&views.Redeem{
		Auditor: auditor,
		Wallet:  wallet,
		Type:    typ,
		Amount:  amount,
	}))
	Expect(err).NotTo(HaveOccurred())
	Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
}

func RedeemCashByIDs(network *integration.Infrastructure, id string, wallet string, ids []*token2.ID, amount uint64, auditor string) {
	txid, err := network.Client(id).CallView("redeem", common.JSONMarshall(&views.Redeem{
		Auditor:  auditor,
		Wallet:   wallet,
		Type:     "",
		TokenIDs: ids,
		Amount:   amount,
	}))
	Expect(err).NotTo(HaveOccurred())
	Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
}

func SwapCash(network *integration.Infrastructure, id string, wallet string, typeLeft string, amountLeft uint64, typRight string, amountRight uint64, receiver string, auditor string) {
	txid, err := network.Client(id).CallView("swap", common.JSONMarshall(&views.Swap{
		Auditor:         auditor,
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

func CheckPublicParamsViewFactory(network *integration.Infrastructure, ids ...string) {
	for _, id := range ids {
		_, err := network.Client(id).CallView("CheckPublicParamsMatch", nil)
		Expect(err).NotTo(HaveOccurred())
	}
}

func JSONUnmarshalFloat64(v interface{}) float64 {
	var s float64
	switch v := v.(type) {
	case []byte:
		err := json.Unmarshal(v, &s)
		Expect(err).NotTo(HaveOccurred())
	case string:
		err := json.Unmarshal([]byte(v), &s)
		Expect(err).NotTo(HaveOccurred())
	default:
		panic(fmt.Sprintf("type not recognized [%T]", v))
	}
	return s
}
