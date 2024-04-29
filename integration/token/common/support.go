/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/views"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	. "github.com/onsi/gomega"
)

func CheckFinality(network *integration.Infrastructure, id string, txID string, tmsID *token.TMSID, fail bool) {
	if len(id) == 0 {
		return
	}
	_, err := network.Client(id).CallView("TxFinality", common.JSONMarshall(&views.TxFinality{
		TxID:  txID,
		TMSID: tmsID,
	}))
	if fail {
		Expect(err).To(HaveOccurred())
	} else {
		Expect(err).NotTo(HaveOccurred())
	}
}
