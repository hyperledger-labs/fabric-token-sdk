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

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
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

	CheckPublicParams(network, "issuer", "auditor", "alice", "bob", "charlie", "manager")

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

	// Register a new issuer wallet and issue with that wallet
	tokenPlatform := token.GetPlatform(network.Ctx, "token")
	// Gen crypto material for the new issuer wallet
	newIssuerWalletPath := tokenPlatform.GenIssuerCryptoMaterial(tokenPlatform.Topology.TMSs[0].BackendTopology.Name(), "issuer", "issuer.ExtraId")
	// Register it
	RegisterIssuerWallet(network, "issuer", "newIssuerWallet", newIssuerWalletPath)
	// Issuer tokens with this new wallet
	t4 := time.Now()
	IssueCash(network, "newIssuerWallet", "EUR", 10, "bob", auditor, false)
	//t5 := time.Now()
	CheckBalanceAndHolding(network, "bob", "", "EUR", 10)
	IssueCash(network, "newIssuerWallet", "EUR", 10, "bob", auditor, true)
	//t6 := time.Now()
	CheckBalanceAndHolding(network, "bob", "", "EUR", 20)
	IssueCash(network, "newIssuerWallet", "EUR", 10, "bob", auditor, false)
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

	h = ListIssuerHistory(network, "newIssuerWallet", "EUR")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(30))).To(BeEquivalentTo(0))
	Expect(h.ByType("EUR").Count()).To(BeEquivalentTo(h.Count()))

	CheckBalanceAndHolding(network, "alice", "", "USD", 120)
	CheckBalanceAndHolding(network, "bob", "", "EUR", 30)

	Restart(network, false, "alice")

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
	IssueCash(network, "newIssuerWallet", "EUR", 150, "issuer", auditor, true)
	IssueCash(network, "issuer.id1", "EUR", 10, "issuer.owner", auditor, true)

	h = ListIssuerHistory(network, "", "USD")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(241))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = ListIssuerHistory(network, "newIssuerWallet", "EUR")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(180))).To(BeEquivalentTo(0))
	Expect(h.ByType("EUR").Count()).To(BeEquivalentTo(h.Count()))

	CheckBalanceAndHolding(network, "issuer", "", "USD", 110)
	CheckBalanceAndHolding(network, "issuer", "", "EUR", 150)
	CheckBalanceAndHolding(network, "issuer", "issuer.owner", "EUR", 10)

	// Restart the auditor
	Restart(network, false, auditor)
	RegisterAuditor(network, auditor)

	CheckBalanceAndHolding(network, "issuer", "", "USD", 110)
	CheckBalanceAndHolding(network, "issuer", "", "EUR", 150)
	CheckBalanceAndHolding(network, "issuer", "issuer.owner", "EUR", 10)

	CheckOwnerDB(network, nil, "issuer", "alice", "bob", "charlie", "manager")
	CheckAuditorDB(network, auditor, "")

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
	CheckBalanceAndHolding(network, "bob", "", "EUR", 20)
	CheckBalanceAndHolding(network, "bob", "", "USD", 110)
	Restart(network, true, "bob")
	Restart(network, false, auditor)
	RegisterAuditor(network, auditor)
	CheckBalance(network, "bob", "", "PINE", 0)
	CheckHolding(network, "bob", "", "PINE", 110)
	CheckBalanceAndHolding(network, "bob", "", "EUR", 20)
	CheckBalanceAndHolding(network, "bob", "", "USD", 110)
	CheckOwnerDB(network, nil, "bob")
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
	CheckOwnerDB(network, nil, "issuer", "alice", "bob", "charlie", "manager")
	CheckAuditorDB(network, auditor, "")

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
	SetTransactionOwnersStatus(network, txID3, ttx.Deleted, "alice", "bob")

	// Restart
	CheckOwnerDB(network, nil, "bob", "alice")
	CheckOwnerDB(network, nil, "issuer", "charlie", "manager")
	CheckAuditorDB(network, auditor, "")
	Restart(network, false, "alice", "bob", "charlie", "manager")
	CheckOwnerDB(network, nil, "bob", "alice")
	CheckOwnerDB(network, nil, "issuer", "charlie", "manager")
	CheckAuditorDB(network, auditor, "")

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
	IssueCash(network, "", "USD", 10, "alice", auditor, true, "no more USD can be issued, reached quota of 241")

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
	{
		txID1 := IssueCash(network, "", "CHF", 17, "alice", auditor, true)
		TransferCashByIDs(network, "alice", "", []*token2.ID{{TxId: txID1, Index: 0}}, 17, "bob", auditor, true, "test release")
		// the previous call should not keep the token locked if release is successful
		txID2 := TransferCashByIDs(network, "alice", "", []*token2.ID{{TxId: txID1, Index: 0}}, 17, "bob", auditor, false)
		WhoDeletedToken(network, "alice", []*token2.ID{{TxId: txID1, Index: 0}}, txID2)
		WhoDeletedToken(network, auditor, []*token2.ID{{TxId: txID1, Index: 0}}, txID2)
		// redeem newly created token
		RedeemCashByIDs(network, "bob", "", []*token2.ID{{TxId: txID2, Index: 0}}, 17, auditor)
	}

	// Test Max Token Value
	IssueCash(network, "", "MAX", 9999, "charlie", auditor, true)
	IssueCash(network, "", "MAX", 9999, "charlie", auditor, true)
	TransferCash(network, "charlie", "", "MAX", 10000, "alice", auditor, "cannot create output with value [10000], max [9999]")
	IssueCash(network, "", "MAX", 10000, "charlie", auditor, true, "q is larger than max token value [9999]")

	// Check consistency
	CheckPublicParams(network, "issuer", "auditor", "alice", "bob", "charlie", "manager")
	CheckOwnerDB(network, nil, "bob", "alice", "issuer", "charlie", "manager")
	CheckAuditorDB(network, auditor, "")
	PruneInvalidUnspentTokens(network, "issuer", "auditor", "alice", "bob", "charlie", "manager")
}
