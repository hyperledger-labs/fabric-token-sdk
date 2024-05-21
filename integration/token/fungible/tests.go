/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fungible

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/gomega"
)

const (
	DLogNamespace     = "dlog-token-chaincode"
	FabTokenNamespace = "fabtoken-token-chaincode"
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
	{
		TxID:         "",
		ActionType:   ttxdb.Issue,
		SenderEID:    "",
		RecipientEID: "alice",
		TokenType:    "LIRA",
		Amount:       big.NewInt(3),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Issue,
		SenderEID:    "",
		RecipientEID: "alice",
		TokenType:    "LIRA",
		Amount:       big.NewInt(3),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Transfer,
		SenderEID:    "alice",
		RecipientEID: "bob",
		TokenType:    "LIRA",
		Amount:       big.NewInt(2),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Transfer,
		SenderEID:    "alice",
		RecipientEID: "alice",
		TokenType:    "LIRA",
		Amount:       big.NewInt(1),
		Status:       ttxdb.Confirmed,
	},
	{
		TxID:         "",
		ActionType:   ttxdb.Transfer,
		SenderEID:    "alice",
		RecipientEID: "charlie",
		TokenType:    "LIRA",
		Amount:       big.NewInt(3),
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

type OnAuditorRestartFunc = func(*integration.Infrastructure, string)

func TestAll(network *integration.Infrastructure, auditor string, aries bool, sel *token3.ReplicaSelector) {
	testInit(network, auditor, sel)

	t0 := time.Now()
	Eventually(DoesWalletExist).WithArguments(network, "issuer", "", views.IssuerWallet).WithTimeout(1 * time.Minute).WithPolling(15 * time.Second).Should(Equal(true))
	Eventually(DoesWalletExist).WithArguments(network, "issuer", "pineapple", views.IssuerWallet).WithTimeout(1 * time.Minute).WithPolling(15 * time.Second).Should(Equal(false))
	Eventually(DoesWalletExist).WithArguments(network, "alice", "", views.OwnerWallet).WithTimeout(1 * time.Minute).WithPolling(15 * time.Second).Should(Equal(true))
	Eventually(DoesWalletExist).WithArguments(network, "alice", "mango", views.OwnerWallet).WithTimeout(1 * time.Minute).WithPolling(15 * time.Second).Should(Equal(false))
	IssueCash(network, "", "USD", 110, "alice", auditor, true, "issuer")
	t1 := time.Now()
	CheckBalanceAndHolding(network, "alice", "", "USD", 110, auditor)
	CheckAuditedTransactions(network, sel.Get(auditor), AuditedTransactions[:1], nil, nil)
	CheckAuditedTransactions(network, sel.Get(auditor), AuditedTransactions[:1], &t0, &t1)
	CheckAcceptedTransactions(network, sel.Get("alice"), "", AliceAcceptedTransactions[:1], nil, nil, nil, ttxdb.Issue)
	CheckAcceptedTransactions(network, sel.Get("alice"), "", AliceAcceptedTransactions[:1], nil, nil, []ttxdb.TxStatus{ttxdb.Confirmed}, ttxdb.Issue)
	CheckAcceptedTransactions(network, sel.Get("alice"), "", AliceAcceptedTransactions[:1], nil, nil, nil)
	CheckAcceptedTransactions(network, sel.Get("alice"), "", AliceAcceptedTransactions[:1], &t0, &t1, nil)

	t2 := time.Now()
	Withdraw(network, nil, "alice", "", "USD", 10, auditor, "issuer")
	t3 := time.Now()
	CheckBalanceAndHolding(network, sel.Get("alice"), "", "USD", 120, auditor)
	CheckBalanceAndHolding(network, sel.Get("alice"), "alice", "USD", 120, auditor)
	CheckAuditedTransactions(network, auditor, AuditedTransactions[:2], nil, nil)
	CheckAuditedTransactions(network, auditor, AuditedTransactions[:2], &t0, &t3)
	CheckAuditedTransactions(network, auditor, AuditedTransactions[1:2], &t2, &t3)
	CheckAcceptedTransactions(network, sel.Get("alice"), "", AliceAcceptedTransactions[:2], nil, nil, nil)
	CheckAcceptedTransactions(network, sel.Get("alice"), "", AliceAcceptedTransactions[:2], &t0, &t3, nil)
	CheckAcceptedTransactions(network, sel.Get("alice"), "", AliceAcceptedTransactions[1:2], &t2, &t3, nil)

	h := ListIssuerHistory(network, "", "USD", "issuer")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(120))).To(BeEquivalentTo(0), "expected [%d]=[120]", h.Sum(64).ToBigInt().Int64())
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = ListIssuerHistory(network, "", "EUR", "issuer")
	Expect(h.Count()).To(BeEquivalentTo(0))

	Restart(network, true, auditor)
	RegisterAuditor(network, auditor)

	CheckBalanceAndHolding(network, sel.Get("alice"), "", "USD", 120, auditor)
	CheckBalanceAndHolding(network, sel.Get("alice"), "alice", "USD", 120, auditor)
	CheckAuditedTransactions(network, sel.Get(auditor), AuditedTransactions[:2], nil, nil)
	CheckAuditedTransactions(network, sel.Get(auditor), AuditedTransactions[:2], &t0, &t3)
	CheckAuditedTransactions(network, sel.Get(auditor), AuditedTransactions[1:2], &t2, &t3)
	CheckAcceptedTransactions(network, sel.Get("alice"), "", AliceAcceptedTransactions[:2], nil, nil, nil)
	CheckAcceptedTransactions(network, sel.Get("alice"), "", AliceAcceptedTransactions[:2], &t0, &t3, nil)
	CheckAcceptedTransactions(network, sel.Get("alice"), "", AliceAcceptedTransactions[1:2], &t2, &t3, nil)

	h = ListIssuerHistory(network, "", "USD", sel.Get("issuer"))
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(120))).To(BeEquivalentTo(0), "expected [%d]=[120]", h.Sum(64).ToBigInt().Int64())
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = ListIssuerHistory(network, "", "EUR", sel.Get("issuer"))
	Expect(h.Count()).To(BeEquivalentTo(0))

	// Register a new issuer wallet and issue with that wallet
	tokenPlatform := token.GetPlatform(network.Ctx, "token")
	Expect(tokenPlatform).ToNot(BeNil(), "cannot find token platform in context")
	Expect(tokenPlatform.GetTopology()).ToNot(BeNil(), "invalid token topology, it is nil")
	Expect(len(tokenPlatform.GetTopology().TMSs)).ToNot(BeEquivalentTo(0), "no tms defined in token topology")
	// Gen crypto material for the new issuer wallet
	newIssuerWalletPath := tokenPlatform.GenIssuerCryptoMaterial(tokenPlatform.GetTopology().TMSs[0].BackendTopology.Name(), "issuer", "issuer.ExtraId")
	// Register it
	RegisterIssuerIdentity(network, "issuer", "newIssuerWallet", newIssuerWalletPath)
	// Issuer tokens with this new wallet
	t4 := time.Now()
	IssueCash(network, "newIssuerWallet", "EUR", 10, "bob", auditor, false, "issuer")
	//t5 := time.Now()
	CheckBalanceAndHolding(network, "bob", "", "EUR", 10, auditor)
	IssueCash(network, "newIssuerWallet", "EUR", 10, "bob", auditor, true, "issuer")
	//t6 := time.Now()
	CheckBalanceAndHolding(network, "bob", "", "EUR", 20, auditor)
	IssueCash(network, "newIssuerWallet", "EUR", 10, "bob", auditor, false, "issuer")
	t7 := time.Now()
	CheckBalanceAndHolding(network, "bob", "", "EUR", 30, auditor)
	CheckAuditedTransactions(network, sel.Get(auditor), AuditedTransactions[:5], nil, nil)
	CheckAuditedTransactions(network, sel.Get(auditor), AuditedTransactions[:5], &t0, &t7)
	CheckAuditedTransactions(network, sel.Get(auditor), AuditedTransactions[2:5], &t4, &t7)
	CheckAcceptedTransactions(network, sel.Get("bob"), "", BobAcceptedTransactions[:3], nil, nil, nil)
	CheckAcceptedTransactions(network, sel.Get("bob"), "", BobAcceptedTransactions[:3], &t4, &t7, nil)

	h = ListIssuerHistory(network, "", "USD", sel.Get("issuer"))
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(120))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = ListIssuerHistory(network, "newIssuerWallet", "EUR", sel.Get("issuer"))
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(30))).To(BeEquivalentTo(0))
	Expect(h.ByType("EUR").Count()).To(BeEquivalentTo(h.Count()))

	CheckBalanceAndHolding(network, sel.Get("alice"), "", "USD", 120, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "EUR", 30, auditor)

	Restart(network, false, "alice")

	t8 := time.Now()
	TransferCash(network, sel.Get("alice"), "", "USD", 111, "bob", auditor)
	t9 := time.Now()
	CheckAuditedTransactions(network, auditor, AuditedTransactions[5:7], &t8, &t9)
	CheckSpending(network, sel.Get("alice"), "", "USD", auditor, 111)
	CheckAcceptedTransactions(network, sel.Get("bob"), "", BobAcceptedTransactions[3:4], &t8, &t9, nil)
	CheckAcceptedTransactions(network, sel.Get("bob"), "", BobAcceptedTransactions[3:4], &t8, &t9, nil, ttxdb.Transfer)
	CheckAcceptedTransactions(network, sel.Get("bob"), "", BobAcceptedTransactions[3:4], &t8, &t9, []ttxdb.TxStatus{ttxdb.Confirmed}, ttxdb.Transfer)
	CheckAcceptedTransactions(network, sel.Get("bob"), "", BobAcceptedTransactions[:4], &t0, &t9, nil)
	CheckAcceptedTransactions(network, sel.Get("bob"), "", BobAcceptedTransactions[:4], nil, nil, nil)
	ut := ListUnspentTokens(network, sel.Get("alice"), "", "USD")
	Expect(ut.Count() > 0).To(BeTrue())
	Expect(ut.Sum(64).ToBigInt().Cmp(big.NewInt(9))).To(BeEquivalentTo(0), "got [%d], expected 9", ut.Sum(64).ToBigInt())
	Expect(ut.ByType("USD").Count()).To(BeEquivalentTo(ut.Count()))
	ut = ListUnspentTokens(network, sel.Get("bob"), "", "USD")
	Expect(ut.Count() > 0).To(BeTrue())
	Expect(ut.Sum(64).ToBigInt().Cmp(big.NewInt(111))).To(BeEquivalentTo(0), "got [%d], expected 111", ut.Sum(64).ToBigInt())
	Expect(ut.ByType("USD").Count()).To(BeEquivalentTo(ut.Count()))

	RedeemCash(network, sel.Get("bob"), "", "USD", 11, auditor)
	t10 := time.Now()
	CheckAcceptedTransactions(network, sel.Get("bob"), "", BobAcceptedTransactions[:6], nil, nil, nil)
	CheckAcceptedTransactions(network, sel.Get("bob"), "", BobAcceptedTransactions[5:6], nil, nil, nil, ttxdb.Redeem)
	CheckAcceptedTransactions(network, sel.Get("bob"), "", BobAcceptedTransactions[5:6], nil, nil, []ttxdb.TxStatus{ttxdb.Confirmed}, ttxdb.Redeem)
	CheckAuditedTransactions(network, sel.Get(auditor), AuditedTransactions[7:9], &t9, &t10)

	t11 := time.Now()
	IssueCash(network, "", "USD", 10, "bob", auditor, true, "issuer")
	t12 := time.Now()
	CheckAuditedTransactions(network, sel.Get(auditor), AuditedTransactions[9:10], &t11, &t12)
	CheckAuditedTransactions(network, sel.Get(auditor), AuditedTransactions[:10], &t0, &t12)
	CheckSpending(network, sel.Get("bob"), "", "USD", auditor, 11)

	// test multi action transfer...
	t13 := time.Now()
	IssueCash(network, "", "LIRA", 3, "alice", auditor, true, "issuer")
	IssueCash(network, "", "LIRA", 3, "alice", auditor, true, "issuer")
	t14 := time.Now()
	CheckAuditedTransactions(network, auditor, AuditedTransactions[10:12], &t13, &t14)
	// perform the normal transaction
	txLiraTransfer := TransferCashMultiActions(network, "alice", "", "LIRA", []uint64{2, 3}, []string{"bob", "charlie"}, auditor, nil)
	t16 := time.Now()
	AuditedTransactions[12].TxID = txLiraTransfer
	AuditedTransactions[13].TxID = txLiraTransfer
	AuditedTransactions[14].TxID = txLiraTransfer
	CheckBalanceAndHolding(network, sel.Get("alice"), "", "LIRA", 1, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "LIRA", 2, auditor)
	CheckBalanceAndHolding(network, sel.Get("charlie"), "", "LIRA", 3, auditor)
	CheckAuditedTransactions(network, sel.Get(auditor), AuditedTransactions[:], &t0, &t16)
	CheckOwnerDB(network, nil, sel.All("issuer", "alice", "bob", "charlie", "manager")...)

	IssueCash(network, "", "USD", 1, "alice", auditor, true, "issuer")

	testTwoGeneratedOwnerWalletsSameNode(network, auditor, !aries)

	CheckBalanceAndHolding(network, sel.Get("alice"), "", "USD", 10, auditor)
	CheckBalanceAndHolding(network, sel.Get("alice"), "", "EUR", 0, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "EUR", 30, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "bob", "EUR", 30, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "USD", 110, auditor)

	SwapCash(network, "alice", "", "USD", 10, "EUR", 10, "bob", auditor)

	CheckBalanceAndHolding(network, sel.Get("alice"), "", "USD", 0, auditor)
	CheckBalanceAndHolding(network, sel.Get("alice"), "", "EUR", 10, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "EUR", 20, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "USD", 120, auditor)
	CheckSpending(network, sel.Get("alice"), "", "USD", auditor, 121)
	CheckSpending(network, sel.Get("bob"), "", "EUR", auditor, 10)

	RedeemCash(network, sel.Get("bob"), "", "USD", 10, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "USD", 110, auditor)
	CheckSpending(network, sel.Get("bob"), "", "USD", auditor, 21)

	// Check self endpoints
	IssueCash(network, "", "USD", 110, "issuer", auditor, true, "issuer")
	IssueCash(network, "newIssuerWallet", "EUR", 150, "issuer", auditor, true, "issuer")
	IssueCash(network, "issuer.id1", "EUR", 10, "issuer.owner", auditor, true, "issuer")

	h = ListIssuerHistory(network, "", "USD", sel.Get("issuer"))
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(241))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = ListIssuerHistory(network, "newIssuerWallet", "EUR", sel.Get("issuer"))
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(180))).To(BeEquivalentTo(0))
	Expect(h.ByType("EUR").Count()).To(BeEquivalentTo(h.Count()))

	CheckBalanceAndHolding(network, sel.Get("issuer"), "", "USD", 110, auditor)
	CheckBalanceAndHolding(network, sel.Get("issuer"), "", "EUR", 150, auditor)
	CheckBalanceAndHolding(network, sel.Get("issuer"), "issuer.owner", "EUR", 10, auditor)

	// Restart the auditor
	Restart(network, true, auditor)
	RegisterAuditor(network, auditor)

	CheckBalanceAndHolding(network, sel.Get("issuer"), "", "USD", 110, auditor)
	CheckBalanceAndHolding(network, sel.Get("issuer"), "", "EUR", 150, auditor)
	CheckBalanceAndHolding(network, sel.Get("issuer"), "issuer.owner", "EUR", 10, auditor)

	CheckOwnerDB(network, nil, sel.All("issuer", "alice", "bob", "charlie", "manager")...)
	CheckAuditorDB(network, sel.Get(auditor), "", nil)

	// Check double spending
	txIDPine := IssueCash(network, "", "PINE", 55, "alice", auditor, true, "issuer")
	tokenIDPine := &token2.ID{
		TxId:  txIDPine,
		Index: 0,
	}
	txID1, tx1 := PrepareTransferCash(network, sel.Get("alice"), "", "PINE", 55, "bob", auditor, tokenIDPine)
	CheckBalance(network, sel.Get("alice"), "", "PINE", 55)
	CheckHolding(network, sel.Get("alice"), "", "PINE", 0, auditor)
	CheckBalance(network, "bob", "", "PINE", 0)
	CheckHolding(network, "bob", "", "PINE", 55, auditor)
	txID2, tx2 := PrepareTransferCash(network, "alice", "", "PINE", 55, "bob", auditor, tokenIDPine)
	CheckBalance(network, sel.Get("alice"), "", "PINE", 55)
	CheckHolding(network, sel.Get("alice"), "", "PINE", -55, auditor)
	CheckBalance(network, "bob", "", "PINE", 0)
	CheckHolding(network, "bob", "", "PINE", 110, auditor)
	CheckBalanceAndHolding(network, "bob", "", "EUR", 20, auditor)
	CheckBalanceAndHolding(network, "bob", "", "USD", 110, auditor)
	CheckOwnerDB(network, []string{
		fmt.Sprintf("transaction record [%s] is unknown for vault but not for the db [Pending]", txID1),
		fmt.Sprintf("transaction record [%s] is unknown for vault but not for the db [Pending]", txID2),
	}, "bob")
	fmt.Printf("prepared transactions [%s:%s]", txID1, txID2)
	Restart(network, true, "bob")
	Restart(network, false, auditor)
	RegisterAuditor(network, auditor)
	CheckBalance(network, "bob", "", "PINE", 0)
	CheckHolding(network, "bob", "", "PINE", 110, auditor)
	CheckBalanceAndHolding(network, "bob", "", "EUR", 20, auditor)
	CheckBalanceAndHolding(network, "bob", "", "USD", 110, auditor)
	BroadcastPreparedTransferCash(network, "alice", txID1, tx1, true)
	common2.CheckFinality(network, "bob", txID1, nil, false)
	common2.CheckFinality(network, auditor, txID1, nil, false)
	CheckBalance(network, sel.Get("alice"), "", "PINE", 0)
	CheckHolding(network, sel.Get("alice"), "", "PINE", -55, auditor)
	CheckBalance(network, sel.Get("bob"), "", "PINE", 55)
	CheckHolding(network, sel.Get("bob"), "", "PINE", 110, auditor)
	BroadcastPreparedTransferCash(network, sel.Get("alice"), txID2, tx2, true, "is not valid")
	common2.CheckFinality(network, sel.Get("bob"), txID2, nil, true)
	common2.CheckFinality(network, sel.Get(auditor), txID2, nil, true)
	CheckBalanceAndHolding(network, sel.Get("alice"), "", "PINE", 0, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "PINE", 55, auditor)
	CheckOwnerDB(network, nil, sel.All("issuer", "alice", "bob", "charlie", "manager")...)
	CheckAuditorDB(network, sel.Get(auditor), "", nil)

	// Test Auditor ability to override transaction state
	txID3, tx3 := PrepareTransferCash(network, sel.Get("bob"), "", "PINE", 10, "alice", auditor, nil)
	CheckBalance(network, sel.Get("alice"), "", "PINE", 0)
	CheckHolding(network, sel.Get("alice"), "", "PINE", 10, auditor)
	CheckBalance(network, sel.Get("bob"), "", "PINE", 55)
	CheckHolding(network, sel.Get("bob"), "", "PINE", 45, auditor)
	SetTransactionAuditStatus(network, sel.Get(auditor), txID3, ttx.Deleted)
	CheckBalanceAndHolding(network, sel.Get("alice"), "", "PINE", 0, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "PINE", 55, auditor)
	TokenSelectorUnlock(network, sel.Get("bob"), txID3)
	FinalityWithTimeout(network, sel.Get("bob"), tx3, 20*time.Second)
	SetTransactionOwnersStatus(network, txID3, ttx.Deleted, sel.All("alice", "bob")...)

	// Restart
	CheckOwnerDB(network, nil, sel.All("bob", "alice")...)
	CheckOwnerDB(network, nil, sel.All("issuer", "charlie", "manager")...)
	CheckAuditorDB(network, auditor, "", nil)
	Restart(network, false, sel.All("alice", "bob", "charlie", "manager")...)
	CheckOwnerDB(network, nil, sel.All("bob", "alice")...)
	CheckOwnerDB(network, nil, sel.All("issuer", "charlie", "manager")...)
	CheckAuditorDB(network, sel.Get(auditor), "", nil)

	// Addition transfers
	TransferCash(network, "issuer", "", "USD", 50, "issuer", auditor)
	CheckBalanceAndHolding(network, sel.Get("issuer"), "", "USD", 110, auditor)
	CheckBalanceAndHolding(network, sel.Get("issuer"), "", "EUR", 150, auditor)

	TransferCash(network, "issuer", "", "USD", 50, "manager", auditor)
	TransferCash(network, "issuer", "", "EUR", 20, "manager", auditor)
	CheckBalanceAndHolding(network, sel.Get("issuer"), "", "USD", 60, auditor)
	CheckBalanceAndHolding(network, sel.Get("issuer"), "", "EUR", 130, auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "", "USD", 50, auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "", "EUR", 20, auditor)

	// Play with wallets
	TransferCash(network, "manager", "", "USD", 10, "manager.id1", auditor)
	TransferCash(network, "manager", "", "USD", 10, "manager.id2", auditor)
	TransferCash(network, "manager", "", "USD", 10, "manager.id3", auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "", "USD", 20, auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "manager.id1", "USD", 10, auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "manager.id2", "USD", 10, auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "manager.id3", "USD", 10, auditor)

	TransferCash(network, sel.Get("manager"), "manager.id1", "USD", 10, "manager.id2", auditor)
	CheckSpending(network, sel.Get("manager"), "manager.id1", "USD", auditor, 10)
	CheckBalanceAndHolding(network, sel.Get("manager"), "", "USD", 20, auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "manager.id1", "USD", 0, auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "manager.id2", "USD", 20, auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "manager.id3", "USD", 10, auditor)

	// Swap among wallets
	TransferCash(network, sel.Get("manager"), "", "EUR", 10, "manager.id1", auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "", "EUR", 10, auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "manager.id1", "EUR", 10, auditor)

	SwapCash(network, sel.Get("manager"), "manager.id1", "EUR", 10, "USD", 10, "manager.id2", auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "", "USD", 20, auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "", "EUR", 10, auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "manager.id1", "USD", 10, auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "manager.id1", "EUR", 0, auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "manager.id2", "USD", 10, auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "manager.id2", "EUR", 10, auditor)
	CheckBalanceAndHolding(network, sel.Get("manager"), "manager.id3", "USD", 10, auditor)

	// no more USD can be issued, reached quota of 220
	IssueCash(network, "", "USD", 10, "alice", auditor, true, "issuer", "no more USD can be issued, reached quota of 241")

	CheckBalanceAndHolding(network, sel.Get("alice"), "", "USD", 0, auditor)
	CheckBalanceAndHolding(network, sel.Get("alice"), "", "EUR", 10, auditor)

	// limits
	CheckBalanceAndHolding(network, sel.Get("alice"), "", "USD", 0, auditor)
	CheckBalanceAndHolding(network, sel.Get("alice"), "", "EUR", 10, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "EUR", 20, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "USD", 110, auditor)
	IssueCash(network, "", "EUR", 2200, "alice", auditor, true, "issuer")
	IssueCash(network, "", "EUR", 2000, "charlie", auditor, true, "issuer")
	CheckBalanceAndHolding(network, sel.Get("alice"), "", "EUR", 2210, auditor)
	CheckBalanceAndHolding(network, sel.Get("charlie"), "", "EUR", 2000, auditor)
	TransferCash(network, sel.Get("alice"), "", "EUR", 210, "bob", auditor, "payment limit reached", "alice", "[EUR][210]")
	CheckBalanceAndHolding(network, sel.Get("alice"), "", "EUR", 2210, auditor)
	CheckBalanceAndHolding(network, sel.Get("charlie"), "", "EUR", 2000, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "USD", 110, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "EUR", 20, auditor)

	PruneInvalidUnspentTokens(network, "issuer", auditor, "alice", "bob", "charlie", "manager")

	TransferCash(network, sel.Get("alice"), "", "EUR", 200, "bob", auditor)
	TransferCash(network, sel.Get("alice"), "", "EUR", 200, "bob", auditor)
	TransferCash(network, sel.Get("alice"), "", "EUR", 200, "bob", auditor)
	TransferCash(network, sel.Get("alice"), "", "EUR", 200, "bob", auditor)
	TransferCash(network, sel.Get("alice"), "", "EUR", 200, "bob", auditor)
	TransferCash(network, sel.Get("alice"), "", "EUR", 200, "bob", auditor)
	TransferCash(network, sel.Get("alice"), "", "EUR", 200, "bob", auditor)
	TransferCash(network, sel.Get("alice"), "", "EUR", 200, "bob", auditor)
	TransferCash(network, sel.Get("alice"), "", "EUR", 200, "bob", auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "EUR", 1820, auditor)
	CheckSpending(network, sel.Get("alice"), "", "EUR", auditor, 1800)
	TransferCash(network, sel.Get("alice"), "", "EUR", 200, "bob", auditor, "cumulative payment limit reached", "alice", "[EUR][2000]")

	TransferCash(network, sel.Get("charlie"), "", "EUR", 200, "bob", auditor)
	PruneInvalidUnspentTokens(network, sel.All("issuer", auditor, "alice", "bob", "charlie", "manager")...)
	TransferCash(network, sel.Get("charlie"), "", "EUR", 200, "bob", auditor)
	TransferCash(network, sel.Get("charlie"), "", "EUR", 200, "bob", auditor)
	TransferCash(network, sel.Get("charlie"), "", "EUR", 200, "bob", auditor)
	TransferCash(network, sel.Get("charlie"), "", "EUR", 200, "bob", auditor)
	CheckBalanceAndHolding(network, sel.Get("charlie"), "", "EUR", 1000, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "EUR", 2820, auditor)
	TransferCash(network, sel.Get("charlie"), "", "EUR", 200, "bob", auditor, "holding limit reached", "bob", "[EUR][3020]")
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "EUR", 2820, auditor)

	PruneInvalidUnspentTokens(network, sel.All("issuer", auditor, "alice", "bob", "charlie", "manager")...)

	// Routing
	IssueCash(network, "", "EUR", 10, "alice.id1", auditor, true, "issuer")
	CheckAcceptedTransactions(network, sel.Get("alice"), "alice.id1", AliceID1AcceptedTransactions[:], nil, nil, nil)
	TransferCash(network, sel.Get("alice"), "alice.id1", "EUR", 10, "bob.id1", auditor)
	CheckBalanceAndHolding(network, sel.Get("alice"), "alice.id1", "EUR", 0, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "bob.id1", "EUR", 10, auditor)

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
			_, err := network.Client(sel.Get("bob")).CallView("transferWithSelector", common.JSONMarshall(&views.Transfer{
				Auditor:      auditor,
				Wallet:       "",
				Type:         "EUR",
				Amount:       v,
				Recipient:    network.Identity("alice"),
				RecipientEID: "alice",
				Retry:        true,
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
	CheckBalanceAndHolding(network, "bob", "", "EUR", 2820-sum, auditor)

	TestParallelTransferWithSelector(network, auditor, sel)

	// Transfer by IDs
	{
		txID1 := IssueCash(network, "", "CHF", 17, "alice", auditor, true, "issuer")
		TransferCashByIDs(network, sel.Get("alice"), "", []*token2.ID{{TxId: txID1, Index: 0}}, 17, "bob", auditor, true, "test release")
		// the previous call should not keep the token locked if release is successful
		txID2 := TransferCashByIDs(network, sel.Get("alice"), "", []*token2.ID{{TxId: txID1, Index: 0}}, 17, "bob", auditor, false)
		WhoDeletedToken(network, sel.Get("alice"), []*token2.ID{{TxId: txID1, Index: 0}}, txID2)
		WhoDeletedToken(network, sel.Get(auditor), []*token2.ID{{TxId: txID1, Index: 0}}, txID2)
		// redeem newly created token
		RedeemCashByIDs(network, sel.Get("bob"), "", []*token2.ID{{TxId: txID2, Index: 0}}, 17, auditor)
	}

	PruneInvalidUnspentTokens(network, sel.All("issuer", auditor, "alice", "bob", "charlie", "manager")...)

	// Test Max Token Value
	IssueCash(network, "", "MAX", 65535, "charlie", auditor, true, "issuer")
	IssueCash(network, "", "MAX", 65535, "charlie", auditor, true, "issuer")
	TransferCash(network, sel.Get("charlie"), "", "MAX", 65536, "alice", auditor, "cannot create output with value [65536], max [65535]")
	IssueCash(network, "", "MAX", 65536, "charlie", auditor, true, "issuer", "q is larger than max token value [65535]")

	// Check consistency
	CheckPublicParams(network, sel.All("issuer", auditor, "alice", "bob", "charlie", "manager")...)
	CheckOwnerDB(network, nil, sel.All("bob", "alice", "issuer", "charlie", "manager")...)
	CheckAuditorDB(network, sel.Get(auditor), "", nil)
	PruneInvalidUnspentTokens(network, sel.All("issuer", auditor, "alice", "bob", "charlie", "manager")...)

	for _, name := range []string{"alice", "bob", "charlie", "manager"} {
		IDs := ListVaultUnspentTokens(network, sel.Get(name))
		CheckIfExistsInVault(network, sel.Get(auditor), IDs)
	}

	// Check double spending by multiple action in the same transaction

	// use the same token for both actions, this must fail
	txIssuedPineapples1 := IssueCash(network, "", "Pineapples", 3, "alice", auditor, true, "issuer")
	IssueCash(network, "", "Pineapples", 3, "alice", auditor, true, "issuer")
	failedTransferTxID := TransferCashMultiActions(network, sel.Get("alice"), "", "Pineapples", []uint64{2, 3}, []string{"bob", "charlie"}, auditor, &token2.ID{TxId: txIssuedPineapples1}, "failed to append spent id", txIssuedPineapples1)
	// the above transfer must fail at execution phase, therefore the auditor should be explicitly informed about this transaction
	CheckBalance(network, sel.Get("alice"), "", "Pineapples", 6)
	CheckHolding(network, sel.Get("alice"), "", "Pineapples", 1, auditor)
	CheckBalance(network, sel.Get("bob"), "", "Pineapples", 0)
	CheckHolding(network, sel.Get("bob"), "", "Pineapples", 2, auditor)
	CheckBalance(network, sel.Get("charlie"), "", "Pineapples", 0)
	CheckHolding(network, sel.Get("charlie"), "", "Pineapples", 3, auditor)
	fmt.Printf("failed transaction [%s]\n", failedTransferTxID)
	SetTransactionAuditStatus(network, sel.Get(auditor), failedTransferTxID, ttx.Deleted)
	CheckBalanceAndHolding(network, sel.Get("alice"), "", "Pineapples", 6, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "Pineapples", 0, auditor)
	CheckBalanceAndHolding(network, sel.Get("charlie"), "", "Pineapples", 0, auditor)
	CheckAuditorDB(network, sel.Get(auditor), "", nil)
}

func TestParallelTransferWithSelector(network *integration.Infrastructure, auditor string, sel *token3.ReplicaSelector) {

	// Transfer With Selector
	IssueCash(network, "", "YUAN", 17, "alice", auditor, true, "issuer")
	TransferCashWithSelector(network, sel.Get("alice"), "", "YUAN", 10, "bob", auditor)
	CheckBalanceAndHolding(network, sel.Get("alice"), "", "YUAN", 7, auditor)
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "YUAN", 10, auditor)
	TransferCashWithSelector(network, sel.Get("alice"), "", "YUAN", 10, "bob", auditor, "pineapple", "insufficient funds")

	// Now, the tests asks Bob to transfer to Charlie 14 YUAN split in two parallel transactions each one transferring 7 YUAN.
	// Notice that Bob has only 10 YUAN, therefore bob will be able to assemble only one transfer.
	// We use two channels to collect the results of the two transfers.
	transferErrors := make([]chan error, 2)
	for i := range transferErrors {
		transferErrors[i] = make(chan error, 1)

		transferError := transferErrors[i]
		go func() {
			txid, err := network.Client(sel.Get("bob")).CallView("transferWithSelector", common.JSONMarshall(&views.Transfer{
				Auditor:      auditor,
				Wallet:       "",
				Type:         "YUAN",
				Amount:       7,
				Recipient:    network.Identity("charlie"),
				RecipientEID: "charlie",
				Retry:        false,
			}))
			if err != nil {
				// The transaction failed, we return the error to the caller.
				transferError <- err
				return
			}
			// The transaction didn't fail, let's wait for it to be confirmed, and return no error
			common2.CheckFinality(network, "charlie", common.JSONUnmarshalString(txid), nil, false)
			transferError <- nil
		}()
	}
	// collect the errors, and check that they are all nil, and one of them is the error we expect.
	var errs []error
	for _, transfer := range transferErrors {
		errs = append(errs, <-transfer)
	}
	Expect((errs[0] == nil && errs[1] != nil) || (errs[0] != nil && errs[1] == nil)).To(BeTrue())
	var errStr string
	if errs[0] == nil {
		errStr = errs[1].Error()
	} else {
		errStr = errs[0].Error()
	}
	// TODO: Temporarily disabled.
	// The token selectors update their tokens based on the finality notifications. These come from the commit pipeline.
	// Currently, if a replica is the receiver of a token, the DB will be updated, but only the commit pipeline of that
	// replica will process the token. The other replicas (and hence their token selectors) will not register this new token.
	// Hence, we introduced a temporary extra check in the DB when we find no token, to make sure that in the meantime
	// no other replica has added a token that this replica could use.
	// The same we did when we are looking for a token in a currency that we haven't seen before.
	// However, in this test where two processes of the token selector try to use the same token, the first one will lock it.
	// The second process will not see the token, hence it will look it up in the DB. It will find it there and create another
	// transaction to spend the same token. The first transaction will go through, but the second one will fail, as we
	// are trying to write the same RWSet (MVCC_READ_CONFLICT) and this token will appear as deleted.
	// The result will be the same (the quickest transfer will go through and the slowest will fail), but the error
	// will be different. Hence, we take out the error check until the concurrent token selection is solved.
	//v := strings.Contains(errStr, "pineapple") || strings.Contains(errStr, "lemonade")
	//Expect(v).To(BeEquivalentTo(true))
	Expect(errStr).NotTo(BeEmpty())

	CheckBalanceAndHolding(network, sel.Get("bob"), "", "YUAN", 3, auditor)
	CheckBalanceAndHolding(network, sel.Get("alice"), "", "YUAN", 7, auditor)
	CheckBalanceAndHolding(network, sel.Get("charlie"), "", "YUAN", 7, auditor)
}

func testInit(network *integration.Infrastructure, auditor string, sel *token3.ReplicaSelector) {
	RegisterAuditor(network, sel.Get(auditor))

	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	SetKVSEntry(network, "issuer", sel.Get("auditor"), auditor)
	CheckPublicParams(network, sel.All("issuer", auditor, "alice", "bob", "charlie", "manager")...)
}

func TestPublicParamsUpdate(network *integration.Infrastructure, auditor string, ppBytes []byte, tms *topology.TMS, issuerAsAuditor bool) {
	var errorMessage string
	if issuerAsAuditor {
		errorMessage = "failed verifying auditor signature"
		RegisterAuditor(network, "issuer")
		txId := IssueCash(network, "", "USD", 110, "alice", "issuer", true, "issuer")
		Expect(txId).NotTo(BeEmpty())
		CheckBalanceAndHolding(network, "alice", "", "USD", 110, "issuer")
	} else {
		errorMessage = "failed to verify issuers' signatures"
		RegisterAuditor(network, "auditor")
		txId := IssueCash(network, "", "USD", 110, "alice", "auditor", true, "issuer")
		Expect(txId).NotTo(BeEmpty())
		CheckBalanceAndHolding(network, "alice", "", "USD", 110, "auditor")
	}

	RegisterAuditor(network, auditor)
	UpdatePublicParams(network, ppBytes, tms)

	Eventually(GetPublicParams).WithArguments(network, "newIssuer").WithTimeout(30 * time.Second).WithPolling(15 * time.Second).Should(Equal(ppBytes))
	if !issuerAsAuditor {
		Eventually(GetPublicParams).WithArguments(network, auditor).WithTimeout(30 * time.Second).WithPolling(15 * time.Second).Should(Equal(ppBytes))
	}
	// give time to the issuer and the auditor to update their public parameters and reload their wallets
	Eventually(DoesWalletExist).WithArguments(network, "newIssuer", "", views.IssuerWallet).WithTimeout(1 * time.Minute).WithPolling(15 * time.Second).Should(Equal(true))
	if issuerAsAuditor {
		Eventually(DoesWalletExist).WithArguments(network, "newIssuer", "", views.AuditorWallet).WithTimeout(1 * time.Minute).WithPolling(15 * time.Second).Should(Equal(true))
	} else {
		Eventually(DoesWalletExist).WithArguments(network, auditor, "", views.AuditorWallet).WithTimeout(1 * time.Minute).WithPolling(15 * time.Second).Should(Equal(true))
	}
	Eventually(DoesWalletExist).WithArguments(network, "alice", "", views.OwnerWallet).WithTimeout(1 * time.Minute).WithPolling(15 * time.Second).Should(Equal(true))
	Eventually(DoesWalletExist).WithArguments(network, "manager", "manager.id1", views.OwnerWallet).WithTimeout(1 * time.Minute).WithPolling(15 * time.Second).Should(Equal(true))

	txId := IssueCash(network, "", "USD", 110, "alice", auditor, true, "newIssuer")
	Expect(txId).NotTo(BeEmpty())
	CheckBalance(network, "alice", "", "USD", 220)
	CheckHolding(network, "alice", "", "USD", 110, auditor)
	IssueCash(network, "", "USD", 110, "alice", auditor, true, "issuer", errorMessage)

	CheckOwnerWalletIDs(network, "manager", "manager.id1", "manager.id2", "manager.id3")
}

func testTwoGeneratedOwnerWalletsSameNode(network *integration.Infrastructure, auditor string, useFabricCA bool) {
	tokenPlatform := token.GetPlatform(network.Ctx, "token")
	idConfig1 := tokenPlatform.GenOwnerCryptoMaterial(tokenPlatform.GetTopology().TMSs[0].BackendTopology.Name(), "charlie", "charlie.ExtraId1", false)
	RegisterOwnerIdentity(network, "charlie", idConfig1)
	idConfig2 := tokenPlatform.GenOwnerCryptoMaterial(tokenPlatform.GetTopology().TMSs[0].BackendTopology.Name(), "charlie", "charlie.ExtraId2", useFabricCA)
	RegisterOwnerIdentity(network, "charlie", idConfig2)

	IssueCash(network, "", "SPE", 100, "charlie", auditor, true, "issuer")
	TransferCash(network, "charlie", "", "SPE", 25, "charlie.ExtraId1", auditor)
	Restart(network, false, "charlie")
	TransferCash(network, "charlie", "charlie.ExtraId1", "SPE", 15, "charlie.ExtraId2", auditor)

	CheckBalanceAndHolding(network, "charlie", "", "SPE", 75, auditor)
	CheckBalanceAndHolding(network, "charlie", "charlie.ExtraId1", "SPE", 10, auditor)
	CheckBalanceAndHolding(network, "charlie", "charlie.ExtraId2", "SPE", 15, auditor)
}

func TestRevokeIdentity(network *integration.Infrastructure, auditor string, revocationHandle string, errorMessage string) {
	IssueCash(network, "", "USD", 110, "alice", auditor, true, "issuer")
	CheckBalanceAndHolding(network, "alice", "", "USD", 110, auditor)
	CheckBalanceAndHolding(network, "bob", "", "USD", 0, auditor)
	CheckBalanceAndHolding(network, "bob", "bob.id1", "USD", 0, auditor)

	rId := GetRevocationHandle(network, "bob")
	RevokeIdentity(network, auditor, revocationHandle)
	// try to issue to bob
	IssueCash(network, "", "USD", 22, "bob", auditor, true, "issuer", hash.Hashable(rId).String()+" Identity is in revoked state")
	// try to transfer to bob
	TransferCash(network, "alice", "", "USD", 22, "bob", auditor, hash.Hashable(rId).String()+" Identity is in revoked state")
	CheckBalanceAndHolding(network, "alice", "", "USD", 110, auditor)
	CheckBalanceAndHolding(network, "bob", "", "USD", 0, auditor)
	CheckBalanceAndHolding(network, "bob", "bob.id1", "USD", 0, auditor)

	// Issuer to bob.id1
	IssueCash(network, "", "USD", 90, "bob.id1", auditor, true, "issuer")
	CheckBalanceAndHolding(network, "alice", "", "USD", 110, auditor)
	CheckBalanceAndHolding(network, "bob", "", "USD", 0, auditor)
	CheckBalanceAndHolding(network, "bob", "bob.id1", "USD", 90, auditor)
}

func TestMixed(network *integration.Infrastructure) {
	dlogId := getTmsId(network, DLogNamespace)
	fabTokenId := getTmsId(network, FabTokenNamespace)
	RegisterAuditorForTMSID(network, "auditor1", dlogId)
	RegisterAuditorForTMSID(network, "auditor2", fabTokenId)

	// give some time to the nodes to get the public parameters
	time.Sleep(40 * time.Second)

	Eventually(CheckPublicParamsMatch).WithArguments(network, dlogId, "issuer1", "auditor1", "alice", "bob").WithTimeout(1 * time.Minute).WithPolling(15 * time.Second).Should(Equal(true))
	Eventually(CheckPublicParamsMatch).WithArguments(network, fabTokenId, "issuer2", "auditor2", "alice", "bob").WithTimeout(1 * time.Minute).WithPolling(15 * time.Second).Should(Equal(true))

	IssueCashForTMSID(network, "", "USD", 110, "alice", "auditor1", true, "issuer1", dlogId)
	IssueCashForTMSID(network, "", "USD", 115, "alice", "auditor2", true, "issuer2", fabTokenId)

	TransferCashForTMSID(network, "alice", "", "USD", 20, "bob", "auditor1", dlogId)
	TransferCashForTMSID(network, "alice", "", "USD", 30, "bob", "auditor2", fabTokenId)

	RedeemCashForTMSID(network, "bob", "", "USD", 11, "auditor1", dlogId)
	CheckSpendingForTMSID(network, "bob", "", "USD", "auditor1", 11, dlogId)

	CheckBalanceAndHoldingForTMSID(network, "alice", "", "USD", 90, "auditor1", dlogId)
	CheckBalanceAndHoldingForTMSID(network, "alice", "", "USD", 85, "auditor2", fabTokenId)
	CheckBalanceAndHoldingForTMSID(network, "bob", "", "USD", 9, "auditor1", dlogId)
	CheckBalanceAndHoldingForTMSID(network, "bob", "", "USD", 30, "auditor2", fabTokenId)

	h := ListIssuerHistoryForTMSID(network, "", "USD", "issuer1", dlogId)
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(110))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	// Error cases

	// Try to approve dlog with auditor2
	TransferCashForTMSID(network, "alice", "", "USD", 20, "bob", "auditor2", dlogId, "")
	// Try to issue on dlog with issuer2
	IssueCashForTMSID(network, "", "USD", 110, "alice", "auditor1", true, "issuer2", dlogId, "")
	// Try to spend on dlog coins from fabtoken
	TransferCashForTMSID(network, "alice", "", "USD", 120, "bob", "auditor2", fabTokenId, "")
	// Try to issue more coins than the max
	IssueCashForTMSID(network, "", "MAX", 65535, "bob", "auditor1", true, "issuer1", dlogId)
	IssueCashForTMSID(network, "", "MAX", 65536, "bob", "auditor2", true, "issuer2", fabTokenId, "q is larger than max token value [65535]")

	// Shut down one auditor and try to issue cash for both chaincodes
	Restart(network, true, "auditor2")
	IssueCashForTMSID(network, "", "USD", 10, "alice", "auditor1", true, "issuer1", dlogId)
	IssueCashForTMSID(network, "", "USD", 20, "alice", "auditor2", true, "issuer2", fabTokenId, "")
	RegisterAuditor(network, "auditor2")
	IssueCashForTMSID(network, "", "USD", 30, "alice", "auditor2", true, "issuer2", fabTokenId)

	CheckBalanceAndHoldingForTMSID(network, "alice", "", "USD", 100, "auditor1", dlogId)
	CheckBalanceAndHoldingForTMSID(network, "alice", "", "USD", 115, "auditor2", fabTokenId)
}

