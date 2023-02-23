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
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/orion"
	platform "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/gomega"
)

func RegisterAuditor(network *integration.Infrastructure, id string, onAuditorRestart OnAuditorRestartFunc) {
	_, err := network.Client(id).CallView("registerAuditor", nil)
	Expect(err).NotTo(HaveOccurred())
	if onAuditorRestart != nil {
		onAuditorRestart(network, id)
	}
}

func RegisterCertifier(network *integration.Infrastructure, id string) {
	_, err := network.Client(id).CallView("registerCertifier", nil)
	Expect(err).NotTo(HaveOccurred())
}

func IssueCash(network *integration.Infrastructure, wallet string, typ string, amount uint64, receiver string, auditor string, anonymous bool, IssuerId string, expectedErrorMsgs ...string) string {
	if auditor == "issuer" || auditor == "newIssuer" {
		// the issuer is the auditor, choose default identity
		auditor = ""
	}
	txid, err := network.Client(IssuerId).CallView("issue", common.JSONMarshall(&views.IssueCash{
		Anonymous:    anonymous,
		Auditor:      auditor,
		IssuerWallet: wallet,
		TokenType:    typ,
		Quantity:     amount,
		Recipient:    network.Identity(receiver),
		RecipientEID: receiver,
	}))

	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Client(receiver).IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
		if len(auditor) == 0 {
			Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
		} else {
			Expect(network.Client(auditor).IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
		}
		return common.JSONUnmarshalString(txid)
	}

	Expect(err).To(HaveOccurred())
	for _, msg := range expectedErrorMsgs {
		Expect(err.Error()).To(ContainSubstring(msg), "err [%s] should contain [%s]", err.Error(), msg)
	}
	return ""
}

