/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	topology2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/multisig/views"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/gomega"
	"strconv"
)

var (
	RestartEnabled = true
)

func RegisterAuditor(network *integration.Infrastructure, auditor *token3.NodeReference) {
	RegisterAuditorForTMSID(network, auditor, nil)
}

func RegisterAuditorForTMSID(network *integration.Infrastructure, auditor *token3.NodeReference, tmsId *token2.TMSID) {
	_, err := network.Client(auditor.ReplicaName()).CallView("registerAuditor", common.JSONMarshall(&views.RegisterAuditor{
		TMSID: tmsId,
	}))
	Expect(err).NotTo(HaveOccurred())
}

func getFabricTopology(network *integration.Infrastructure) *topology2.Topology {
	for _, t := range network.Topologies {
		if t.Type() == "fabric" {
			return t.(*topology2.Topology)
		}
	}
	return nil
}

func CheckBalance(network *integration.Infrastructure, ref *token3.NodeReference, wallet string, typ token.Type, owned uint64, coowned uint64) {
	CheckBalanceForTMSID(network, ref, wallet, typ, owned, coowned, nil)
}

func CheckBalanceForTMSID(network *integration.Infrastructure, ref *token3.NodeReference, wallet string, typ token.Type, owned uint64, coowned uint64, tmsID *token2.TMSID) {
	res, err := network.Client(ref.ReplicaName()).CallView("balance", common.JSONMarshall(&views2.BalanceQuery{
		Wallet: wallet,
		Type:   typ,
		TMSID:  tmsID,
	}))
	Expect(err).NotTo(HaveOccurred())
	b := &views2.BalanceResult{}
	common.JSONUnmarshal(res.([]byte), b)
	Expect(b.Type).To(BeEquivalentTo(typ))
	q, err := token.ToQuantity(b.Quantity, 64)
	Expect(err).NotTo(HaveOccurred())
	expectedQ := token.NewQuantityFromUInt64(owned)
	Expect(expectedQ.Cmp(q)).To(BeEquivalentTo(0), "[%s]!=[%s]", expectedQ, q)

	q, err = token.ToQuantity(b.CoOwned, 64)
	Expect(err).NotTo(HaveOccurred())
	expectedQ = token.NewQuantityFromUInt64(coowned)
	Expect(expectedQ.Cmp(q)).To(BeEquivalentTo(0), "[%s]!=[%s]", expectedQ, q)
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

func LockCash(network *integration.Infrastructure, sender *token3.NodeReference, wallet string, typ token.Type, amount uint64, receivers []*token3.NodeReference, auditor *token3.NodeReference) string {
	return LockCashForTMSID(network, sender, wallet, typ, amount, receivers, auditor, nil)
}

func LockCashForTMSID(network *integration.Infrastructure, sender *token3.NodeReference, wallet string, typ token.Type, amount uint64, receivers []*token3.NodeReference, auditor *token3.NodeReference, tmsId *token2.TMSID) string {
	identities := make([]view.Identity, len(receivers))
	eids := make([]string, len(receivers))
	for i := 0; i < len(receivers); i++ {
		eids[i] = receivers[i].Id()
		identities[i] = network.Identity(eids[i])
	}
	txidBoxed, err := network.Client(sender.ReplicaName()).CallView("lock", common.JSONMarshall(&views2.Lock{
		Auditor:    auditor.Id(),
		Wallet:     wallet,
		Type:       typ,
		Amount:     amount,
		Escrow:     identities,
		EscrowEIDs: eids,
		TMSID:      tmsId,
	}))
	Expect(err).NotTo(HaveOccurred())
	txID := common.JSONUnmarshalString(txidBoxed)
	return txID

}

func IssueCash(network *integration.Infrastructure, wallet string, typ token.Type, amount uint64, receiver, auditor *token3.NodeReference, anonymous bool, issuer *token3.NodeReference, expectedErrorMsgs ...string) string {
	return IssueCashForTMSID(network, wallet, typ, amount, receiver, auditor, anonymous, issuer, nil, expectedErrorMsgs...)
}

func IssueCashForTMSID(network *integration.Infrastructure, wallet string, typ token.Type, amount uint64, receiver, auditor *token3.NodeReference, anonymous bool, issuer *token3.NodeReference, tmsId *token2.TMSID, expectedErrorMsgs ...string) string {
	return issueCashForTMSID(network, wallet, typ, amount, receiver, auditor, anonymous, issuer, tmsId, []*token3.NodeReference{}, expectedErrorMsgs)
}

func issueCashForTMSID(network *integration.Infrastructure, wallet string, typ token.Type, amount uint64, receiver, auditor *token3.NodeReference, anonymous bool, issuer *token3.NodeReference, tmsId *token2.TMSID, endorsers []*token3.NodeReference, expectedErrorMsgs []string) string {

	txIDBoxed, err := network.Client(issuer.ReplicaName()).CallView("issue", common.JSONMarshall(&views.IssueCash{
		Anonymous:    anonymous,
		IssuerWallet: wallet,
		TokenType:    typ,
		Quantity:     amount,
		Recipient:    network.Identity(receiver.Id()),
		RecipientEID: receiver.Id(),
		TMSID:        tmsId,
		Auditor:      auditor.Id(),
	}))
	Expect(err).NotTo(HaveOccurred())
	return common.JSONUnmarshalString(txIDBoxed)
}
