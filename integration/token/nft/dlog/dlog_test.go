/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

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
		var ts = token.NewTestSuite(nil, StartPortDlog, nft.Topology(nft.Opts{
			Backend:        "fabric",
			CommType:       fsc.LibP2P,
			TokenSDKDriver: "dlog",
			SDKs:           []api.SDK{&fabric.SDK{}, &sdk.SDK{}},
		}))
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("succeeded", func() { nft.TestAll(ts.II) })
	})

	Describe("NFT with websockets", func() {
		var ts = token.NewTestSuite(nil, StartPortDlog, nft.Topology(nft.Opts{
			Backend:        "fabric",
			CommType:       fsc.WebSocket,
			TokenSDKDriver: "dlog",
			SDKs:           []api.SDK{&fabric.SDK{}, &sdk.SDK{}},
		}))
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("succeeded", func() { nft.TestAll(ts.II) })
	})

	Describe("NFT with websockets and replicas", func() {
		replicationOpts := &integration.ReplicationOptions{
			ReplicationFactors: map[string]int{
				"alice": 3,
				"bob":   2,
			},
			SQLConfigs: map[string]*sql.PostgresConfig{
				"alice": sql.DefaultConfig("alice-db"),
				"bob":   sql.DefaultConfig("bob-db"),
			},
		}
		var ts = token.NewTestSuite(replicationOpts.SQLConfigs, StartPortDlog, nft.Topology(nft.Opts{
			Backend:        "fabric",
			CommType:       fsc.WebSocket,
			TokenSDKDriver: "dlog",
			SDKs:           []api.SDK{&fabric.SDK{}, &sdk.SDK{}},
			Replication:    &token.ReplicationOptions{ReplicationOptions: replicationOpts},
		}))
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("succeeded", func() { nft.TestAllWithReplicas(ts.II) })
	})
})
