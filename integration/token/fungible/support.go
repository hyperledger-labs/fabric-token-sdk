/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fungible

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	topology2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	tplatform "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
)

var (
	RestartEnabled = true
)

type TransactionRecord struct {
	ttxdb.TransactionRecord
	CheckPrevious bool
	CheckNext     bool
}

type Stream interface {
	Recv(m interface{}) error
	Send(m interface{}) error
	Result() ([]byte, error)
}

func RegisterAuditor(network *integration.Infrastructure, auditor *token3.NodeReference) {
	RegisterAuditorForTMSID(network, auditor, nil)
}

func RegisterAuditorForTMSID(network *integration.Infrastructure, auditor *token3.NodeReference, tmsId *token2.TMSID) {
	_, err := network.Client(auditor.ReplicaName()).CallView("registerAuditor", common.JSONMarshall(&views.RegisterAuditor{
		TMSID: tmsId,
	}))
	Expect(err).NotTo(HaveOccurred())
}

func getTmsId(network *integration.Infrastructure, namespace string) *token2.TMSID {
	fabricTopology := getFabricTopology(network)
	Expect(fabricTopology).NotTo(BeNil())
	return &token2.TMSID{
		Network:   fabricTopology.Name(),
		Channel:   fabricTopology.Channels[0].Name,
		Namespace: namespace,
	}
}

func getFabricTopology(network *integration.Infrastructure) *topology2.Topology {
	for _, t := range network.Topologies {
		if t.Type() == "fabric" {
			return t.(*topology2.Topology)
		}
	}
	return nil
}

func IssueCash(network *integration.Infrastructure, wallet string, typ token.Type, amount uint64, receiver *token3.NodeReference, auditor *token3.NodeReference, anonymous bool, issuer *token3.NodeReference, expectedErrorMsgs ...string) string {
	return IssueCashForTMSID(network, wallet, typ, amount, receiver, auditor, anonymous, issuer, nil, expectedErrorMsgs...)
}

func IssueSuccessfulCash(network *integration.Infrastructure, wallet string, typ token.Type, amount uint64, receiver *token3.NodeReference, auditor *token3.NodeReference, anonymous bool, issuer *token3.NodeReference, finalities ...*token3.NodeReference) string {
	return issueCashForTMSID(network, wallet, typ, amount, receiver, auditor, anonymous, issuer, nil, finalities, []string{})
}

func IssueCashForTMSID(network *integration.Infrastructure, wallet string, typ token.Type, amount uint64, receiver *token3.NodeReference, auditor *token3.NodeReference, anonymous bool, issuer *token3.NodeReference, tmsId *token2.TMSID, expectedErrorMsgs ...string) string {
	return issueCashForTMSID(network, wallet, typ, amount, receiver, auditor, anonymous, issuer, tmsId, []*token3.NodeReference{}, expectedErrorMsgs)
}

func issueCashForTMSID(network *integration.Infrastructure, wallet string, typ token.Type, amount uint64, receiver *token3.NodeReference, auditor *token3.NodeReference, anonymous bool, issuer *token3.NodeReference, tmsId *token2.TMSID, endorsers []*token3.NodeReference, expectedErrorMsgs []string) string {
	targetAuditor := auditor.Id()
	if auditor.Id() == "issuer" || auditor.Id() == "newIssuer" {
		// the issuer is the auditor, choose default identity
		targetAuditor = ""
	}
	txIDBoxed, err := network.Client(issuer.ReplicaName()).CallView("issue", common.JSONMarshall(&views.IssueCash{
		Anonymous:    anonymous,
		Auditor:      targetAuditor,
		IssuerWallet: wallet,
		TokenType:    typ,
		Quantity:     amount,
		Recipient:    network.Identity(receiver.Id()),
		RecipientEID: receiver.Id(),
		TMSID:        tmsId,
	}))

	topology.ToOptions(network.FscPlatform.Peers[0].Options).Endorser()
	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		txID := common.JSONUnmarshalString(txIDBoxed)
		for _, n := range []*token3.NodeReference{receiver, auditor} {
			common2.CheckFinality(network, n, txID, tmsId, false)
		}
		// Perform this check only if there is a fabric network
		if getFabricTopology(network) != nil {
			for _, n := range endorsers {
				common2.CheckEndorserFinality(network, n, txID, tmsId, false)
			}
		}
		return common.JSONUnmarshalString(txIDBoxed)
	}

	Expect(err).To(HaveOccurred())
	for _, msg := range expectedErrorMsgs {
		Expect(err.Error()).To(ContainSubstring(msg), "err [%s] should contain [%s]", err.Error(), msg)
	}
	return ""
}

func GetEndorsers(network *integration.Infrastructure, sel *token3.ReplicaSelector) []*token3.NodeReference {
	endorsers := make([]*token3.NodeReference, 0)
	for _, p := range network.FscPlatform.Peers {
		if topology.ToOptions(p.Options).Endorser() {
			endorsers = append(endorsers, sel.Get(p.Name))
		}
	}
	return endorsers
}

func CheckAuditedTransactions(network *integration.Infrastructure, auditor *token3.NodeReference, expected []TransactionRecord, start *time.Time, end *time.Time) {
	txsBoxed, err := network.Client(auditor.ReplicaName()).CallView("historyAuditing", common.JSONMarshall(&views.ListAuditedTransactions{
		From: start,
		To:   end,
	}))
	Expect(err).NotTo(HaveOccurred())
	var txs []*ttxdb.TransactionRecord
	common.JSONUnmarshal(txsBoxed.([]byte), &txs)
	Expect(len(txs)).To(Equal(len(expected)), "expected [%v] transactions, got [%v]. Params: start [%v], end [%v]", expected, txs, start, end)
	for i, tx := range txs {
		fmt.Printf("tx %d: %+v\n", i, tx)
		fmt.Printf("expected %d: %+v\n", i, expected[i])
		txExpected := expected[i]
		Expect(tx.TokenType).To(Equal(txExpected.TokenType), "tx [%d][%s] expected token type [%v], got [%v]", i, tx.TxID, txExpected.TokenType, tx.TokenType)
		Expect(strings.HasPrefix(tx.SenderEID, txExpected.SenderEID)).To(BeTrue(), "tx [%d][%s] expected sender [%v], got [%v]", i, tx.TxID, txExpected.SenderEID, tx.SenderEID)
		Expect(strings.HasPrefix(tx.RecipientEID, txExpected.RecipientEID)).To(BeTrue(), "tx [%d][%s] tx.RecipientEID: %s, txExpected.RecipientEID: %s", i, tx.TxID, tx.RecipientEID, txExpected.RecipientEID)
		Expect(tx.ActionType).To(Equal(txExpected.ActionType), "tx [%d][%s] expected transaction type [%v], got [%v]", i, tx.TxID, txExpected.ActionType, tx.ActionType)
		Expect(tx.Amount).To(Equal(txExpected.Amount), "tx [%d][%s] expected amount [%v], got [%v]", i, tx.TxID, txExpected.Amount, tx.Amount)
		if len(txExpected.TxID) != 0 {
			Expect(txExpected.TxID).To(Equal(tx.TxID), "tx [%d][%s] expected id [%s], got [%s]", i, tx.TxID, txExpected.TxID, tx.TxID)
		}
		Expect(tx.Status).To(Equal(txExpected.Status), "tx [%d][%s] expected status [%v], got [%v]", i, tx.TxID, txExpected.Status, tx.Status)
	}
}

