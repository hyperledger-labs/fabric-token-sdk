/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interop

import (
	"crypto"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/gomega"
)

func RegisterAuditor(network *integration.Infrastructure, opts ...token.ServiceOption) {
	options, err := token.CompileServiceOptions(opts...)
	Expect(err).NotTo(HaveOccurred())

	_, err = network.Client("auditor").CallView("registerAuditor", common.JSONMarshall(&views2.RegisterAuditor{
		TMSID: options.TMSID(),
	}))
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

func IssueCashWithTMS(network *integration.Infrastructure, tmsID token.TMSID, issuer string, wallet string, typ string, amount uint64, receiver string) string {
	txid, err := network.Client(issuer).CallView("issue", common.JSONMarshall(&views2.IssueCash{
		TMSID:        tmsID,
		IssuerWallet: wallet,
		TokenType:    typ,
		Quantity:     amount,
		Recipient:    network.Identity(receiver),
	}))
	Expect(err).NotTo(HaveOccurred())
	Expect(network.Client(receiver).IsTxFinal(
		common.JSONUnmarshalString(txid),
		api.WithNetwork(tmsID.Network),
		api.WithChannel(tmsID.Channel),
	)).NotTo(HaveOccurred())
	Expect(network.Client("auditor").IsTxFinal(
		common.JSONUnmarshalString(txid),
		api.WithNetwork(tmsID.Network),
		api.WithChannel(tmsID.Channel),
	)).NotTo(HaveOccurred())

	return common.JSONUnmarshalString(txid)
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

func CheckBalance(network *integration.Infrastructure, id string, wallet string, typ string, expected uint64, opts ...token.ServiceOption) {
	options, err := token.CompileServiceOptions(opts...)
	Expect(err).NotTo(HaveOccurred())
	res, err := network.Client(id).CallView("balance", common.JSONMarshall(&views2.Balance{
		Wallet: wallet,
		Type:   typ,
		TMSID: token.TMSID{
			Network:   options.Network,
			Channel:   options.Channel,
			Namespace: options.Namespace,
		},
	}))
	Expect(err).NotTo(HaveOccurred())
	b := &views2.BalanceResult{}
	common.JSONUnmarshal(res.([]byte), b)
	Expect(b.Type).To(BeEquivalentTo(typ))
	q, err := token2.ToQuantity(b.Quantity, 64)
	Expect(err).NotTo(HaveOccurred())
	expectedQ := token2.NewQuantityFromUInt64(expected)
	Expect(expectedQ.Cmp(q)).To(BeEquivalentTo(0), "[%s]!=[%s]", expected, q)
}

func CheckHolding(network *integration.Infrastructure, id string, wallet string, typ string, expected int64, opts ...token.ServiceOption) {
	opt, err := token.CompileServiceOptions(opts...)
	Expect(err).NotTo(HaveOccurred(), "failed to compile options [%v]", opts)
	eIDBoxed, err := network.Client(id).CallView("GetEnrollmentID", common.JSONMarshall(&views.GetEnrollmentID{
		Wallet: wallet,
		TMSID:  opt.TMSID(),
	}))
	Expect(err).NotTo(HaveOccurred())
	eID := common.JSONUnmarshalString(eIDBoxed)
	holdingBoxed, err := network.Client("auditor").CallView("holding", common.JSONMarshall(&views.CurrentHolding{
		EnrollmentID: eID,
		TokenType:    typ,
		TMSID:        opt.TMSID(),
	}))
	Expect(err).NotTo(HaveOccurred())
	holding, err := strconv.Atoi(common.JSONUnmarshalString(holdingBoxed))
	Expect(err).NotTo(HaveOccurred())
	Expect(holding).To(Equal(int(expected)))
}

func CheckBalanceWithLocked(network *integration.Infrastructure, id string, wallet string, typ string, expected uint64, expectedLocked uint64, expectedExpired uint64, opts ...token.ServiceOption) {
	opt, err := token.CompileServiceOptions(opts...)
	Expect(err).NotTo(HaveOccurred(), "failed to compile options [%v]", opts)
	resBoxed, err := network.Client(id).CallView("balance", common.JSONMarshall(&views2.Balance{
		Wallet: wallet,
		Type:   typ,
		TMSID:  opt.TMSID(),
	}))
	Expect(err).NotTo(HaveOccurred())
	result := &views2.BalanceResult{}
	common.JSONUnmarshal(resBoxed.([]byte), result)
	Expect(err).NotTo(HaveOccurred())

	balance, err := strconv.Atoi(result.Quantity)
	Expect(err).NotTo(HaveOccurred())
	locked, err := strconv.Atoi(result.Locked)
	Expect(err).NotTo(HaveOccurred())
	expired, err := strconv.Atoi(result.Expired)
	Expect(err).NotTo(HaveOccurred())

	Expect(balance).To(Equal(int(expected)), "expected [%d], got [%d]", expected, balance)
	Expect(locked).To(Equal(int(expectedLocked)), "expected locked [%d], got [%d]", expectedLocked, locked)
	Expect(expired).To(Equal(int(expectedExpired)), "expected expired [%d], got [%d]", expectedExpired, expired)
}

func CheckBalanceAndHolding(network *integration.Infrastructure, id string, wallet string, typ string, expected uint64, opts ...token.ServiceOption) {
	CheckBalance(network, id, wallet, typ, expected, opts...)
	CheckHolding(network, id, wallet, typ, int64(expected), opts...)
}

func CheckBalanceWithLockedAndHolding(network *integration.Infrastructure, id string, wallet string, typ string, expectedBalance uint64, expectedLocked uint64, expectedExpired uint64, expectedHolding int64, opts ...token.ServiceOption) {
	CheckBalanceWithLocked(network, id, wallet, typ, expectedBalance, expectedLocked, expectedExpired, opts...)
	if expectedHolding == -1 {
		expectedHolding = int64(expectedBalance + expectedLocked + expectedExpired)
	}
	CheckHolding(network, id, wallet, typ, expectedHolding, opts...)
}

func CheckPublicParams(network *integration.Infrastructure, tmsID token.TMSID, ids ...string) {
	for _, id := range ids {
		_, err := network.Client(id).CallView("CheckPublicParamsMatch", common.JSONMarshall(&views.CheckPublicParamsMatch{
			TMSID: tmsID,
		}))
		Expect(err).NotTo(HaveOccurred())
	}
}

func CheckOwnerDB(network *integration.Infrastructure, tmsID token.TMSID, expectedErrors []string, ids ...string) {
	for _, id := range ids {
		errorMessagesBoxed, err := network.Client(id).CallView("CheckTTXDB", common.JSONMarshall(&views.CheckTTXDB{
			TMSID: tmsID,
		}))
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

func CheckAuditorDB(network *integration.Infrastructure, tmsID token.TMSID, auditorID string, walletID string, errorCheck func([]string) error) {
	errorMessagesBoxed, err := network.Client(auditorID).CallView("CheckTTXDB", common.JSONMarshall(&views.CheckTTXDB{
		Auditor:         true,
		AuditorWalletID: walletID,
		TMSID:           tmsID,
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

func PruneInvalidUnspentTokens(network *integration.Infrastructure, tmsID token.TMSID, ids ...string) {
	for _, id := range ids {
		eIDBoxed, err := network.Client(id).CallView("PruneInvalidUnspentTokens", common.JSONMarshall(&views.PruneInvalidUnspentTokens{TMSID: tmsID}))
		Expect(err).NotTo(HaveOccurred())

		var deleted []*token2.ID
		common.JSONUnmarshal(eIDBoxed.([]byte), &deleted)
		Expect(len(deleted)).To(BeZero())
	}
}

func ListVaultUnspentTokens(network *integration.Infrastructure, tmsID token.TMSID, id string) []*token2.ID {
	res, err := network.Client(id).CallView("ListVaultUnspentTokens", common.JSONMarshall(&views.ListVaultUnspentTokens{TMSID: tmsID}))
	Expect(err).NotTo(HaveOccurred())

	unspentTokens := &token2.UnspentTokens{}
	common.JSONUnmarshal(res.([]byte), unspentTokens)
	count := unspentTokens.Count()
	var IDs []*token2.ID
	for i := 0; i < count; i++ {
		tok := unspentTokens.At(i)
		IDs = append(IDs, tok.Id)
	}
	return IDs
}

func CheckIfExistsInVault(network *integration.Infrastructure, tmsID token.TMSID, id string, tokenIDs []*token2.ID) {
	_, err := network.Client(id).CallView("CheckIfExistsInVault", common.JSONMarshall(&views.CheckIfExistsInVault{TMSID: tmsID, IDs: tokenIDs}))
	Expect(err).NotTo(HaveOccurred())
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

func htlcLock(network *integration.Infrastructure, tmsID token.TMSID, id string, wallet string, typ string, amount uint64, receiver string, deadline time.Duration, hash []byte, hashFunc crypto.Hash, errorMsgs ...string) (string, []byte, []byte) {
	result, err := network.Client(id).CallView("htlc.lock", common.JSONMarshall(&htlc.Lock{
		TMSID:               tmsID,
		ReclamationDeadline: deadline,
		Wallet:              wallet,
		Type:                typ,
		Amount:              amount,
		Recipient:           network.Identity(receiver),
		Hash:                hash,
		HashFunc:            hashFunc,
	}))
	if len(errorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		lockResult := &htlc.LockInfo{}
		common.JSONUnmarshal(result.([]byte), lockResult)

		Expect(network.Client(receiver).IsTxFinal(
			lockResult.TxID,
			api.WithNetwork(tmsID.Network),
			api.WithChannel(tmsID.Channel),
		)).NotTo(HaveOccurred())
		Expect(network.Client("auditor").IsTxFinal(
			lockResult.TxID,
			api.WithNetwork(tmsID.Network),
			api.WithChannel(tmsID.Channel),
		)).NotTo(HaveOccurred())

		if len(hash) == 0 {
			Expect(lockResult.PreImage).NotTo(BeNil())
		}
		Expect(lockResult.Hash).NotTo(BeNil())
		if len(hash) != 0 {
			Expect(lockResult.Hash).To(BeEquivalentTo(hash))
		}
		return lockResult.TxID, lockResult.PreImage, lockResult.Hash
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range errorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg))
		}
		time.Sleep(5 * time.Second)

		errMsg := err.Error()
		fmt.Printf("Got error message [%s]\n", errMsg)
		txID := ""
		index := strings.Index(err.Error(), "<<<[")
		if index != -1 {
			txID = errMsg[index+4 : index+strings.Index(err.Error()[index:], "]>>>")]
		}
		fmt.Printf("Got error message, extracted tx id [%s]\n", txID)
		return txID, nil, nil
	}
}

func htlcReclaimAll(network *integration.Infrastructure, id string, wallet string, errorMsgs ...string) {
	txID, err := network.Client(id).CallView("htlc.reclaimAll", common.JSONMarshall(&htlc.ReclaimAll{
		Wallet: wallet,
	}))
	if len(errorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Client(id).IsTxFinal(common.JSONUnmarshalString(txID))).NotTo(HaveOccurred())
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range errorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg))
		}
		time.Sleep(5 * time.Second)
	}
}

func htlcClaim(network *integration.Infrastructure, tmsID token.TMSID, id string, wallet string, preImage []byte, errorMsgs ...string) string {
	txIDBoxed, err := network.Client(id).CallView("htlc.claim", common.JSONMarshall(&htlc.Claim{
		TMSID:    tmsID,
		Wallet:   wallet,
		PreImage: preImage,
	}))
	if len(errorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		txID := common.JSONUnmarshalString(txIDBoxed)
		Expect(network.Client(id).IsTxFinal(
			txID,
			api.WithNetwork(tmsID.Network),
			api.WithChannel(tmsID.Channel),
		)).NotTo(HaveOccurred())
		Expect(network.Client("auditor").IsTxFinal(
			txID,
			api.WithNetwork(tmsID.Network),
			api.WithChannel(tmsID.Channel),
		)).NotTo(HaveOccurred())
		return txID
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range errorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg))
		}
		time.Sleep(5 * time.Second)

		errMsg := err.Error()
		fmt.Printf("Got error message [%s]\n", errMsg)
		txID := ""
		index := strings.Index(err.Error(), "<<<[")
		if index != -1 {
			txID = errMsg[index+4 : index+strings.Index(err.Error()[index:], "]>>>")]
		}
		fmt.Printf("Got error message, extracted tx id [%s]\n", txID)
		return txID
	}
}

