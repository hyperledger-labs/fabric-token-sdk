/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	fabric "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/nft"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {

	Describe("NFT with libp2p", func() {
		var ts = newTestSuite(fsc.LibP2P, integration.NoReplication)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("succeeded", func() { nft.TestAll(ts.II) })
	})

	Describe("NFT with websockets", func() {
		var ts = newTestSuite(fsc.WebSocket, integration.NoReplication)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("succeeded", func() { nft.TestAll(ts.II) })
	})

	Describe("NFT with websockets and replicas", func() {
		var ts = newTestSuite(fsc.WebSocket, &integration.ReplicationOptions{
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

func newTestSuite(commType fsc.P2PCommunicationType, replicationOpts *integration.ReplicationOptions) *token.TestSuite {
	return token.NewTestSuite(replicationOpts.SQLConfigs, StartPortDlog, nft.Topology(nft.Opts{
		Backend:        "fabric",
		CommType:       commType,
		TokenSDKDriver: "fabtoken",
		SDKs:           []api.SDK{&fabric.SDK{}, &sdk.SDK{}},
		Replication:    &token.ReplicationOptions{ReplicationOptions: replicationOpts},
	}))
}