func CheckAuditedTransactions(network *integration.Infrastructure, auditor string, expected []*ttxdb.TransactionRecord, start *time.Time, end *time.Time) {
	txsBoxed, err := network.Client(auditor).CallView("historyAuditing", common.JSONMarshall(&views.ListAuditedTransactions{
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

func CheckBalanceAndHolding(network *integration.Infrastructure, id string, wallet string, typ string, expected uint64, auditorId string) {
	CheckBalance(network, id, wallet, typ, expected)
	CheckHolding(network, id, wallet, typ, int64(expected), auditorId)
}

func CheckBalance(network *integration.Infrastructure, id string, wallet string, typ string, expected uint64) {
	res, err := network.Client(id).CallView("balance", common.JSONMarshall(&views.BalanceQuery{
		Wallet: wallet,
		Type:   typ,
	}))
	Expect(err).NotTo(HaveOccurred())
	b := &views.Balance{}
	common.JSONUnmarshal(res.([]byte), b)
	Expect(b.Type).To(BeEquivalentTo(typ))
	q, err := token.ToQuantity(b.Quantity, 64)
	Expect(err).NotTo(HaveOccurred())
	expectedQ := token.NewQuantityFromUInt64(expected)
	Expect(expectedQ.Cmp(q)).To(BeEquivalentTo(0), "[%s]!=[%s]", expected, q)
}

func CheckHolding(network *integration.Infrastructure, id string, wallet string, typ string, expected int64, auditorId string) {
	eIDBoxed, err := network.Client(id).CallView("GetEnrollmentID", common.JSONMarshall(&views.GetEnrollmentID{
		Wallet: wallet,
	}))
	Expect(err).NotTo(HaveOccurred())
	eID := common.JSONUnmarshalString(eIDBoxed)
	holdingBoxed, err := network.Client(auditorId).CallView("holding", common.JSONMarshall(&views.CurrentHolding{
		EnrollmentID: eID,
		TokenType:    typ,
	}))
	Expect(err).NotTo(HaveOccurred())
	holding, err := strconv.Atoi(common.JSONUnmarshalString(holdingBoxed))
	Expect(err).NotTo(HaveOccurred())
	Expect(holding).To(Equal(int(expected)))
}

func CheckSpending(network *integration.Infrastructure, id string, wallet string, tokenType string, auditor string, expected uint64) {
	// check spending
	// first get the enrollment id
	eIDBoxed, err := network.Client(id).CallView("GetEnrollmentID", common.JSONMarshall(&views.GetEnrollmentID{
		Wallet: wallet,
	}))
	Expect(err).NotTo(HaveOccurred())
	eID := common.JSONUnmarshalString(eIDBoxed)
	spendingBoxed, err := network.Client(auditor).CallView("spending", common.JSONMarshall(&views.CurrentSpending{
		EnrollmentID: eID,
		TokenType:    tokenType,
	}))
	Expect(err).NotTo(HaveOccurred())
	spending, err := strconv.ParseUint(common.JSONUnmarshalString(spendingBoxed), 10, 64)
	Expect(err).NotTo(HaveOccurred())
	Expect(spending).To(Equal(expected))
}

func ListIssuerHistory(network *integration.Infrastructure, wallet string, typ string) *token.IssuedTokens {
	res, err := network.Client("issuer").CallView("historyIssuedToken", common.JSONMarshall(&views.ListIssuedTokens{
		Wallet:    wallet,
		TokenType: typ,
	}))
	Expect(err).NotTo(HaveOccurred())

	issuedTokens := &token.IssuedTokens{}
	common.JSONUnmarshal(res.([]byte), issuedTokens)
	return issuedTokens
}

func ListUnspentTokens(network *integration.Infrastructure, id string, wallet string, typ string) *token.UnspentTokens {
	res, err := network.Client(id).CallView("history", common.JSONMarshall(&views.ListUnspentTokens{
		Wallet:    wallet,
		TokenType: typ,
	}))
	Expect(err).NotTo(HaveOccurred())

	unspentTokens := &token.UnspentTokens{}
	common.JSONUnmarshal(res.([]byte), unspentTokens)
	return unspentTokens
}

func TransferCash(network *integration.Infrastructure, id string, wallet string, typ string, amount uint64, receiver string, auditor string, expectedErrorMsgs ...string) string {
	txidBoxed, err := network.Client(id).CallView("transfer", common.JSONMarshall(&views.Transfer{
		Auditor:      auditor,
		Wallet:       wallet,
		Type:         typ,
		Amount:       amount,
		Recipient:    network.Identity(receiver),
		RecipientEID: receiver,
	}))
	if len(expectedErrorMsgs) == 0 {
		txID := common.JSONUnmarshalString(txidBoxed)
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Client(receiver).IsTxFinal(txID)).NotTo(HaveOccurred())
		Expect(network.Client("auditor").IsTxFinal(txID)).NotTo(HaveOccurred())

		signers := []string{auditor}
		if !strings.HasPrefix(receiver, id) {
			signers = append(signers, strings.Split(receiver, ".")[0])
		}
		txInfo := GetTransactionInfo(network, id, txID)
		for _, identity := range signers {
			sigma, ok := txInfo.EndorsementAcks[network.Identity(identity).UniqueID()]
			Expect(ok).To(BeTrue(), "identity %s not found in txInfo.EndorsementAcks", identity)
			Expect(sigma).ToNot(BeNil(), "endorsement ack sigma is nil for identity %s", identity)
		}
		Expect(len(txInfo.EndorsementAcks)).To(BeEquivalentTo(len(signers)))
		return txID
	}

	Expect(err).To(HaveOccurred())
	for _, msg := range expectedErrorMsgs {
		Expect(err.Error()).To(ContainSubstring(msg), "err [%s] should contain [%s]", err.Error(), msg)
	}
	time.Sleep(5 * time.Second)
	return ""
}

func PrepareTransferCash(network *integration.Infrastructure, id string, wallet string, typ string, amount uint64, receiver string, auditor string, tokenID *token.ID, expectedErrorMsgs ...string) (string, []byte) {
	transferInput := &views.Transfer{
		Auditor:      auditor,
		Wallet:       wallet,
		Type:         typ,
		Amount:       amount,
		Recipient:    network.Identity(receiver),
		RecipientEID: receiver,
	}
	if tokenID != nil {
		transferInput.TokenIDs = []*token.ID{tokenID}
	}
	txBoxed, err := network.Client(id).CallView("prepareTransfer", common.JSONMarshall(transferInput))
	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range expectedErrorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg), "err [%s] should contain [%s]", err.Error(), msg)
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

func BroadcastPreparedTransferCash(network *integration.Infrastructure, id string, txID string, tx []byte, finality bool, expectedErrorMsgs ...string) {
	_, err := network.Client(id).CallView("broadcastPreparedTransfer", common.JSONMarshall(&views.BroadcastPreparedTransfer{
		Tx:       tx,
		Finality: finality,
	}))
	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	Expect(err).To(HaveOccurred(), "transaction [%s] must have been marked as invalid", txID)
	fmt.Println("Failed to broadcast ", err)
	for _, msg := range expectedErrorMsgs {
		Expect(err.Error()).To(ContainSubstring(msg), "err [%s] should contain [%s]", err.Error(), msg)
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

func TransferCashByIDs(network *integration.Infrastructure, id string, wallet string, ids []*token.ID, amount uint64, receiver string, auditor string, failToRelease bool, expectedErrorMsgs ...string) string {
	txIDBoxed, err := network.Client(id).CallView("transfer", common.JSONMarshall(&views.Transfer{
		Auditor:       auditor,
		Wallet:        wallet,
		Type:          "",
		TokenIDs:      ids,
		Amount:        amount,
		Recipient:     network.Identity(receiver),
		RecipientEID:  receiver,
		FailToRelease: failToRelease,
	}))
	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Client(receiver).IsTxFinal(common.JSONUnmarshalString(txIDBoxed))).NotTo(HaveOccurred())
		Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txIDBoxed))).NotTo(HaveOccurred())
		return common.JSONUnmarshalString(txIDBoxed)
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range expectedErrorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg), "err [%s] should contain [%s]", err.Error(), msg)
		}
		time.Sleep(5 * time.Second)
		return ""
	}
}

