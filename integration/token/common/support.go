/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/views"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	. "github.com/onsi/gomega"
)

var logger = logging.MustGetLogger("token-sdk.common")

func CheckFinality(network *integration.Infrastructure, id *token2.NodeReference, txID string, tmsID *token.TMSID, fail bool) {
	if id == nil || len(id.Id()) == 0 {
		return
	}
	_, err := network.Client(id.ReplicaName()).CallView("TxFinality", common.JSONMarshall(&views.TxFinality{
		TxID:    txID,
		TMSID:   tmsID,
		Timeout: 30 * time.Second,
	}))
	if fail {
		Expect(err).To(HaveOccurred())
	} else {
		Expect(err).NotTo(HaveOccurred())
	}
}

func CheckTTXDB(network *integration.Infrastructure, auditor bool, tmsID token.TMSID, expectedErrors []string, ids ...*token2.NodeReference) {
	time.Sleep(10 * time.Second) // TODO: Remove
	for _, id := range ids {
		for _, replicaName := range id.AllNames() {
			Expect(checkTTXDB(network, replicaName, auditor, tmsID, expectedErrors)).To(Succeed())
			//Eventually(checkTTXDB).WithArguments(network, replicaName, auditor, tmsID, expectedErrors).
			//	WithTimeout(20 * time.Second).
			//	ProbeEvery(1 * time.Second).
			//	Should(Succeed())
		}
	}
}

func checkTTXDB(network *integration.Infrastructure, replicaName string, auditor bool, tmsID token.TMSID, expectedErrors []string) error {
	logger.Infof("Calling CheckTTDB for client [%s]", replicaName)
	errorMessagesBoxed, err := network.Client(replicaName).CallView("CheckTTXDB", common.JSONMarshall(&views2.CheckTTXDB{
		Auditor: auditor,
		TMSID:   tmsID,
	}))
	Expect(err).ToNot(HaveOccurred())
	var errorMessages []string
	common.JSONUnmarshal(errorMessagesBoxed.([]byte), &errorMessages)
	Expect(errorMessages).To(HaveLen(len(expectedErrors)))
	if len(expectedErrors) == 0 {
		return nil
	}

	elements := make([]any, len(expectedErrors))
	for i, errorMessage := range expectedErrors {
		elements[i] = ContainSubstring(errorMessage)
	}
	Expect(errorMessages).To(ContainElements(elements...), "cannot find all error messages [%v] in [%v]", expectedErrors, errorMessages)
	return nil
}

func CheckEndorserFinality(network *integration.Infrastructure, id *token2.NodeReference, txID string, tmsID *token.TMSID, fail bool) {
	if id == nil || len(id.Id()) == 0 {
		return
	}
	if tmsID == nil {
		tmsID = &token.TMSID{}
	}
	_, err := network.Client(id.ReplicaName()).CallView("EndorserFinality", common.JSONMarshall(&endorser.Finality{
		TxID:    txID,
		Network: tmsID.Network,
		Channel: tmsID.Channel,
	}))
	if fail {
		Expect(err).To(HaveOccurred())
	} else {
		Expect(err).NotTo(HaveOccurred())
	}
}
