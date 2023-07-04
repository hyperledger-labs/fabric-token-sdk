package mixed

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	. "github.com/onsi/gomega"
	"math/big"
	"time"
)

func TestAll(network *integration.Infrastructure) {
	fungible.RegisterAuditor(network, "auditor", nil)

	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	fungible.CheckPublicParams(network, "issuer", "auditor", "alice", "bob", "charlie")

	t0 := time.Now()

	fungible.IssueCash(network, "", "USD", 110, "alice", "auditor", true, "issuer")
	t1 := time.Now()
	fungible.CheckBalanceAndHolding(network, "alice", "", "USD", 110, "auditor")
	fungible.CheckAuditedTransactions(network, "auditor", fungible.AuditedTransactions[:1], nil, nil)
	fungible.CheckAuditedTransactions(network, "auditor", fungible.AuditedTransactions[:1], &t0, &t1)
	fungible.CheckAcceptedTransactions(network, "alice", "", fungible.AliceAcceptedTransactions[:1], nil, nil, nil, ttxdb.Issue)
	fungible.CheckAcceptedTransactions(network, "alice", "", fungible.AliceAcceptedTransactions[:1], nil, nil, []ttxdb.TxStatus{ttxdb.Confirmed}, ttxdb.Issue)
	fungible.CheckAcceptedTransactions(network, "alice", "", fungible.AliceAcceptedTransactions[:1], nil, nil, nil)
	fungible.CheckAcceptedTransactions(network, "alice", "", fungible.AliceAcceptedTransactions[:1], &t0, &t1, nil)

	t2 := time.Now()
	fungible.IssueCash(network, "", "USD", 10, "alice", "auditor", false, "issuer")
	t3 := time.Now()
	fungible.CheckBalanceAndHolding(network, "alice", "", "USD", 120, "auditor")
	fungible.CheckBalanceAndHolding(network, "alice", "alice", "USD", 120, "auditor")
	fungible.CheckAuditedTransactions(network, "auditor", fungible.AuditedTransactions[:2], nil, nil)
	fungible.CheckAuditedTransactions(network, "auditor", fungible.AuditedTransactions[:2], &t0, &t3)
	fungible.CheckAuditedTransactions(network, "auditor", fungible.AuditedTransactions[1:2], &t2, &t3)
	fungible.CheckAcceptedTransactions(network, "alice", "", fungible.AliceAcceptedTransactions[:2], nil, nil, nil)
	fungible.CheckAcceptedTransactions(network, "alice", "", fungible.AliceAcceptedTransactions[:2], &t0, &t3, nil)
	fungible.CheckAcceptedTransactions(network, "alice", "", fungible.AliceAcceptedTransactions[1:2], &t2, &t3, nil)

	fungible.Restart(network, true, "auditor")
	fungible.RegisterAuditor(network, "auditor", nil)

	fungible.CheckBalanceAndHolding(network, "alice", "", "USD", 120, "auditor")
	fungible.CheckBalanceAndHolding(network, "alice", "alice", "USD", 120, "auditor")
	fungible.CheckAuditedTransactions(network, "auditor", fungible.AuditedTransactions[:2], nil, nil)
	fungible.CheckAuditedTransactions(network, "auditor", fungible.AuditedTransactions[:2], &t0, &t3)
	fungible.CheckAuditedTransactions(network, "auditor", fungible.AuditedTransactions[1:2], &t2, &t3)
	fungible.CheckAcceptedTransactions(network, "alice", "", fungible.AliceAcceptedTransactions[:2], nil, nil, nil)
	fungible.CheckAcceptedTransactions(network, "alice", "", fungible.AliceAcceptedTransactions[:2], &t0, &t3, nil)
	fungible.CheckAcceptedTransactions(network, "alice", "", fungible.AliceAcceptedTransactions[1:2], &t2, &t3, nil)

	h := fungible.ListIssuerHistory(network, "", "USD")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(120))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = fungible.ListIssuerHistory(network, "", "EUR")
	Expect(h.Count()).To(BeEquivalentTo(0))

	// Register a new issuer wallet and issue with that wallet
	tokenPlatform := token.GetPlatform(network.Ctx, "token")
	Expect(tokenPlatform).ToNot(BeNil(), "cannot find token platform in context")
	Expect(tokenPlatform.GetTopology()).ToNot(BeNil(), "invalid token topology, it is nil")
	Expect(len(tokenPlatform.GetTopology().TMSs)).ToNot(BeEquivalentTo(0), "no tms defined in token topology")

	fungible.CheckBalanceAndHolding(network, "alice", "", "USD", 120, "auditor")
	//fungible.CheckBalanceAndHolding(network, "bob", "", "EUR", 30, "auditor")

	fungible.Restart(network, false, "alice")

	t8 := time.Now()
	fungible.TransferCash(network, "alice", "", "USD", 111, "bob", "auditor")
	t9 := time.Now()
	fungible.CheckAuditedTransactions(network, "auditor", fungible.AuditedTransactions[5:7], &t8, &t9)
	fungible.CheckSpending(network, "alice", "", "USD", "auditor", 111)
	fungible.CheckAcceptedTransactions(network, "bob", "", fungible.BobAcceptedTransactions[0:1], &t8, &t9, nil)
	fungible.CheckAcceptedTransactions(network, "bob", "", fungible.BobAcceptedTransactions[0:1], &t8, &t9, nil, ttxdb.Transfer)
	fungible.CheckAcceptedTransactions(network, "bob", "", fungible.BobAcceptedTransactions[0:1], &t8, &t9, []ttxdb.TxStatus{ttxdb.Confirmed}, ttxdb.Transfer)
	fungible.CheckAcceptedTransactions(network, "bob", "", fungible.BobAcceptedTransactions[:1], &t0, &t9, nil)
	fungible.CheckAcceptedTransactions(network, "bob", "", fungible.BobAcceptedTransactions[:1], nil, nil, nil)
	ut := fungible.ListUnspentTokens(network, "alice", "", "USD")
	Expect(ut.Count() > 0).To(BeTrue())
	Expect(ut.Sum(64).ToBigInt().Cmp(big.NewInt(9))).To(BeEquivalentTo(0))
	Expect(ut.ByType("USD").Count()).To(BeEquivalentTo(ut.Count()))
	ut = fungible.ListUnspentTokens(network, "bob", "", "USD")
	Expect(ut.Count() > 0).To(BeTrue())
	Expect(ut.Sum(64).ToBigInt().Cmp(big.NewInt(111))).To(BeEquivalentTo(0))
	Expect(ut.ByType("USD").Count()).To(BeEquivalentTo(ut.Count()))

	fungible.RedeemCash(network, "bob", "", "USD", 11, "auditor")
	t10 := time.Now()
	fungible.CheckAcceptedTransactions(network, "bob", "", fungible.BobAcceptedTransactions[:3], nil, nil, nil)
	fungible.CheckAcceptedTransactions(network, "bob", "", fungible.BobAcceptedTransactions[2:3], nil, nil, nil, ttxdb.Redeem)
	fungible.CheckAcceptedTransactions(network, "bob", "", fungible.BobAcceptedTransactions[2:3], nil, nil, []ttxdb.TxStatus{ttxdb.Confirmed}, ttxdb.Redeem)
	fungible.CheckAuditedTransactions(network, "auditor", fungible.AuditedTransactions[7:9], &t9, &t10)

	// DONE HERE --
	//t11 := time.Now()
	//IssueCash(network, "", "USD", 10, "bob", auditor, true, "issuer")
	//t12 := time.Now()
	//CheckAuditedTransactions(network, auditor, AuditedTransactions[9:10], &t11, &t12)
	//CheckAuditedTransactions(network, auditor, AuditedTransactions[:10], &t0, &t12)
	//CheckSpending(network, "bob", "", "USD", auditor, 11)
	//
	//// test multi action transfer...
	//t13 := time.Now()
	//IssueCash(network, "", "LIRA", 3, "alice", auditor, true, "issuer")
	//IssueCash(network, "", "LIRA", 3, "alice", auditor, true, "issuer")
	//t14 := time.Now()
	//CheckAuditedTransactions(network, auditor, AuditedTransactions[10:12], &t13, &t14)
	//txLiraTransfer := TransferCashMultiActions(network, "alice", "", "LIRA", []uint64{2, 3}, []string{"bob", "charlie"}, auditor)
	//t16 := time.Now()
	//AuditedTransactions[12].TxID = txLiraTransfer
	//AuditedTransactions[13].TxID = txLiraTransfer
	//AuditedTransactions[14].TxID = txLiraTransfer
	//CheckBalanceAndHolding(network, "alice", "", "LIRA", 1, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "LIRA", 2, auditor)
	//CheckBalanceAndHolding(network, "charlie", "", "LIRA", 3, auditor)
	//CheckAuditedTransactions(network, auditor, AuditedTransactions[:], &t0, &t16)
	//
	//IssueCash(network, "", "USD", 1, "alice", auditor, true, "issuer")
	//
	//testTwoGeneratedOwnerWalletsSameNode(network, auditor)
	//
	//CheckBalanceAndHolding(network, "alice", "", "USD", 10, auditor)
	//CheckBalanceAndHolding(network, "alice", "", "EUR", 0, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "EUR", 30, auditor)
	//CheckBalanceAndHolding(network, "bob", "bob", "EUR", 30, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "USD", 110, auditor)
	//
	//SwapCash(network, "alice", "", "USD", 10, "EUR", 10, "bob", auditor)
	//
	//CheckBalanceAndHolding(network, "alice", "", "USD", 0, auditor)
	//CheckBalanceAndHolding(network, "alice", "", "EUR", 10, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "EUR", 20, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "USD", 120, auditor)
	//CheckSpending(network, "alice", "", "USD", auditor, 121)
	//CheckSpending(network, "bob", "", "EUR", auditor, 10)
	//
	//RedeemCash(network, "bob", "", "USD", 10, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "USD", 110, auditor)
	//CheckSpending(network, "bob", "", "USD", auditor, 21)
	//
	//// Check self endpoints
	//IssueCash(network, "", "USD", 110, "issuer", auditor, true, "issuer")
	//IssueCash(network, "newIssuerWallet", "EUR", 150, "issuer", auditor, true, "issuer")
	//IssueCash(network, "issuer.id1", "EUR", 10, "issuer.owner", auditor, true, "issuer")
	//
	//h = ListIssuerHistory(network, "", "USD")
	//Expect(h.Count() > 0).To(BeTrue())
	//Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(241))).To(BeEquivalentTo(0))
	//Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))
	//
	//h = ListIssuerHistory(network, "newIssuerWallet", "EUR")
	//Expect(h.Count() > 0).To(BeTrue())
	//Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(180))).To(BeEquivalentTo(0))
	//Expect(h.ByType("EUR").Count()).To(BeEquivalentTo(h.Count()))
	//
	//CheckBalanceAndHolding(network, "issuer", "", "USD", 110, auditor)
	//CheckBalanceAndHolding(network, "issuer", "", "EUR", 150, auditor)
	//CheckBalanceAndHolding(network, "issuer", "issuer.owner", "EUR", 10, auditor)
	//
	//// Restart the auditor
	//Restart(network, true, auditor)
	//RegisterAuditor(network, auditor, onAuditorRestart)
	//
	//CheckBalanceAndHolding(network, "issuer", "", "USD", 110, auditor)
	//CheckBalanceAndHolding(network, "issuer", "", "EUR", 150, auditor)
	//CheckBalanceAndHolding(network, "issuer", "issuer.owner", "EUR", 10, auditor)
	//
	//CheckOwnerDB(network, nil, "issuer", "alice", "bob", "charlie", "manager")
	//CheckAuditorDB(network, auditor, "")
	//
	//// Check double spending
	//txIDPine := IssueCash(network, "", "PINE", 55, "alice", auditor, true, "issuer")
	//tokenIDPine := &token2.ID{
	//	TxId:  txIDPine,
	//	Index: 0,
	//}
	//txID1, tx1 := PrepareTransferCash(network, "alice", "", "PINE", 55, "bob", auditor, tokenIDPine)
	//CheckBalance(network, "alice", "", "PINE", 55)
	//CheckHolding(network, "alice", "", "PINE", 0, auditor)
	//CheckBalance(network, "bob", "", "PINE", 0)
	//CheckHolding(network, "bob", "", "PINE", 55, auditor)
	//txID2, tx2 := PrepareTransferCash(network, "alice", "", "PINE", 55, "bob", auditor, tokenIDPine)
	//CheckBalance(network, "alice", "", "PINE", 55)
	//CheckHolding(network, "alice", "", "PINE", -55, auditor)
	//CheckBalance(network, "bob", "", "PINE", 0)
	//CheckHolding(network, "bob", "", "PINE", 110, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "EUR", 20, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "USD", 110, auditor)
	//Restart(network, true, "bob")
	//Restart(network, false, auditor)
	//RegisterAuditor(network, auditor, onAuditorRestart)
	//CheckBalance(network, "bob", "", "PINE", 0)
	//CheckHolding(network, "bob", "", "PINE", 110, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "EUR", 20, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "USD", 110, auditor)
	//CheckOwnerDB(network, nil, "bob")
	//BroadcastPreparedTransferCash(network, "alice", txID1, tx1, true)
	//Expect(network.Client("bob").IsTxFinal(txID1)).NotTo(HaveOccurred())
	//Expect(network.Client(auditor).IsTxFinal(txID1)).NotTo(HaveOccurred())
	//CheckBalance(network, "alice", "", "PINE", 0)
	//CheckHolding(network, "alice", "", "PINE", -55, auditor)
	//CheckBalance(network, "bob", "", "PINE", 55)
	//CheckHolding(network, "bob", "", "PINE", 110, auditor)
	//BroadcastPreparedTransferCash(network, "alice", txID2, tx2, true, "is not valid")
	//Expect(network.Client("bob").IsTxFinal(txID2)).To(HaveOccurred())
	//Expect(network.Client(auditor).IsTxFinal(txID2)).To(HaveOccurred())
	//CheckBalanceAndHolding(network, "alice", "", "PINE", 0, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "PINE", 55, auditor)
	//CheckOwnerDB(network, nil, "issuer", "alice", "bob", "charlie", "manager")
	//CheckAuditorDB(network, auditor, "")
	//
	//// Test Auditor ability to override transaction state
	//txID3, tx3 := PrepareTransferCash(network, "bob", "", "PINE", 10, "alice", auditor, nil)
	//CheckBalance(network, "alice", "", "PINE", 0)
	//CheckHolding(network, "alice", "", "PINE", 10, auditor)
	//CheckBalance(network, "bob", "", "PINE", 55)
	//CheckHolding(network, "bob", "", "PINE", 45, auditor)
	//SetTransactionAuditStatus(network, auditor, txID3, ttx.Deleted)
	//CheckBalanceAndHolding(network, "alice", "", "PINE", 0, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "PINE", 55, auditor)
	//TokenSelectorUnlock(network, "bob", txID3)
	//FinalityWithTimeout(network, "bob", tx3, 20*time.Second)
	//SetTransactionOwnersStatus(network, txID3, ttx.Deleted, "alice", "bob")
	//
	//// Restart
	//CheckOwnerDB(network, nil, "bob", "alice")
	//CheckOwnerDB(network, nil, "issuer", "charlie", "manager")
	//CheckAuditorDB(network, auditor, "")
	//Restart(network, false, "alice", "bob", "charlie", "manager")
	//CheckOwnerDB(network, nil, "bob", "alice")
	//CheckOwnerDB(network, nil, "issuer", "charlie", "manager")
	//CheckAuditorDB(network, auditor, "")
	//
	//// Addition transfers
	//TransferCash(network, "issuer", "", "USD", 50, "issuer", auditor)
	//CheckBalanceAndHolding(network, "issuer", "", "USD", 110, auditor)
	//CheckBalanceAndHolding(network, "issuer", "", "EUR", 150, auditor)
	//
	//TransferCash(network, "issuer", "", "USD", 50, "manager", auditor)
	//TransferCash(network, "issuer", "", "EUR", 20, "manager", auditor)
	//CheckBalanceAndHolding(network, "issuer", "", "USD", 60, auditor)
	//CheckBalanceAndHolding(network, "issuer", "", "EUR", 130, auditor)
	//CheckBalanceAndHolding(network, "manager", "", "USD", 50, auditor)
	//CheckBalanceAndHolding(network, "manager", "", "EUR", 20, auditor)
	//
	//// Play with wallets
	//TransferCash(network, "manager", "", "USD", 10, "manager.id1", auditor)
	//TransferCash(network, "manager", "", "USD", 10, "manager.id2", auditor)
	//TransferCash(network, "manager", "", "USD", 10, "manager.id3", auditor)
	//CheckBalanceAndHolding(network, "manager", "", "USD", 20, auditor)
	//CheckBalanceAndHolding(network, "manager", "manager.id1", "USD", 10, auditor)
	//CheckBalanceAndHolding(network, "manager", "manager.id2", "USD", 10, auditor)
	//CheckBalanceAndHolding(network, "manager", "manager.id3", "USD", 10, auditor)
	//
	//TransferCash(network, "manager", "manager.id1", "USD", 10, "manager.id2", auditor)
	//CheckSpending(network, "manager", "manager.id1", "USD", auditor, 10)
	//CheckBalanceAndHolding(network, "manager", "", "USD", 20, auditor)
	//CheckBalanceAndHolding(network, "manager", "manager.id1", "USD", 0, auditor)
	//CheckBalanceAndHolding(network, "manager", "manager.id2", "USD", 20, auditor)
	//CheckBalanceAndHolding(network, "manager", "manager.id3", "USD", 10, auditor)
	//
	//// Swap among wallets
	//TransferCash(network, "manager", "", "EUR", 10, "manager.id1", auditor)
	//CheckBalanceAndHolding(network, "manager", "", "EUR", 10, auditor)
	//CheckBalanceAndHolding(network, "manager", "manager.id1", "EUR", 10, auditor)
	//
	//SwapCash(network, "manager", "manager.id1", "EUR", 10, "USD", 10, "manager.id2", auditor)
	//CheckBalanceAndHolding(network, "manager", "", "USD", 20, auditor)
	//CheckBalanceAndHolding(network, "manager", "", "EUR", 10, auditor)
	//CheckBalanceAndHolding(network, "manager", "manager.id1", "USD", 10, auditor)
	//CheckBalanceAndHolding(network, "manager", "manager.id1", "EUR", 0, auditor)
	//CheckBalanceAndHolding(network, "manager", "manager.id2", "USD", 10, auditor)
	//CheckBalanceAndHolding(network, "manager", "manager.id2", "EUR", 10, auditor)
	//CheckBalanceAndHolding(network, "manager", "manager.id3", "USD", 10, auditor)
	//
	//// no more USD can be issued, reached quota of 220
	//IssueCash(network, "", "USD", 10, "alice", auditor, true, "issuer", "no more USD can be issued, reached quota of 241")
	//
	//CheckBalanceAndHolding(network, "alice", "", "USD", 0, auditor)
	//CheckBalanceAndHolding(network, "alice", "", "EUR", 10, auditor)
	//
	//// limits
	//CheckBalanceAndHolding(network, "alice", "", "USD", 0, auditor)
	//CheckBalanceAndHolding(network, "alice", "", "EUR", 10, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "EUR", 20, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "USD", 110, auditor)
	//IssueCash(network, "", "EUR", 2200, "alice", auditor, true, "issuer")
	//IssueCash(network, "", "EUR", 2000, "charlie", auditor, true, "issuer")
	//CheckBalanceAndHolding(network, "alice", "", "EUR", 2210, auditor)
	//CheckBalanceAndHolding(network, "charlie", "", "EUR", 2000, auditor)
	//TransferCash(network, "alice", "", "EUR", 210, "bob", auditor, "payment limit reached", "alice", "[EUR][210]")
	//CheckBalanceAndHolding(network, "bob", "", "USD", 110, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "EUR", 20, auditor)
	//
	//PruneInvalidUnspentTokens(network, "issuer", auditor, "alice", "bob", "charlie", "manager")
	//
	//TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	//TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	//TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	//TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	//TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	//TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	//TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	//TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	//TransferCash(network, "alice", "", "EUR", 200, "bob", auditor)
	//CheckBalanceAndHolding(network, "bob", "", "EUR", 1820, auditor)
	//CheckSpending(network, "alice", "", "EUR", auditor, 1800)
	//TransferCash(network, "alice", "", "EUR", 200, "bob", auditor, "cumulative payment limit reached", "alice", "[EUR][2000]")
	//
	//TransferCash(network, "charlie", "", "EUR", 200, "bob", auditor)
	//PruneInvalidUnspentTokens(network, "issuer", auditor, "alice", "bob", "charlie", "manager")
	//TransferCash(network, "charlie", "", "EUR", 200, "bob", auditor)
	//TransferCash(network, "charlie", "", "EUR", 200, "bob", auditor)
	//TransferCash(network, "charlie", "", "EUR", 200, "bob", auditor)
	//TransferCash(network, "charlie", "", "EUR", 200, "bob", auditor)
	//CheckBalanceAndHolding(network, "charlie", "", "EUR", 1000, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "EUR", 2820, auditor)
	//TransferCash(network, "charlie", "", "EUR", 200, "bob", auditor, "holding limit reached", "bob", "[EUR][3020]")
	//CheckBalanceAndHolding(network, "bob", "", "EUR", 2820, auditor)
	//
	//PruneInvalidUnspentTokens(network, "issuer", auditor, "alice", "bob", "charlie", "manager")
	//
	//// Routing
	//IssueCash(network, "", "EUR", 10, "alice.id1", auditor, true, "issuer")
	//CheckAcceptedTransactions(network, "alice", "alice.id1", AliceID1AcceptedTransactions[:], nil, nil, nil)
	//TransferCash(network, "alice", "alice.id1", "EUR", 10, "bob.id1", auditor)
	//CheckBalanceAndHolding(network, "alice", "alice.id1", "EUR", 0, auditor)
	//CheckBalanceAndHolding(network, "bob", "bob.id1", "EUR", 10, auditor)
	//
	//// Concurrent transfers
	//transferErrors := make([]chan error, 5)
	//var sum uint64
	//for i := range transferErrors {
	//	transferErrors[i] = make(chan error, 1)
	//
	//	transfer := transferErrors[i]
	//	r, err := rand.Int(rand.Reader, big.NewInt(200))
	//	Expect(err).ToNot(HaveOccurred())
	//	v := r.Uint64() + 1
	//	sum += v
	//	go func() {
	//		_, err := network.Client("bob").CallView("transferWithSelector", common.JSONMarshall(&views.Transfer{
	//			Auditor:      auditor,
	//			Wallet:       "",
	//			Type:         "EUR",
	//			Amount:       v,
	//			Recipient:    network.Identity("alice"),
	//			RecipientEID: "alice",
	//			Retry:        true,
	//		}))
	//		if err != nil {
	//			transfer <- err
	//			return
	//		}
	//		transfer <- nil
	//	}()
	//}
	//for _, transfer := range transferErrors {
	//	err := <-transfer
	//	Expect(err).ToNot(HaveOccurred())
	//}
	//CheckBalanceAndHolding(network, "bob", "", "EUR", 2820-sum, auditor)
	//
	//// Transfer With Selector
	//IssueCash(network, "", "YUAN", 17, "alice", auditor, true, "issuer")
	//TransferCashWithSelector(network, "alice", "", "YUAN", 10, "bob", auditor)
	//CheckBalanceAndHolding(network, "alice", "", "YUAN", 7, auditor)
	//CheckBalanceAndHolding(network, "bob", "", "YUAN", 10, auditor)
	//TransferCashWithSelector(network, "alice", "", "YUAN", 10, "bob", auditor, "pineapple", "insufficient funds")
	//
	//// Now, the tests asks Bob to transfer to Charlie 14 YUAN split in two parallel transactions each one transferring 7 YUAN.
	//// Notice that Bob has only 10 YUAN, therefore bob will be able to assemble only one transfer.
	//// We use two channels to collect the results of the two transfers.
	//transferErrors = make([]chan error, 2)
	//for i := range transferErrors {
	//	transferErrors[i] = make(chan error, 1)
	//
	//	transferError := transferErrors[i]
	//	go func() {
	//		txid, err := network.Client("bob").CallView("transferWithSelector", common.JSONMarshall(&views.Transfer{
	//			Auditor:      auditor,
	//			Wallet:       "",
	//			Type:         "YUAN",
	//			Amount:       7,
	//			Recipient:    network.Identity("charlie"),
	//			RecipientEID: "charlie",
	//			Retry:        false,
	//		}))
	//		if err != nil {
	//			// The transaction failed, we return the error to the caller.
	//			transferError <- err
	//			return
	//		}
	//		// The transaction didn't fail, let's wait for it to be confirmed, and return no error
	//		Expect(network.Client("charlie").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
	//		transferError <- nil
	//	}()
	//}
	//// collect the errors, and check that they are all nil, and one of them is the error we expect.
	//var errors []error
	//for _, transfer := range transferErrors {
	//	errors = append(errors, <-transfer)
	//}
	//Expect((errors[0] == nil && errors[1] != nil) || (errors[0] != nil && errors[1] == nil)).To(BeTrue())
	//var errStr string
	//if errors[0] == nil {
	//	errStr = errors[1].Error()
	//} else {
	//	errStr = errors[0].Error()
	//}
	//v := strings.Contains(errStr, "pineapple") || strings.Contains(errStr, "lemonade")
	//Expect(v).To(BeEquivalentTo(true))
	//
	//CheckBalanceAndHolding(network, "bob", "", "YUAN", 3, auditor)
	//CheckBalanceAndHolding(network, "alice", "", "YUAN", 7, auditor)
	//CheckBalanceAndHolding(network, "charlie", "", "YUAN", 7, auditor)
	//
	//// Transfer by IDs
	//{
	//	txID1 := IssueCash(network, "", "CHF", 17, "alice", auditor, true, "issuer")
	//	TransferCashByIDs(network, "alice", "", []*token2.ID{{TxId: txID1, Index: 0}}, 17, "bob", auditor, true, "test release")
	//	// the previous call should not keep the token locked if release is successful
	//	txID2 := TransferCashByIDs(network, "alice", "", []*token2.ID{{TxId: txID1, Index: 0}}, 17, "bob", auditor, false)
	//	WhoDeletedToken(network, "alice", []*token2.ID{{TxId: txID1, Index: 0}}, txID2)
	//	WhoDeletedToken(network, auditor, []*token2.ID{{TxId: txID1, Index: 0}}, txID2)
	//	// redeem newly created token
	//	RedeemCashByIDs(network, "bob", "", []*token2.ID{{TxId: txID2, Index: 0}}, 17, auditor)
	//}
	//
	//PruneInvalidUnspentTokens(network, "issuer", auditor, "alice", "bob", "charlie", "manager")
	//
	//// Test Max Token Value
	//IssueCash(network, "", "MAX", 9999, "charlie", auditor, true, "issuer")
	//IssueCash(network, "", "MAX", 9999, "charlie", auditor, true, "issuer")
	//TransferCash(network, "charlie", "", "MAX", 10000, "alice", auditor, "cannot create output with value [10000], max [9999]")
	//IssueCash(network, "", "MAX", 10000, "charlie", auditor, true, "issuer", "q is larger than max token value [9999]")
	//
	//// Check consistency
	//CheckPublicParams(network, "issuer", auditor, "alice", "bob", "charlie", "manager")
	//CheckOwnerDB(network, nil, "bob", "alice", "issuer", "charlie", "manager")
	//CheckAuditorDB(network, auditor, "")
	//PruneInvalidUnspentTokens(network, "issuer", auditor, "alice", "bob", "charlie", "manager")
	//
	//for _, name := range []string{"alice", "bob", "charlie", "manager"} {
	//	IDs := ListVaultUnspentTokens(network, name)
	//	CheckIfExistsInVault(network, auditor, IDs)
	//}

}