func TransferCashWithSelector(network *integration.Infrastructure, id string, wallet string, typ string, amount uint64, receiver string, auditor string, expectedErrorMsgs ...string) {
	txid, err := network.Client(id).CallView("transferWithSelector", common.JSONMarshall(&views.Transfer{
		Auditor:      auditor,
		Wallet:       wallet,
		Type:         typ,
		Amount:       amount,
		Recipient:    network.Identity(receiver),
		RecipientEID: receiver,
	}))
	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Client(receiver).IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
		Expect(network.Client("auditor").IsTxFinal(common.JSONUnmarshalString(txid))).NotTo(HaveOccurred())
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range expectedErrorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg), "err [%s] should contain [%s]", err.Error(), msg)
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

func RedeemCashByIDs(network *integration.Infrastructure, id string, wallet string, ids []*token.ID, amount uint64, auditor string) {
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
		Expect(err).NotTo(HaveOccurred(), "failed to check public params at [%s]", id)
	}
}

func GetTMS(network *integration.Infrastructure, networkName string) *topology.TMS {
	var tms *topology.TMS
	p := network.Ctx.PlatformsByName["token"]
	for _, TMS := range p.(*platform.Platform).Topology.TMSs {
		if TMS.Network == networkName {
			tms = TMS
			break
		}
	}
	Expect(tms).NotTo(BeNil())
	return tms
}

func UpdatePublicParams(network *integration.Infrastructure, publicParams []byte, tms *topology.TMS) {
	p := network.Ctx.PlatformsByName["token"]
	p.(*platform.Platform).UpdatePublicParams(tms, publicParams)
}