func TestRemoteOwnerWallet(network *integration.Infrastructure, auditor string, websSocket bool) {
	TestRemoteOwnerWalletWithWMP(network, NewWalletManagerProvider(&walletManagerLoader{II: network}), auditor, websSocket)
}

func TestRemoteOwnerWalletWithWMP(network *integration.Infrastructure, wmp *WalletManagerProvider, auditor string, websSocket bool) {
	RegisterAuditor(network, auditor)

	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	SetKVSEntry(network, "issuer", "auditor", auditor)
	CheckPublicParams(network, "issuer", auditor, "alice", "bob", "charlie", "manager")

	Withdraw(network, wmp, "alice", "alice_remote", "USD", 10, auditor, "issuer")
	CheckBalanceAndHolding(network, "alice", "alice_remote", "USD", 10, auditor)

	TransferCashFromExternalWallet(network, wmp, websSocket, "alice", "alice_remote", "USD", 7, "bob", auditor)
	CheckBalanceAndHolding(network, "alice", "alice_remote", "USD", 3, auditor)
	CheckBalanceAndHolding(network, "alice", "alice_remote_2", "USD", 0, auditor)
	CheckBalanceAndHolding(network, "bob", "", "USD", 7, auditor)
	TransferCashToExternalWallet(network, wmp, "bob", "", "USD", 3, "alice", "alice_remote", auditor)
	CheckBalanceAndHolding(network, "alice", "alice_remote", "USD", 6, auditor)
	CheckBalanceAndHolding(network, "alice", "alice_remote_2", "USD", 0, auditor)
	CheckBalanceAndHolding(network, "bob", "", "USD", 4, auditor)
	TransferCashFromExternalWallet(network, wmp, websSocket, "alice", "alice_remote", "USD", 4, "charlie", auditor)
	CheckBalanceAndHolding(network, "alice", "alice_remote", "USD", 2, auditor)
	CheckBalanceAndHolding(network, "alice", "alice_remote_2", "USD", 0, auditor)
	CheckBalanceAndHolding(network, "bob", "", "USD", 4, auditor)
	CheckBalanceAndHolding(network, "bob", "bob_remote", "USD", 0, auditor)
	CheckBalanceAndHolding(network, "charlie", "", "USD", 4, auditor)
	TransferCashFromAndToExternalWallet(network, wmp, websSocket, "alice", "alice_remote", "USD", 1, "bob", "bob_remote", auditor)
	CheckBalanceAndHolding(network, "alice", "alice_remote", "USD", 1, auditor)
	CheckBalanceAndHolding(network, "alice", "alice_remote_2", "USD", 0, auditor)
	CheckBalanceAndHolding(network, "bob", "", "USD", 4, auditor)
	CheckBalanceAndHolding(network, "bob", "bob_remote", "USD", 1, auditor)
	CheckBalanceAndHolding(network, "charlie", "", "USD", 4, auditor)
	TransferCashFromAndToExternalWallet(network, wmp, websSocket, "alice", "alice_remote", "USD", 1, "alice", "alice_remote_2", auditor)
	CheckBalanceAndHolding(network, "alice", "alice_remote", "USD", 0, auditor)
	CheckBalanceAndHolding(network, "alice", "alice_remote_2", "USD", 1, auditor)
	CheckBalanceAndHolding(network, "bob", "", "USD", 4, auditor)
	CheckBalanceAndHolding(network, "bob", "bob_remote", "USD", 1, auditor)
	CheckBalanceAndHolding(network, "charlie", "", "USD", 4, auditor)
}

