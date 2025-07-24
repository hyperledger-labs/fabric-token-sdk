/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package update

import (
	"strconv"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	nodepkg "github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	"github.com/hyperledger-labs/fabric-token-sdk/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/fabtokenv1"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/zkatdlognoghv1"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fall"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {
	for _, t := range integration.AllTestTypes {
		Describe("Tokens Upgrade with Auditor ne Issuer", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, 64, []common.TMSOpts{
				{
					Alias:               "dlog-32bits",
					TokenSDKDriver:      zkatdlognoghv1.DriverIdentifier,
					PublicParamsGenArgs: []string{"32"},
				},
			}, t.ReplicationFactor, "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T1"), func() { fungible.TestTokensUpgrade(ts.II, "auditor", nil, selector) })
		})

		Describe("Tokens Local Upgrade with Auditor ne Issuer", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, 32, []common.TMSOpts{
				{
					Alias:               "dlog-32bits",
					TokenSDKDriver:      zkatdlognoghv1.DriverIdentifier,
					PublicParamsGenArgs: []string{"32"},
				},
			}, t.ReplicationFactor, "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T2"), func() { fungible.TestLocalTokensUpgrade(ts.II, "auditor", nil, selector) })
		})

		Describe("Rotate Idemix Issuer", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, 32, []common.TMSOpts{
				{
					Alias:               "dlog-32bits",
					TokenSDKDriver:      zkatdlognoghv1.DriverIdentifier,
					PublicParamsGenArgs: []string{"32"},
				},
				{
					Alias:               "dlog-32bits-bis",
					TokenSDKDriver:      zkatdlognoghv1.DriverIdentifier,
					PublicParamsGenArgs: []string{"32"},
				},
			}, t.ReplicationFactor, "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T3"), func() { fungible.TestIdemixIssuerPublicKeyRotation(ts.II, "auditor", nil, selector) })
		})
	}
})

func newTestSuite(commType fsc.P2PCommunicationType, fabtokenPrecision int, extraTMSs []common.TMSOpts, factor int, names ...string) (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(factor, names...)
	ts := token2.NewTestSuite(StartPortDlog, topology.Topology(common.Opts{
		Backend:         "fabric",
		CommType:        commType,
		SDKs:            []nodepkg.SDK{&fall.SDK{}},
		ReplicationOpts: opts,
		DefaultTMSOpts:  common.TMSOpts{TokenSDKDriver: fabtokenv1.DriverIdentifier, PublicParamsGenArgs: []string{strconv.Itoa(fabtokenPrecision)}},
		ExtraTMSs:       extraTMSs,
		// FSCLogSpec:      "token-sdk=debug:fabric-sdk=debug:info",
	}))
	return ts, selector
}