func CheckAcceptedTransactions(network *integration.Infrastructure, id *token3.NodeReference, wallet string, expected []TransactionRecord, start *time.Time, end *time.Time, statuses []ttxdb.TxStatus, actionTypes ...ttxdb.ActionType) {
	eIDBoxed, err := network.Client(id.ReplicaName()).CallView("GetEnrollmentID", common.JSONMarshall(&views.GetEnrollmentID{
		Wallet: wallet,
	}))
	Expect(err).NotTo(HaveOccurred())
	eID := common.JSONUnmarshalString(eIDBoxed)

	params := views.ListAcceptedTransactions{
		SenderWallet:    eID,
		RecipientWallet: eID,
		From:            start,
		To:              end,
		ActionTypes:     actionTypes,
		Statuses:        statuses,
	}
	txsBoxed, err := network.Client(id.ReplicaName()).CallView("acceptedTransactionHistory", common.JSONMarshall(&params))
	Expect(err).NotTo(HaveOccurred())
	var txs []*ttxdb.TransactionRecord
	common.JSONUnmarshal(txsBoxed.([]byte), &txs)
	Expect(len(txs)).To(Equal(len(expected)), "expected [%v] transactions, got [%v]. Params [%v]", expected, txs, params)
	for i, tx := range txs {
		fmt.Printf("tx %d: %+v\n", i, tx)
		fmt.Printf("expected %d: %+v\n", i, expected[i])
		txExpected := expected[i]
		err := matchTransactionRecord(tx, txExpected, i)
		if err != nil {
			if txExpected.CheckNext {
				Expect(matchTransactionRecord(tx, expected[i+1], i+1)).ToNot(HaveOccurred())
				continue
			}
			if txExpected.CheckPrevious {
				Expect(matchTransactionRecord(tx, expected[i-1], i-1)).ToNot(HaveOccurred())
				continue
			}
			Expect(false).To(BeTrue(), err.Error())
		}
	}
}

func matchTransactionRecord(tx *ttxdb.TransactionRecord, txExpected TransactionRecord, i int) error {
	if tx.TokenType != txExpected.TokenType {
		return errors.Errorf("tx [%d] tx.TokenFormat: %s, txExpected.TokenFormat: %s", i, tx.TokenType, txExpected.TokenType)
	}
	if !strings.HasPrefix(tx.SenderEID, txExpected.SenderEID) {
		return errors.Errorf("tx [%d] tx.SenderEID: %s, txExpected.SenderEID: %s", i, tx.SenderEID, txExpected.SenderEID)
	}
	if !strings.HasPrefix(tx.RecipientEID, txExpected.RecipientEID) {
		return errors.Errorf("tx [%d] tx.RecipientEID: %s, txExpected.RecipientEID: %s", i, tx.RecipientEID, txExpected.RecipientEID)
	}
	if tx.Status != txExpected.Status {
		return errors.Errorf("tx [%d] tx.Status: %d, txExpected.Status: %d", i, tx.Status, txExpected.Status)
	}
	if tx.ActionType != txExpected.ActionType {
		return errors.Errorf("tx [%d] tx.ActionType: %d, txExpected.ActionType: %d", i, tx.ActionType, txExpected.ActionType)
	}
	if tx.Amount.Cmp(txExpected.Amount) != 0 {
		return errors.Errorf("tx [%d] tx.Amount: %d, txExpected.Amount: %d", i, tx.Amount, txExpected.Amount)
	}
	return nil
}

func CheckBalanceAndHolding(network *integration.Infrastructure, ref *token3.NodeReference, wallet string, typ token.Type, expected uint64, auditor *token3.NodeReference) {
	CheckBalance(network, ref, wallet, typ, expected)
	CheckHolding(network, ref, wallet, typ, int64(expected), auditor)
}

func CheckBalanceAndHoldingForTMSID(network *integration.Infrastructure, ref *token3.NodeReference, wallet string, typ token.Type, expected uint64, auditor *token3.NodeReference, tmsID *token2.TMSID) {
	CheckBalanceForTMSID(network, ref, wallet, typ, expected, tmsID)
	CheckHoldingForTMSID(network, ref, wallet, typ, int64(expected), auditor, tmsID)
}

func CheckBalance(network *integration.Infrastructure, ref *token3.NodeReference, wallet string, typ token.Type, expected uint64) {
	CheckBalanceForTMSID(network, ref, wallet, typ, expected, nil)
}