func TestMaliciousTransactions(net *integration.Infrastructure) {
	CheckPublicParams(net, "issuer", "alice", "bob", "charlie", "manager")

	Eventually(DoesWalletExist).WithArguments(net, "issuer", "", views.IssuerWallet).WithTimeout(1 * time.Minute).WithPolling(15 * time.Second).Should(Equal(true))
	Eventually(DoesWalletExist).WithArguments(net, "alice", "", views.OwnerWallet).WithTimeout(1 * time.Minute).WithPolling(15 * time.Second).Should(Equal(true))
	Eventually(DoesWalletExist).WithArguments(net, "bob", "", views.OwnerWallet).WithTimeout(1 * time.Minute).WithPolling(15 * time.Second).Should(Equal(true))
	IssueCash(net, "", "USD", 110, "alice", "", true, "issuer")
	CheckBalance(net, "alice", "", "USD", 110)

	txID := MaliciousTransferCash(net, "alice", "", "USD", 2, "bob", "", nil)
	txStatusAlice := GetTXStatus(net, "alice", txID)
	Expect(txStatusAlice.ValidationCode).To(BeEquivalentTo(ttx.Deleted))
	Expect(txStatusAlice.ValidationMessage).To(ContainSubstring("token requests do not match, tr hashes"))
	txStatusBob := GetTXStatus(net, "bob", txID)
	Expect(txStatusBob.ValidationCode).To(BeEquivalentTo(ttx.Deleted))
	Expect(txStatusBob.ValidationMessage).To(ContainSubstring("token requests do not match, tr hashes"))
}
