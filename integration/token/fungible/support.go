/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fungible

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/query"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/gomega"
)

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

func SetTransactionOwnersStatus(network *integration.Infrastructure, txID string, txStatus ttx.TxStatus, ids ...string) {
	for _, id := range ids {
		_, err := network.Client(id).CallView("SetTransactionOwnerStatus", common.JSONMarshall(views.SetTransactionOwnerStatus{
			TxID:   txID,
			Status: txStatus,
		}))
		Expect(err).NotTo(HaveOccurred())
	}
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

func CheckPublicParams(network *integration.Infrastructure, ids ...string) {
	for _, id := range ids {
		_, err := network.Client(id).CallView("CheckPublicParamsMatch", common.JSONMarshall(&views.CheckPublicParamsMatch{}))
		Expect(err).NotTo(HaveOccurred())
	}
}

func CheckOwnerDB(network *integration.Infrastructure, expectedErrors []string, ids ...string) {
	for _, id := range ids {
		errorMessagesBoxed, err := network.Client(id).CallView("CheckTTXDB", common.JSONMarshall(&views.CheckTTXDB{}))
		Expect(err).NotTo(HaveOccurred())
		var errorMessages []string
		common.JSONUnmarshal(errorMessagesBoxed.([]byte), &errorMessages)

		Expect(len(errorMessages)).To(Equal(len(expectedErrors)), "expected %d error messages from [%s], got [% v]", len(expectedErrors), id, errorMessages)
		for _, expectedError := range expectedErrors {
			found := false
			for _, message := range errorMessages {
				if message == expectedError {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "cannot find error message [%s] in [% v]", expectedError, errorMessages)
		}
	}
}

func CheckAuditorDB(network *integration.Infrastructure, auditorID string, walletID string) {
	errorMessagesBoxed, err := network.Client(auditorID).CallView("CheckTTXDB", common.JSONMarshall(&views.CheckTTXDB{
		Auditor:         true,
		AuditorWalletID: walletID,
	}))
	Expect(err).NotTo(HaveOccurred())
	var errorMessages []string
	common.JSONUnmarshal(errorMessagesBoxed.([]byte), &errorMessages)
	Expect(len(errorMessages)).To(Equal(0), "expected 0 error messages, got [% v]", errorMessages)
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

func Restart(network *integration.Infrastructure, ids ...string) {
	for _, id := range ids {
		network.StopFSCNode(id)
	}
	time.Sleep(10 * time.Second)
	for _, id := range ids {
		network.StartFSCNode(id)
	}
	time.Sleep(10 * time.Second)
}