func fastExchange(network *integration.Infrastructure, id string, recipient string, tmsID1 token.TMSID, typ1 string, amount1 uint64, tmsID2 token.TMSID, typ2 string, amount2 uint64, deadline time.Duration) {
	_, err := network.Client(id).CallView("htlc.fastExchange", common.JSONMarshall(&htlc.FastExchange{
		Recipient:           network.Identity(recipient),
		TMSID1:              tmsID1,
		Type1:               typ1,
		Amount1:             amount1,
		TMSID2:              tmsID2,
		Type2:               typ2,
		Amount2:             amount2,
		ReclamationDeadline: deadline,
	}))
	Expect(err).NotTo(HaveOccurred())
	// give time to bob to commit the transaction
	time.Sleep(10 * time.Second)
}

func scan(network *integration.Infrastructure, id string, hash []byte, hashFunc crypto.Hash, startingTransactionID string, opts ...token.ServiceOption) {
	options, err := token.CompileServiceOptions(opts...)
	Expect(err).NotTo(HaveOccurred())

	_, err = network.Client(id).CallView("htlc.scan", common.JSONMarshall(&htlc.Scan{
		TMSID:                 options.TMSID(),
		Timeout:               120 * time.Second,
		Hash:                  hash,
		HashFunc:              hashFunc,
		StartingTransactionID: startingTransactionID,
	}))
	Expect(err).NotTo(HaveOccurred())
}

func scanWithError(network *integration.Infrastructure, id string, hash []byte, hashFunc crypto.Hash, startingTransactionID string, errorMsgs []string, opts ...token.ServiceOption) {
	options, err := token.CompileServiceOptions(opts...)
	Expect(err).NotTo(HaveOccurred())

	_, err = network.Client(id).CallView("htlc.scan", common.JSONMarshall(&htlc.Scan{
		TMSID:                 options.TMSID(),
		Timeout:               30 * time.Second,
		Hash:                  hash,
		HashFunc:              hashFunc,
		StartingTransactionID: startingTransactionID,
	}))
	Expect(err).To(HaveOccurred())
	for _, msg := range errorMsgs {
		Expect(err.Error()).To(ContainSubstring(msg))
	}
}
