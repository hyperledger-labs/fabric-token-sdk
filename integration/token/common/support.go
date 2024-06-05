/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/views"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	. "github.com/onsi/gomega"
)

func CheckFinality(network *integration.Infrastructure, id *token2.NodeReference, txID string, tmsID *token.TMSID, fail bool) {
	if id == nil || len(id.Id()) == 0 {
		return
	}
	_, err := network.Client(id.ReplicaName()).CallView("TxFinality", common.JSONMarshall(&views.TxFinality{
		TxID:  txID,
		TMSID: tmsID,
	}))
	if fail {
		Expect(err).To(HaveOccurred())
	} else {
		Expect(err).NotTo(HaveOccurred())
	}
}