func CheckBalanceForTMSID(network *integration.Infrastructure, ref *token3.NodeReference, wallet string, typ token.Type, expected uint64, tmsID *token2.TMSID) {
	res, err := network.Client(ref.ReplicaName()).CallView("balance", common.JSONMarshall(&views.BalanceQuery{
		Wallet: wallet,
		Type:   typ,
		TMSID:  tmsID,
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

func CheckHolding(network *integration.Infrastructure, ref *token3.NodeReference, wallet string, typ token.Type, expected int64, auditor *token3.NodeReference) {
	CheckHoldingForTMSID(network, ref, wallet, typ, expected, auditor, nil)
}

func CheckHoldingForTMSID(network *integration.Infrastructure, ref *token3.NodeReference, wallet string, typ token.Type, expected int64, auditor *token3.NodeReference, tmsID *token2.TMSID) {
	eIDBoxed, err := network.Client(ref.ReplicaName()).CallView("GetEnrollmentID", common.JSONMarshall(&views.GetEnrollmentID{
		Wallet: wallet,
		TMSID:  tmsID,
	}))
	Expect(err).NotTo(HaveOccurred())
	eID := common.JSONUnmarshalString(eIDBoxed)
	holdingBoxed, err := network.Client(auditor.ReplicaName()).CallView("holding", common.JSONMarshall(&views.CurrentHolding{
		EnrollmentID: eID,
		TokenType:    typ,
	}))
	Expect(err).NotTo(HaveOccurred())
	holding, err := strconv.Atoi(common.JSONUnmarshalString(holdingBoxed))
	Expect(err).NotTo(HaveOccurred())
	Expect(holding).To(Equal(int(expected)))
}

func CheckSpending(network *integration.Infrastructure, id *token3.NodeReference, wallet string, tokenType token.Type, auditor *token3.NodeReference, expected uint64) {
	CheckSpendingForTMSID(network, id, wallet, tokenType, auditor, expected, nil)
}

func CheckSpendingForTMSID(network *integration.Infrastructure, id *token3.NodeReference, wallet string, tokenType token.Type, auditor *token3.NodeReference, expected uint64, tmsId *token2.TMSID) {
	// check spending
	// first get the enrollment id
	eIDBoxed, err := network.Client(id.ReplicaName()).CallView("GetEnrollmentID", common.JSONMarshall(&views.GetEnrollmentID{
		Wallet: wallet,
		TMSID:  tmsId,
	}))
	Expect(err).NotTo(HaveOccurred())
	eID := common.JSONUnmarshalString(eIDBoxed)
	spendingBoxed, err := network.Client(auditor.ReplicaName()).CallView("spending", common.JSONMarshall(&views.CurrentSpending{
		EnrollmentID: eID,
		TokenType:    tokenType,
		TMSID:        tmsId,
	}))
	Expect(err).NotTo(HaveOccurred())
	spending, err := strconv.ParseUint(common.JSONUnmarshalString(spendingBoxed), 10, 64)
	Expect(err).NotTo(HaveOccurred())
	Expect(spending).To(Equal(expected))
}

func ListIssuerHistory(network *integration.Infrastructure, wallet string, typ token.Type, issuer *token3.NodeReference) *token.IssuedTokens {
	return ListIssuerHistoryForTMSID(network, wallet, typ, issuer, nil)
}

func ListIssuerHistoryForTMSID(network *integration.Infrastructure, wallet string, typ token.Type, issuer *token3.NodeReference, tmsId *token2.TMSID) *token.IssuedTokens {
	res, err := network.Client(issuer.ReplicaName()).CallView("historyIssuedToken", common.JSONMarshall(&views.ListIssuedTokens{
		Wallet:    wallet,
		TokenType: typ,
		TMSID:     tmsId,
	}))
	Expect(err).NotTo(HaveOccurred())

	issuedTokens := &token.IssuedTokens{}
	common.JSONUnmarshal(res.([]byte), issuedTokens)
	return issuedTokens
}

func ListUnspentTokens(network *integration.Infrastructure, id *token3.NodeReference, wallet string, typ token.Type) *token.UnspentTokens {
	return ListUnspentTokensForTMSID(network, id, wallet, typ, nil)
}

func ListUnspentTokensForTMSID(network *integration.Infrastructure, id *token3.NodeReference, wallet string, typ token.Type, tmsId *token2.TMSID) *token.UnspentTokens {
	res, err := network.Client(id.ReplicaName()).CallView("history", common.JSONMarshall(&views.ListUnspentTokens{
		Wallet:    wallet,
		TokenType: typ,
		TMSID:     tmsId,
	}))
	Expect(err).NotTo(HaveOccurred())

	unspentTokens := &token.UnspentTokens{}
	common.JSONUnmarshal(res.([]byte), unspentTokens)
	return unspentTokens
}

func TransferCash(network *integration.Infrastructure, sender *token3.NodeReference, wallet string, typ token.Type, amount uint64, receiver *token3.NodeReference, auditor *token3.NodeReference, expectedErrorMsgs ...string) string {
	return TransferCashForTMSID(network, sender, wallet, typ, amount, receiver, auditor, nil, expectedErrorMsgs...)
}

func TransferCashForTMSID(network *integration.Infrastructure, sender *token3.NodeReference, wallet string, typ token.Type, amount uint64, receiver *token3.NodeReference, auditor *token3.NodeReference, tmsId *token2.TMSID, expectedErrorMsgs ...string) string {
	txidBoxed, err := network.Client(sender.ReplicaName()).CallView("transfer", common.JSONMarshall(&views.Transfer{
		Auditor:      auditor.Id(),
		Wallet:       wallet,
		Type:         typ,
		Amount:       amount,
		Recipient:    network.Identity(receiver.Id()),
		RecipientEID: receiver.Id(),
		TMSID:        tmsId,
	}))
	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		txID := common.JSONUnmarshalString(txidBoxed)
		common2.CheckFinality(network, receiver, txID, tmsId, false)
		common2.CheckFinality(network, auditor, txID, tmsId, false)

		signers := []string{auditor.Id()}
		if !strings.Contains(receiver.Id(), sender.Id()) {
			signers = append(signers, strings.Split(receiver.Id(), ".")[0])
		}
		txInfo := GetTransactionInfoForTMSID(network, sender, txID, tmsId)
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

func TransferCashNoFinalityCheck(network *integration.Infrastructure, sender *token3.NodeReference, wallet string, typ token.Type, amount uint64, receiver *token3.NodeReference, auditor *token3.NodeReference) {
	_, err := network.Client(sender.ReplicaName()).CallView("transfer", common.JSONMarshall(&views.Transfer{
		Auditor:      auditor.Id(),
		Wallet:       wallet,
		Type:         typ,
		Amount:       amount,
		Recipient:    network.Identity(receiver.Id()),
		RecipientEID: receiver.Id(),
	}))
	Expect(err).NotTo(HaveOccurred())
}

func TransferCashFromExternalWallet(network *integration.Infrastructure, wmp *WalletManagerProvider, websSocket bool, sender *token3.NodeReference, wallet string, typ token.Type, amount uint64, receiver *token3.NodeReference, auditor *token3.NodeReference, expectedErrorMsgs ...string) string {
	// obtain the recipient for the rest
	restRecipient := wmp.RecipientData(sender.Id(), wallet)
	// start the call as a stream
	var stream Stream
	var err error
	input := common.JSONMarshall(&views.Transfer{
		Auditor:                   auditor.Id(),
		Wallet:                    wallet,
		Type:                      typ,
		Amount:                    amount,
		Recipient:                 network.Identity(receiver.Id()),
		RecipientEID:              receiver.Id(),
		SenderChangeRecipientData: restRecipient,
	})
	if websSocket {
		stream, err = network.WebClient(sender.ReplicaName()).StreamCallView("transfer", input)
	} else {
		stream, err = network.Client(sender.ReplicaName()).StreamCallView("transfer", input)
	}
	Expect(err).NotTo(HaveOccurred())

	// Here we handle the sign requests
	client := ttx.NewStreamExternalWalletSignerClient(wmp.SignerProvider(sender.Id(), wallet), stream, 1)
	Expect(client.Respond()).NotTo(HaveOccurred())

	// wait for the completion of the view
	txidBoxed, err := stream.Result()
	if len(expectedErrorMsgs) == 0 {
		txID := common.JSONUnmarshalString(txidBoxed)
		Expect(err).NotTo(HaveOccurred())
		common2.CheckFinality(network, receiver, txID, nil, false)
		common2.CheckFinality(network, auditor, txID, nil, false)

		signers := []string{auditor.Id()}
		if receiver.Id() != sender.Id() {
			signers = append(signers, receiver.Id())
		}
		txInfo := GetTransactionInfo(network, sender, txID)
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

func TransferCashToExternalWallet(network *integration.Infrastructure, wmp *WalletManagerProvider, sender *token3.NodeReference, wallet string, typ token.Type, amount uint64, receiver *token3.NodeReference, receiverWallet string, auditor *token3.NodeReference, expectedErrorMsgs ...string) string {
	// obtain the recipient data for the recipient and register it
	recipientData := wmp.RecipientData(receiver.Id(), receiverWallet)
	RegisterRecipientData(network, receiver, receiverWallet, recipientData)

	// transfer
	var err error
	input := common.JSONMarshall(&views.Transfer{
		Auditor:       auditor.Id(),
		Wallet:        wallet,
		Type:          typ,
		Amount:        amount,
		RecipientEID:  receiver.Id(),
		Recipient:     view.Identity(receiverWallet),
		RecipientData: recipientData,
	})

	txidBoxed, err := network.Client(sender.ReplicaName()).CallView("transfer", input)
	if len(expectedErrorMsgs) == 0 {
		txID := common.JSONUnmarshalString(txidBoxed)
		Expect(err).NotTo(HaveOccurred())
		common2.CheckFinality(network, receiver, txID, nil, false)
		common2.CheckFinality(network, auditor, txID, nil, false)

		signers := []string{auditor.Id()}
		if receiver.Id() != sender.Id() {
			signers = append(signers, receiver.Id())
		}
		txInfo := GetTransactionInfo(network, sender, txID)
		for _, identity := range signers {
			sigma, ok := txInfo.EndorsementAcks[network.Identity(identity).UniqueID()]
			Expect(ok).To(BeTrue(), "identity [%s] not found in txInfo.EndorsementAcks [%v]", identity, txInfo.EndorsementAcks)
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

func TransferCashFromAndToExternalWallet(network *integration.Infrastructure, wmp *WalletManagerProvider, websSocket bool, sender *token3.NodeReference, wallet string, typ token.Type, amount uint64, receiver *token3.NodeReference, receiverWallet string, auditor *token3.NodeReference, expectedErrorMsgs ...string) string {
	// obtain the recipient for the rest
	restRecipient := wmp.RecipientData(sender.Id(), wallet)

	// obtain the recipient data for the recipient and register it
	recipientData := wmp.RecipientData(receiver.Id(), receiverWallet)
	RegisterRecipientData(network, receiver, receiverWallet, recipientData)

	// start the call as a stream
	var stream Stream
	var err error
	input := common.JSONMarshall(&views.Transfer{
		Auditor:                   auditor.Id(),
		Wallet:                    wallet,
		Type:                      typ,
		Amount:                    amount,
		RecipientEID:              receiver.Id(),
		SenderChangeRecipientData: restRecipient,
		Recipient:                 view.Identity(receiverWallet),
		RecipientData:             recipientData,
	})
	if websSocket {
		stream, err = network.WebClient(sender.ReplicaName()).StreamCallView("transfer", input)
	} else {
		stream, err = network.Client(sender.ReplicaName()).StreamCallView("transfer", input)
	}
	Expect(err).NotTo(HaveOccurred())

	// Here we handle the sign requests
	client := ttx.NewStreamExternalWalletSignerClient(wmp.SignerProvider(sender.Id(), wallet), stream, 1)
	Expect(client.Respond()).NotTo(HaveOccurred())

	// wait for the completion of the view
	txidBoxed, err := stream.Result()
	if len(expectedErrorMsgs) == 0 {
		txID := common.JSONUnmarshalString(txidBoxed)
		Expect(err).NotTo(HaveOccurred())
		common2.CheckFinality(network, receiver, txID, nil, false)
		common2.CheckFinality(network, auditor, txID, nil, false)

		signers := []string{auditor.Id()}
		if receiver != sender {
			signers = append(signers, receiver.Id())
		}
		txInfo := GetTransactionInfo(network, sender, txID)
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

func TransferCashMultiActions(network *integration.Infrastructure, sender *token3.NodeReference, wallet string, typ token.Type, amounts []uint64, receivers []*token3.NodeReference, auditor *token3.NodeReference, tokenID *token.ID, expectedErrorMsgs ...string) string {
	Expect(len(amounts) > 1).To(BeTrue())
	Expect(len(receivers)).To(BeEquivalentTo(len(amounts)))
	transfer := &views.Transfer{
		Auditor:      auditor.Id(),
		Wallet:       wallet,
		Type:         typ,
		Amount:       amounts[0],
		Recipient:    network.Identity(receivers[0].Id()),
		RecipientEID: receivers[0].Id(),
		TokenIDs:     []*token.ID{tokenID},
	}
	for i := 1; i < len(amounts); i++ {
		transfer.TransferAction = append(transfer.TransferAction, views.TransferAction{
			Amount:       amounts[i],
			Recipient:    network.Identity(receivers[i].Id()),
			RecipientEID: receivers[i].Id(),
		})
	}

	txidBoxed, err := network.Client(sender.ReplicaName()).CallView("transfer", common.JSONMarshall(transfer))
	if len(expectedErrorMsgs) == 0 {
		txID := common.JSONUnmarshalString(txidBoxed)
		Expect(err).NotTo(HaveOccurred())
		signers := []string{auditor.Id()}

		for _, receiver := range receivers {
			common2.CheckFinality(network, receiver, txID, nil, false)
			if receiver.Id() != sender.Id() {
				signers = append(signers, receiver.Id())
			}
		}
		common2.CheckFinality(network, auditor, txID, nil, false)

		txInfo := GetTransactionInfo(network, sender, txID)
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
	// extract txID from err
	strErr := err.Error()
	s := strings.LastIndex(strErr, "[<<<")
	e := strings.LastIndex(strErr, ">>>]")
	return strErr[s+4 : e]
}

func MaliciousTransferCash(network *integration.Infrastructure, id *token3.NodeReference, wallet string, typ token.Type, amount uint64, receiver *token3.NodeReference, auditor *token3.NodeReference, tmsId *token2.TMSID, expectedErrorMsgs ...string) string {
	txidBoxed, err := network.Client(id.ReplicaName()).CallView("MaliciousTransfer", common.JSONMarshall(&views.Transfer{
		Auditor:      auditor.Id(),
		Wallet:       wallet,
		Type:         typ,
		Amount:       amount,
		Recipient:    network.Identity(receiver.Id()),
		RecipientEID: receiver.Id(),
		TMSID:        tmsId,
	}))
	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		txID := common.JSONUnmarshalString(txidBoxed)
		time.Sleep(5 * time.Second)
		return txID
	}

	Expect(err).To(HaveOccurred())
	for _, msg := range expectedErrorMsgs {
		Expect(err.Error()).To(ContainSubstring(msg), "err [%s] should contain [%s]", err.Error(), msg)
	}
	time.Sleep(5 * time.Second)
	return ""
}

func PrepareTransferCash(network *integration.Infrastructure, sender *token3.NodeReference, wallet string, typ token.Type, amount uint64, receiver *token3.NodeReference, auditor *token3.NodeReference, tokenID *token.ID, expectedErrorMsgs ...string) (string, []byte) {
	transferInput := &views.Transfer{
		Auditor:      auditor.Id(),
		Wallet:       wallet,
		Type:         typ,
		Amount:       amount,
		Recipient:    network.Identity(receiver.Id()),
		RecipientEID: receiver.Id(),
	}
	if tokenID != nil {
		transferInput.TokenIDs = []*token.ID{tokenID}
	}
	txBoxed, err := network.Client(sender.ReplicaName()).CallView("prepareTransfer", common.JSONMarshall(transferInput))
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

func SetTransactionAuditStatus(network *integration.Infrastructure, id *token3.NodeReference, txID string, txStatus ttx.TxStatus) {
	_, err := network.Client(id.ReplicaName()).CallView("SetTransactionAuditStatus", common.JSONMarshall(views.SetTransactionAuditStatus{
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

func TokenSelectorUnlock(network *integration.Infrastructure, id *token3.NodeReference, txID string) {
	_, err := network.Client(id.ReplicaName()).CallView("TokenSelectorUnlock", common.JSONMarshall(views.TokenSelectorUnlock{
		TxID: txID,
	}))
	Expect(err).NotTo(HaveOccurred())
}

func BroadcastPreparedTransferCash(network *integration.Infrastructure, id *token3.NodeReference, txID string, tx []byte, finality bool, expectedErrorMsgs ...string) {
	_, err := network.Client(id.ReplicaName()).CallView("broadcastPreparedTransfer", common.JSONMarshall(&views.BroadcastPreparedTransfer{
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

func FinalityWithTimeout(network *integration.Infrastructure, id *token3.NodeReference, tx []byte, timeout time.Duration) {
	elapsedBoxed, err := network.Client(id.ReplicaName()).CallView("FinalityWithTimeout", common.JSONMarshall(&views.FinalityWithTimeout{
		Tx:      tx,
		Timeout: timeout,
	}))
	Expect(err).NotTo(HaveOccurred())

	elapsed := JSONUnmarshalFloat64(elapsedBoxed)
	Expect(elapsed > timeout.Seconds()).To(BeTrue())
	Expect(elapsed < timeout.Seconds()*2).To(BeTrue())
}

func GetTransactionInfo(network *integration.Infrastructure, id *token3.NodeReference, txnId string) *ttx.TransactionInfo {
	return GetTransactionInfoForTMSID(network, id, txnId, nil)
}

func GetTransactionInfoForTMSID(network *integration.Infrastructure, id *token3.NodeReference, txnId string, tmsId *token2.TMSID) *ttx.TransactionInfo {
	boxed, err := network.Client(id.ReplicaName()).CallView("transactionInfo", common.JSONMarshall(&views.TransactionInfo{
		TransactionID: txnId,
		TMSID:         tmsId,
	}))
	Expect(err).NotTo(HaveOccurred())
	info := &ttx.TransactionInfo{}
	common.JSONUnmarshal(boxed.([]byte), info)
	return info
}

func TransferCashByIDs(network *integration.Infrastructure, ref *token3.NodeReference, wallet string, ids []*token.ID, amount uint64, receiver *token3.NodeReference, auditor *token3.NodeReference, failToRelease bool, expectedErrorMsgs ...string) string {
	txIDBoxed, err := network.Client(ref.ReplicaName()).CallView("transfer", common.JSONMarshall(&views.Transfer{
		Auditor:       auditor.Id(),
		Wallet:        wallet,
		Type:          "",
		TokenIDs:      ids,
		Amount:        amount,
		Recipient:     network.Identity(receiver.Id()),
		RecipientEID:  receiver.Id(),
		FailToRelease: failToRelease,
	}))
	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		txID := common.JSONUnmarshalString(txIDBoxed)
		common2.CheckFinality(network, receiver, txID, nil, false)
		common2.CheckFinality(network, auditor, txID, nil, false)
		return txID
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range expectedErrorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg), "err [%s] should contain [%s]", err.Error(), msg)
		}
		time.Sleep(5 * time.Second)
		return ""
	}
}

func TransferCashWithSelector(network *integration.Infrastructure, sender *token3.NodeReference, wallet string, typ token.Type, amount uint64, receiver *token3.NodeReference, auditor *token3.NodeReference, expectedErrorMsgs ...string) {
	txIDBoxed, err := network.Client(sender.ReplicaName()).CallView("transferWithSelector", common.JSONMarshall(&views.Transfer{
		Auditor:      auditor.Id(),
		Wallet:       wallet,
		Type:         typ,
		Amount:       amount,
		Recipient:    network.Identity(receiver.Id()),
		RecipientEID: receiver.Id(),
	}))
	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		txID := common.JSONUnmarshalString(txIDBoxed)
		common2.CheckFinality(network, receiver, txID, nil, false)
		common2.CheckFinality(network, auditor, txID, nil, false)
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range expectedErrorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg), "err [%s] should contain [%s]", err.Error(), msg)
		}
		time.Sleep(5 * time.Second)
	}
}

func RedeemCash(network *integration.Infrastructure, id *token3.NodeReference, wallet string, typ token.Type, amount uint64, auditor *token3.NodeReference) {
	RedeemCashForTMSID(network, id, wallet, typ, amount, auditor, nil)
}

func RedeemCashForTMSID(network *integration.Infrastructure, id *token3.NodeReference, wallet string, typ token.Type, amount uint64, auditor *token3.NodeReference, tmsID *token2.TMSID) {
	txid, err := network.Client(id.ReplicaName()).CallView("redeem", common.JSONMarshall(&views.Redeem{
		Auditor: auditor.Id(),
		Wallet:  wallet,
		Type:    typ,
		Amount:  amount,
		TMSID:   tmsID,
	}))
	Expect(err).NotTo(HaveOccurred())
	common2.CheckFinality(network, auditor, common.JSONUnmarshalString(txid), tmsID, false)
}

func RedeemCashByIDs(network *integration.Infrastructure, id *token3.NodeReference, wallet string, ids []*token.ID, amount uint64, auditor *token3.NodeReference) {
	txid, err := network.Client(id.ReplicaName()).CallView("redeem", common.JSONMarshall(&views.Redeem{
		Auditor:  auditor.Id(),
		Wallet:   wallet,
		Type:     "",
		TokenIDs: ids,
		Amount:   amount,
	}))
	Expect(err).NotTo(HaveOccurred())
	common2.CheckFinality(network, auditor, common.JSONUnmarshalString(txid), nil, false)
}

func SwapCash(network *integration.Infrastructure, id *token3.NodeReference, wallet string, typeLeft token.Type, amountLeft uint64, typRight token.Type, amountRight uint64, receiver *token3.NodeReference, auditor *token3.NodeReference) {
	txid, err := network.Client(id.ReplicaName()).CallView("swap", common.JSONMarshall(&views.Swap{
		Auditor:         auditor.Id(),
		AliceWallet:     wallet,
		FromAliceType:   typeLeft,
		FromAliceAmount: amountLeft,
		FromBobType:     typRight,
		FromBobAmount:   amountRight,
		Bob:             network.Identity(receiver.Id()),
	}))
	Expect(err).NotTo(HaveOccurred())
	txID := common.JSONUnmarshalString(txid)
	common2.CheckFinality(network, id, txID, nil, false)
	common2.CheckFinality(network, receiver, txID, nil, false)
	common2.CheckFinality(network, auditor, txID, nil, false)
}

func CheckPublicParams(network *integration.Infrastructure, ids ...*token3.NodeReference) {
	CheckPublicParamsForTMSID(network, nil, ids...)
}

func GetTXStatus(network *integration.Infrastructure, id *token3.NodeReference, txID string) *views.TxStatusResponse {
	boxed, err := network.Client(id.ReplicaName()).CallView("TxStatus", common.JSONMarshall(&views.TxStatus{
		TMSID: token2.TMSID{},
		TxID:  txID,
	}))
	Expect(err).NotTo(HaveOccurred(), "failed to check public params at [%s]", id)
	response := &views.TxStatusResponse{}
	common.JSONUnmarshal(boxed.([]byte), response)
	return response
}

func CheckPublicParamsForTMSID(network *integration.Infrastructure, tmsId *token2.TMSID, ids ...*token3.NodeReference) {
	for _, id := range ids {
		for _, replicaName := range id.AllNames() {
			if network.Client(replicaName) == nil {
				panic("did not find id " + replicaName)
			}
			_, err := network.Client(replicaName).CallView("CheckPublicParamsMatch", common.JSONMarshall(&views.CheckPublicParamsMatch{
				TMSID: tmsId,
			}))
			Expect(err).NotTo(HaveOccurred(), "failed to check public params at [%s]", id)
		}
	}
}

func CheckPublicParamsMatch(network *integration.Infrastructure, tmsId *token2.TMSID, ids ...*token3.NodeReference) bool {
	for _, id := range ids {
		for _, replicaName := range id.AllNames() {
			_, err := network.Client(replicaName).CallView("CheckPublicParamsMatch", common.JSONMarshall(&views.CheckPublicParamsMatch{
				TMSID: tmsId,
			}))
			if err != nil {
				return false
			}
		}
	}
	return true
}

func GetTMSByNetworkName(network *integration.Infrastructure, networkName string) *topology.TMS {
	tp := tplatform.GetPlatform(network.Ctx, "token")
	Expect(tp).NotTo(BeNil())
	for _, TMS := range tp.GetTopology().TMSs {
		if TMS.Network == networkName {
			return TMS
		}
	}
	panic(fmt.Sprintf("TMS not found for network [%s]", networkName))
}

func GetTMSByAlias(network *integration.Infrastructure, alias topology.TMSAlias) *topology.TMS {
	tp := tplatform.GetPlatform(network.Ctx, "token")
	Expect(tp).NotTo(BeNil())
	for _, TMS := range tp.GetTopology().TMSs {
		if TMS.Alias == alias {
			return TMS
		}
	}
	panic(fmt.Sprintf("TMS not found for alias [%s]", alias))
}

func UpdatePublicParams(network *integration.Infrastructure, publicParams []byte, tms *topology.TMS) {
	p := network.Ctx.PlatformsByName["token"]
	p.(*tplatform.Platform).UpdatePublicParams(tms, publicParams)
}

func GetPublicParams(network *integration.Infrastructure, id *token3.NodeReference) []byte {
	pp, err := network.Client(id.ReplicaName()).CallView("GetPublicParams", common.JSONMarshall(&views.GetPublicParams{}))
	Expect(err).NotTo(HaveOccurred())
	return pp.([]byte)
}

func DoesWalletExist(network *integration.Infrastructure, id *token3.NodeReference, wallet string, walletType int) bool {
	boxed, err := network.Client(id.ReplicaName()).CallView("DoesWalletExist", common.JSONMarshall(&views.DoesWalletExist{
		Wallet:     wallet,
		WalletType: walletType,
	}))
	Expect(err).NotTo(HaveOccurred())
	var exists bool
	switch v := boxed.(type) {
	case []byte:
		err := json.Unmarshal(v, &exists)
		Expect(err).NotTo(HaveOccurred())
	case string:
		err := json.Unmarshal([]byte(v), &exists)
		Expect(err).NotTo(HaveOccurred())
	default:
		panic(fmt.Sprintf("type not recognized [%T]", v))
	}
	return exists
}

func CheckOwnerDB(network *integration.Infrastructure, expectedErrors []string, ids ...*token3.NodeReference) {
	for _, id := range ids {
		for _, replicaName := range id.AllNames() {
			errorMessagesBoxed, err := network.Client(replicaName).CallView("CheckTTXDB", common.JSONMarshall(&views.CheckTTXDB{}))
			Expect(err).NotTo(HaveOccurred())
			var errorMessages []string
			common.JSONUnmarshal(errorMessagesBoxed.([]byte), &errorMessages)

			Expect(len(errorMessages)).To(Equal(len(expectedErrors)), "expected %d error messages from [%s], got [% v]", len(expectedErrors), replicaName, errorMessages)
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
}

func CheckAuditorDB(network *integration.Infrastructure, auditor *token3.NodeReference, walletID string, errorCheck func([]string) error) {
	errorMessagesBoxed, err := network.Client(auditor.ReplicaName()).CallView("CheckTTXDB", common.JSONMarshall(&views.CheckTTXDB{
		Auditor:         true,
		AuditorWalletID: walletID,
	}))
	Expect(err).NotTo(HaveOccurred())
	if errorCheck != nil {
		var errorMessages []string
		common.JSONUnmarshal(errorMessagesBoxed.([]byte), &errorMessages)
		Expect(errorCheck(errorMessages)).NotTo(HaveOccurred(), "failed to check errors")
	} else {
		var errorMessages []string
		common.JSONUnmarshal(errorMessagesBoxed.([]byte), &errorMessages)
		Expect(len(errorMessages)).To(Equal(0), "expected 0 error messages, got [% v]", errorMessages)
	}
}

func PruneInvalidUnspentTokens(network *integration.Infrastructure, ids ...*token3.NodeReference) {
	for _, id := range ids {
		eIDBoxed, err := network.Client(id.ReplicaName()).CallView("PruneInvalidUnspentTokens", common.JSONMarshall(&views.PruneInvalidUnspentTokens{}))
		Expect(err).NotTo(HaveOccurred())

		var deleted []*token.ID
		common.JSONUnmarshal(eIDBoxed.([]byte), &deleted)
		Expect(len(deleted)).To(BeZero(), "expected 0 tokens to be deleted at [%s], got [%d]", id, len(deleted))
	}
}

func ListVaultUnspentTokens(network *integration.Infrastructure, id *token3.NodeReference) []*token.ID {
	res, err := network.Client(id.ReplicaName()).CallView("ListVaultUnspentTokens", common.JSONMarshall(&views.ListVaultUnspentTokens{}))
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

func CheckIfExistsInVault(network *integration.Infrastructure, id *token3.NodeReference, tokenIDs []*token.ID) {
	_, err := network.Client(id.ReplicaName()).CallView("CheckIfExistsInVault", common.JSONMarshall(&views.CheckIfExistsInVault{IDs: tokenIDs}))
	Expect(err).NotTo(HaveOccurred())
}

func WhoDeletedToken(network *integration.Infrastructure, id *token3.NodeReference, tokenIDs []*token.ID, txIDs ...string) *views.WhoDeletedTokenResult {
	boxed, err := network.Client(id.ReplicaName()).CallView("WhoDeletedToken", common.JSONMarshall(&views.WhoDeletedToken{
		TokenIDs: tokenIDs,
	}))
	Expect(err).NotTo(HaveOccurred())

	var result *views.WhoDeletedTokenResult
	common.JSONUnmarshal(boxed.([]byte), &result)
	Expect(len(result.Deleted)).To(BeEquivalentTo(len(tokenIDs)))
	for i, txID := range txIDs {
		Expect(result.Deleted[i]).To(BeTrue(), "expected token [%s] to be deleted", tokenIDs[i])
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

type deleteVaultPlatform interface {
	DeleteVault(id string)
}

func Restart(network *integration.Infrastructure, deleteVault bool, onRestart OnRestartFunc, ids ...*token3.NodeReference) {
	logger.Infof("restart [%v], [%v]", ids, RestartEnabled)
	if !RestartEnabled {
		return
	}
	for _, id := range ids {
		network.StopFSCNode(id.Id())
	}
	time.Sleep(10 * time.Second)
	if deleteVault {
		for _, id := range ids {
			for name, platform := range network.Ctx.PlatformsByName {
				if dv, ok := platform.(deleteVaultPlatform); ok {
					logger.Infof("Platform %d supports delete vault. Deleting...", name)
					dv.DeleteVault(id.Id())
				}
			}

			// delete token dbs as well
			tokenPlatform := tplatform.GetPlatform(network.Ctx, "token")
			Expect(tokenPlatform).ToNot(BeNil(), "cannot find token platform in context")
			for _, tms := range tokenPlatform.GetTopology().TMSs {
				tokenPlatform.DeleteDBs(tms, id.Id())
			}
		}
	}
	for _, id := range ids {
		network.StartFSCNode(id.Id())
	}
	time.Sleep(10 * time.Second)
	if deleteVault {
		// Add extra time to wait for the vault to be reconstructed
		time.Sleep(40 * time.Second)
	}
	if onRestart != nil {
		for _, id := range ids {
			logger.Infof("Calling on restart for [%s]", id.Id())
			onRestart(network, id.Id())
		}
	}
}

func CopyDBsTo(network *integration.Infrastructure, to string, ids ...*token3.NodeReference) {
	tokenPlatform := tplatform.GetPlatform(network.Ctx, "token")
	Expect(tokenPlatform).ToNot(BeNil(), "cannot find token platform in context")

	for _, id := range ids {
		for _, tms := range tokenPlatform.GetTopology().TMSs {
			tokenPlatform.CopyDBsTo(tms, id.Id(), filepath.Join(to, tms.ID(), id.Id()))
		}
	}
}

func RegisterIssuerIdentity(network *integration.Infrastructure, id *token3.NodeReference, walletID, walletPath string) {
	_, err := network.Client(id.ReplicaName()).CallView("RegisterIssuerIdentity", common.JSONMarshall(&views.RegisterIssuerWallet{
		ID:   walletID,
		Path: walletPath,
	}))
	Expect(err).NotTo(HaveOccurred())
	network.Ctx.SetViewClient(walletPath, network.Client(id.ReplicaName()))
}

func RegisterOwnerIdentity(network *integration.Infrastructure, id *token3.NodeReference, identityConfiguration token2.IdentityConfiguration) {
	for _, replicaName := range id.AllNames() { // TODO: AF
		_, err := network.Client(replicaName).CallView("RegisterOwnerIdentity", common.JSONMarshall(&views.RegisterOwnerIdentity{
			IdentityConfiguration: identityConfiguration,
		}))
		Expect(err).NotTo(HaveOccurred())
		network.Ctx.SetViewClient(identityConfiguration.ID, network.Client(replicaName))
	}
}

func RegisterRecipientData(network *integration.Infrastructure, ref *token3.NodeReference, walletID string, rd *token2.RecipientData) {
	_, err := network.Client(ref.ReplicaName()).CallView("RegisterRecipientData", common.JSONMarshall(&views.RegisterRecipientData{
		WalletID:      walletID,
		RecipientData: *rd,
	}))
	Expect(err).NotTo(HaveOccurred())
	network.Ctx.SetViewClient(walletID, network.Client(ref.ReplicaName()))
}

func CheckOwnerWalletIDs(network *integration.Infrastructure, owner *token3.NodeReference, ids ...string) {
	idsBoxed, err := network.Client(owner.ReplicaName()).CallView("ListOwnerWalletIDsView", nil)
	Expect(err).NotTo(HaveOccurred())
	var wIDs []string
	common.JSONUnmarshal(idsBoxed.([]byte), &wIDs)
	for _, wID := range ids {
		found := false
		for _, expectedWID := range wIDs {
			if expectedWID == wID {
				found = true
				break
			}
		}
		Expect(found).To(BeTrue(), "[%s] is not in [%v]", wID, wIDs)
	}
}

func RevokeIdentity(network *integration.Infrastructure, auditor *token3.NodeReference, rh string) {
	_, err := network.Client(auditor.ReplicaName()).CallView("RevokeUser", common.JSONMarshall(&views.RevokeUser{
		RH: rh,
	}))
	Expect(err).NotTo(HaveOccurred())
}

func GetRevocationHandle(network *integration.Infrastructure, ref *token3.NodeReference) string {
	rhBoxed, err := network.Client(ref.ReplicaName()).CallView("GetRevocationHandle", common.JSONMarshall(&views.GetRevocationHandle{}))
	Expect(err).NotTo(HaveOccurred())
	rh := &views.RevocationHandle{}
	common.JSONUnmarshal(rhBoxed.([]byte), rh)
	fmt.Printf("GetRevocationHandle [%s][%s]", rh.RH, hash.Hashable(rh.RH).String())
	return rh.RH
}

func SetKVSEntry(network *integration.Infrastructure, user *token3.NodeReference, key string, value string) {
	_, err := network.Client(user.ReplicaName()).CallView("SetKVSEntry", common.JSONMarshall(&views.KVSEntry{
		Key:   key,
		Value: value,
	}))
	Expect(err).NotTo(HaveOccurred())
}

func SetSpendableFlag(network *integration.Infrastructure, user *token3.NodeReference, tokenID token.ID, value bool) {
	_, err := network.Client(user.ReplicaName()).CallView("SetSpendableFlag", common.JSONMarshall(&views.SetSpendableFlag{
		TMSID:     token2.TMSID{},
		TokenID:   tokenID,
		Spendable: value,
	}))
	Expect(err).NotTo(HaveOccurred())
}

func Withdraw(network *integration.Infrastructure, wpm *WalletManagerProvider, user *token3.NodeReference, wallet string, typ token.Type, amount uint64, auditor *token3.NodeReference, issuer *token3.NodeReference, expectedErrorMsgs ...string) string {
	var recipientData *token2.RecipientData
	if wpm != nil {
		recipientData = wpm.RecipientData(user.Id(), wallet)
	}
	txid, err := network.Client(user.ReplicaName()).CallView("withdrawal", common.JSONMarshall(&views.Withdrawal{
		Wallet:        wallet,
		TokenType:     typ,
		Amount:        amount,
		Issuer:        issuer.Id(),
		RecipientData: recipientData,
	}))

	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		txID := common.JSONUnmarshalString(txid)
		common2.CheckFinality(network, user, txID, nil, false)
		common2.CheckFinality(network, auditor, txID, nil, false)
		common2.CheckFinality(network, issuer, txID, nil, false)
		return txID
	}

	Expect(err).To(HaveOccurred())
	for _, msg := range expectedErrorMsgs {
		Expect(err.Error()).To(ContainSubstring(msg), "err [%s] should contain [%s]", err.Error(), msg)
	}
	return ""
}

func DisableRestart() {
	RestartEnabled = false
}

func CheckLocalMetrics(ii *integration.Infrastructure, user string, viewName string) {
	metrics, err := ii.WebClient(user).Metrics()
	Expect(err).To(BeNil())
	Expect(metrics).NotTo(BeEmpty())

	var sum float64
	for _, m := range metrics["fsc_view_operations"].GetMetric() {
		for _, labelPair := range m.Label {
			if labelPair.GetName() == "view" && labelPair.GetValue() == viewName {
				sum += m.Counter.GetValue()
			}
		}
	}

	logger.Infof("Received in total %f view operations for [%s] for user %s: %v", sum, viewName, user, metrics["fsc_view_operations"].GetMetric())
	Expect(sum).NotTo(BeZero())
}

func CheckPrometheusMetrics(ii *integration.Infrastructure, viewName string) {
	cli, err := ii.NWO.PrometheusAPI()
	Expect(err).To(BeNil())
	metric := model.Metric{
		"__name__": model.LabelValue("fsc_view_operations"),
		"view":     model.LabelValue(viewName),
	}
	val, warnings, err := cli.Query(context.Background(), metric.String(), time.Now())
	Expect(warnings).To(BeEmpty())
	Expect(err).To(BeNil())
	Expect(val.Type()).To(Equal(model.ValVector))

	logger.Infof("Received prometheus metrics for view [%s]: %s", viewName, val)

	vector, ok := val.(model.Vector)
	Expect(ok).To(BeTrue())
	Expect(vector).NotTo(BeEmpty())
	for _, v := range vector {
		Expect(v.Value).NotTo(Equal(model.SampleValue(0)))
	}
}

func TokensUpgrade(network *integration.Infrastructure, wpm *WalletManagerProvider, user *token3.NodeReference, wallet string, typ token.Type, auditor *token3.NodeReference, issuer *token3.NodeReference, expectedErrorMsgs ...string) string {
	var recipientData *token2.RecipientData
	if wpm != nil {
		recipientData = wpm.RecipientData(user.Id(), wallet)
	}
	txid, err := network.Client(user.ReplicaName()).CallView("TokensUpgrade", common.JSONMarshall(&views.TokensUpgrade{
		Wallet:        wallet,
		TokenType:     typ,
		Issuer:        issuer.Id(),
		RecipientData: recipientData,
	}))

	if len(expectedErrorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		txID := common.JSONUnmarshalString(txid)
		fmt.Printf("TokensUpgrade txID [%s]\n", txID)
		common2.CheckFinality(network, user, txID, nil, false)
		common2.CheckFinality(network, auditor, txID, nil, false)
		common2.CheckFinality(network, issuer, txID, nil, false)
		return txID
	}

	Expect(err).To(HaveOccurred())
	for _, msg := range expectedErrorMsgs {
		Expect(err.Error()).To(ContainSubstring(msg), "err [%s] should contain [%s]", err.Error(), msg)
	}
	return ""
}
