/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interop

import (
	"crypto"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/query"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/gomega"
)

func registerAuditor(network *integration.Infrastructure, opts ...token.ServiceOption) {
	options, err := token.CompileServiceOptions(opts...)
	Expect(err).NotTo(HaveOccurred())

	_, err = network.Client("auditor").CallView("register", common.JSONMarshall(&views2.RegisterAuditor{
		TMSID: options.TMSID(),
	}))
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

func tmsIssueCash(network *integration.Infrastructure, tmsID token.TMSID, issuer string, wallet string, typ string, amount uint64, receiver string) string {
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

func checkBalance(network *integration.Infrastructure, id string, wallet string, typ string, expected uint64, opts ...token.ServiceOption) {
	b, err := query.NewClient(network.Client(id)).WalletBalance(wallet, typ, opts...)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(b)).To(BeEquivalentTo(1))
	Expect(b[0].Type).To(BeEquivalentTo(typ))
	q, err := token2.ToQuantity(b[0].Quantity, 64)
	Expect(err).NotTo(HaveOccurred())
	expectedQ := token2.NewQuantityFromUInt64(expected)
	Expect(expectedQ.Cmp(q)).To(BeEquivalentTo(0), "[%s]!=[%s]", expected, q)
}

func htlcLock(network *integration.Infrastructure, tmsID token.TMSID, id string, wallet string, typ string, amount uint64, receiver string, deadline time.Duration, hash []byte, hashFunc crypto.Hash, errorMsgs ...string) ([]byte, []byte) {
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
		if len(hash) == 0 {
			Expect(lockResult.PreImage).NotTo(BeNil())
		}
		Expect(lockResult.Hash).NotTo(BeNil())
		if len(hash) != 0 {
			Expect(lockResult.Hash).To(BeEquivalentTo(hash))
		}
		return lockResult.PreImage, lockResult.Hash
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range errorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg))
		}
		time.Sleep(5 * time.Second)
		return nil, nil
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

func htlcClaim(network *integration.Infrastructure, tmsID token.TMSID, id string, wallet string, preImage []byte, errorMsgs ...string) {
	txID, err := network.Client(id).CallView("htlc.claim", common.JSONMarshall(&htlc.Claim{
		TMSID:    tmsID,
		Wallet:   wallet,
		PreImage: preImage,
	}))
	if len(errorMsgs) == 0 {
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Client(id).IsTxFinal(
			common.JSONUnmarshalString(txID),
			api.WithNetwork(tmsID.Network),
			api.WithChannel(tmsID.Channel),
		)).NotTo(HaveOccurred())
	} else {
		Expect(err).To(HaveOccurred())
		for _, msg := range errorMsgs {
			Expect(err.Error()).To(ContainSubstring(msg))
		}
		time.Sleep(5 * time.Second)
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
	time.Sleep(5 * time.Second)
}

func scan(network *integration.Infrastructure, id string, hash []byte, hashFunc crypto.Hash, opts ...token.ServiceOption) {
	options, err := token.CompileServiceOptions(opts...)
	Expect(err).NotTo(HaveOccurred())

	_, err = network.Client(id).CallView("htlc.scan", common.JSONMarshall(&htlc.Scan{
		TMSID:    options.TMSID(),
		Timeout:  30 * time.Second,
		Hash:     hash,
		HashFunc: hashFunc,
	}))
	Expect(err).NotTo(HaveOccurred())
}
