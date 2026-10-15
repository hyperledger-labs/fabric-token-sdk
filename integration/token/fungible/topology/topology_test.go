/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package topology

import (
	"testing"

	"github.com/LFDT-Panurus/panurus/integration/nwo/token"
	fabric2 "github.com/LFDT-Panurus/panurus/integration/nwo/token/fabric"
	"github.com/LFDT-Panurus/panurus/integration/nwo/token/generators/crypto/zkatdlognoghv1"
	tokentopology "github.com/LFDT-Panurus/panurus/integration/nwo/token/topology"
	"github.com/LFDT-Panurus/panurus/integration/token/common"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noReplication is the replication option set of a topology with one instance per node.
type noReplication struct{}

func (noReplication) For(string) []node.Option { return nil }

// tmsOf builds the topology described by opts and returns its single TMS.
func tmsOf(t *testing.T, opts common.Opts) *tokentopology.TMS {
	t.Helper()

	// The topology builders assert with gomega.
	gomega.RegisterTestingT(t)

	opts.Backend = "fabric"
	opts.CommType = fsc.WebSocket
	opts.ReplicationOpts = noReplication{}
	opts.DefaultTMSOpts = common.TMSOpts{TokenSDKDriver: zkatdlognoghv1.DriverIdentifier, Aries: true}

	var tmss []*tokentopology.TMS
	for _, topology := range Topology(opts) {
		if tokenTopology, ok := topology.(*token.Topology); ok {
			tmss = tokenTopology.GetTMSs()
		}
	}
	require.Len(t, tmss, 1)

	return tmss[0]
}

func TestTopologyDefaultsToOneOrgAndNoEndorserSelection(t *testing.T) {
	tms := tmsOf(t, common.Opts{})

	assert.Equal(t, []string{"Org1"}, fabric2.GetOrgs(tms),
		"only Org1 hosts the token chaincode unless the namespace policy spans more orgs")
	assert.Empty(t, fabric2.GetNamespacePolicy(tms), "the default is unanimity over the hosting orgs")
	assert.False(t, fabric2.EndorsersFromOwnOrg(tms),
		"endorser selection is opt-in: an unconfigured topology must keep default discovery")
}

// TestTopologyOwnOrgEndorsersSpanAllOrgs covers the configuration the T15 spec of the dlog
// suite runs: every node pins endorsement to its own organization, which is only
// meaningful if the chaincode is hosted by every organization and the namespace policy can
// be satisfied by one of them alone.
func TestTopologyOwnOrgEndorsersSpanAllOrgs(t *testing.T) {
	const policy = "OR ('Org1MSP.member','Org2MSP.member')"
	tms := tmsOf(t, common.Opts{NamespacePolicy: policy, EndorsersFromOwnOrg: true})

	assert.Equal(t, []string{"Org1", "Org2"}, fabric2.GetOrgs(tms),
		"a namespace policy naming an org whose peers do not host the chaincode can never be satisfied")
	assert.Equal(t, policy, fabric2.GetNamespacePolicy(tms))
	assert.True(t, fabric2.EndorsersFromOwnOrg(tms))
}
