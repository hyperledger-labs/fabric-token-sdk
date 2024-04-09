/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/nft"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {

	Describe("NFT Orion with libp2p", func() {
		var ts = nft.NewTestSuiteLibP2P("orion", StartPortDlog, "fabtoken")
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("succeeded", func() { nft.TestAll(ts.II) })
	})

	Describe("NFT Orion with websockets", func() {
		var ts = nft.NewTestSuiteWebsocket("orion", StartPortDlog, "fabtoken", integration.NoReplication)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("succeeded", func() { nft.TestAll(ts.II) })
	})

	Describe("NFT Orion with websockets and replicas", func() {
		var ts = nft.NewTestSuiteWebsocket("orion", StartPortDlog, "fabtoken", &integration.ReplicationOptions{
			ReplicationFactors: map[string]int{
				"alice": 3,
				"bob":   2,
			},
			SQLConfigs: map[string]*sql.PostgresConfig{
				"alice": sql.DefaultConfig("alice-db"),
				"bob":   sql.DefaultConfig("bob-db"),
			},
		})
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("succeeded", func() { nft.TestAllWithReplicas(ts.II) })
	})

})