func GetPublicParams(network *integration.Infrastructure, id string) []byte {
	pp, err := network.Client(id).CallView("GetPublicParams", common.JSONMarshall(&views.GetPublicParamsViewFactory{}))
	Expect(err).NotTo(HaveOccurred())
	return pp.([]byte)
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

func PruneInvalidUnspentTokens(network *integration.Infrastructure, ids ...string) {
	for _, id := range ids {
		eIDBoxed, err := network.Client(id).CallView("PruneInvalidUnspentTokens", common.JSONMarshall(&views.PruneInvalidUnspentTokens{}))
		Expect(err).NotTo(HaveOccurred())

		var deleted []*token.ID
		common.JSONUnmarshal(eIDBoxed.([]byte), &deleted)
		Expect(len(deleted)).To(BeZero(), "expected 0 tokens to be deleted at [%s], got [%d]", id, len(deleted))
	}
}

func ListVaultUnspentTokens(network *integration.Infrastructure, id string) []*token.ID {
	res, err := network.Client(id).CallView("ListVaultUnspentTokens", common.JSONMarshall(&views.ListVaultUnspentTokens{}))
	Expect(err).NotTo(HaveOccurred())

	unspentTokens := &token.UnspentTokens{}
	common.JSONUnmarshal(res.([]byte), unspentTokens)
	count := unspentTokens.Count()
	var IDs []*token.ID
	for i := 0; i < count; i++ {
		tok := unspentTokens.At(i)
		IDs = append(IDs, tok.Id)
	}
	return IDs
}

func CheckIfExistsInVault(network *integration.Infrastructure, id string, tokenIDs []*token.ID) {
	_, err := network.Client(id).CallView("CheckIfExistsInVault", common.JSONMarshall(&views.CheckIfExistsInVault{IDs: tokenIDs}))
	Expect(err).NotTo(HaveOccurred())
}

func WhoDeletedToken(network *integration.Infrastructure, id string, tokenIDs []*token.ID, txIDs ...string) *views.WhoDeletedTokenResult {
	boxed, err := network.Client(id).CallView("WhoDeletedToken", common.JSONMarshall(&views.WhoDeletedToken{
		TokenIDs: tokenIDs,
	}))
	Expect(err).NotTo(HaveOccurred())

	var result *views.WhoDeletedTokenResult
	common.JSONUnmarshal(boxed.([]byte), &result)
	Expect(len(result.Deleted)).To(BeEquivalentTo(len(tokenIDs)))
	for i, txID := range txIDs {
		Expect(result.Deleted[i]).To(BeTrue())
		Expect(result.Who[i]).To(BeEquivalentTo(txID))
	}
	return result
}

func GetAuditorIdentity(network *integration.Infrastructure, Id string) []byte {
	auditorIdentity, err := network.Client(Id).CallView("GetAuditorWalletIdentity", common.JSONMarshall(&views.GetAuditorWalletIdentityView{GetAuditorWalletIdentity: &views.GetAuditorWalletIdentity{}}))
	Expect(err).NotTo(HaveOccurred())

	auditorId := auditorIdentity.([]byte)
	var aID []byte
	common.JSONUnmarshal(auditorId, &aID)
	return aID
}

func GetIssuerIdentity(network *integration.Infrastructure, Id string) []byte {
	issuerIdentity, err := network.Client(Id).CallView("GetIssuerWalletIdentity", common.JSONMarshall(&views.GetIssuerWalletIdentityView{GetIssuerWalletIdentity: &views.GetIssuerWalletIdentity{}}))
	Expect(err).NotTo(HaveOccurred())

	issuerId := issuerIdentity.([]byte)
	var aID []byte
	common.JSONUnmarshal(issuerId, &aID)
	return aID
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

func Restart(network *integration.Infrastructure, deleteVault bool, ids ...string) {
	for _, id := range ids {
		network.StopFSCNode(id)
	}
	time.Sleep(10 * time.Second)
	if deleteVault {
		for _, id := range ids {
			fn := fabric.Network(network.Ctx, "default")
			if fn != nil {
				fn.DeleteVault(id)
			} else {
				// skip
				on := orion.Network(network.Ctx, "orion")
				if on != nil {
					on.DeleteVault(id)
				} else {
					Expect(false).To(BeTrue(), "neither fabric nor orion network found")
				}
			}
		}
	}
	for _, id := range ids {
		network.StartFSCNode(id)
	}
	time.Sleep(10 * time.Second)
	if deleteVault {
		// Add extra time to wait for the vault to be reconstructed
		time.Sleep(30 * time.Second)
	}
}

func RegisterIssuerWallet(network *integration.Infrastructure, id string, walletID, walletPath string) {
	_, err := network.Client(id).CallView("RegisterIssuerWallet", common.JSONMarshall(&views.RegisterIssuerWallet{
		ID:   walletID,
		Path: walletPath,
	}))
	Expect(err).NotTo(HaveOccurred())
	network.Ctx.SetViewClient(walletPath, network.Client(id))
}

func RegisterOwnerWallet(network *integration.Infrastructure, id string, walletID, walletPath string) {
	_, err := network.Client(id).CallView("RegisterOwnerWallet", common.JSONMarshall(&views.RegisterOwnerWallet{
		ID:   walletID,
		Path: walletPath,
	}))
	Expect(err).NotTo(HaveOccurred())
	network.Ctx.SetViewClient(walletID, network.Client(id))
}
