/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"slices"

	"github.com/LFDT-Panurus/panurus/token"
	tokenconfig "github.com/LFDT-Panurus/panurus/token/services/config"
	fabricnetwork "github.com/LFDT-Panurus/panurus/token/services/network/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

// EndorserSelectionCheck asks a node to confirm the endorser selection it resolved from
// its configuration, and that the selection keeps other peers able to endorse out.
type EndorserSelectionCheck struct {
	// TMSID identifies the TMS whose endorser selection is checked. Nil selects the
	// node's default TMS.
	TMSID *token.TMSID
	// ExpectedMSPIDs is the selection the node is expected to have resolved.
	ExpectedMSPIDs []string
	// ExcludedMSPIDs names organizations that host peers able to endorse the namespace
	// and that the selection leaves out.
	ExcludedMSPIDs []string
}

// EndorserSelectionCheckView checks, on the node itself and against a live channel, that
// the endorser selection the node resolved from its configuration is the expected one,
// that it yields a usable set of endorsers, and that peers it excludes could have
// endorsed instead.
//
// Running the token flow with a selection in place cannot show this: a namespace policy
// that a single organization can satisfy is also satisfied by the endorsers discovery
// returns when no selection applies, so the flow succeeds either way.
type EndorserSelectionCheckView struct {
	*EndorserSelectionCheck
}

func (v *EndorserSelectionCheckView) Call(context view.Context) (any, error) {
	tms, err := token.GetManagementService(context, ServiceOpts(v.TMSID)...)
	assert.NoError(err, "failed to lookup TMS [%s]", v.TMSID)
	assert.NotNil(tms, "failed to get TMS [%s]", v.TMSID)
	tmsID := tms.ID()

	// Read through the provider the endorsement service uses, so what is asserted is what
	// the node will apply rather than a copy of the test's expectation.
	configService, err := context.GetService(&tokenconfig.Service{})
	assert.NoError(err, "failed getting the token configuration service")
	selection, err := fabricnetwork.NewConfigEndorserSelectionProvider(configService.(*tokenconfig.Service)).
		EndorserSelectionFor(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	assert.NoError(err, "failed resolving the endorser selection for [%s]", tmsID)

	assert.True(sortedEqual(v.ExpectedMSPIDs, selection.MSPIDs),
		"node resolved endorser selection %v for [%s], expected %v", selection.MSPIDs, tmsID, v.ExpectedMSPIDs)
	assert.True(len(v.ExcludedMSPIDs) > 0, "the caller must name the organizations the selection excludes")
	for _, mspID := range v.ExcludedMSPIDs {
		assert.False(slices.Contains(selection.MSPIDs, mspID),
			"[%s] is both selected and expected to be excluded", mspID)
	}

	fns, err := fabric.GetFabricNetworkService(context, tmsID.Network)
	assert.NoError(err, "failed getting fabric network service [%s]", tmsID.Network)
	channel, err := fns.Channel(tmsID.Channel)
	assert.NoError(err, "failed getting channel [%s]", tmsID.Channel)
	chaincode := channel.Chaincode(tmsID.Namespace)

	// The selection leaves a usable set of endorsers, all of them inside it.
	selected, err := chaincode.Discover().WithFilterByMSPIDs(selection.MSPIDs...).Call()
	assert.NoError(err, "failed discovering the endorsers of [%s] under %v", tmsID, selection.MSPIDs)
	assert.True(len(selected) > 0, "selection %v leaves no endorser of [%s]", selection.MSPIDs, tmsID)
	for _, mspID := range mspIDsOf(selected) {
		assert.True(slices.Contains(selection.MSPIDs, mspID),
			"peer of [%s] may endorse although the selection is %v", mspID, selection.MSPIDs)
	}

	// And it excludes peers that would otherwise have endorsed, which is what makes the
	// restriction above more than a tautology. The comparison is against the excluded
	// organizations rather than against unrestricted discovery: discovery returns a
	// minimal set of endorsers satisfying the namespace policy, which under a policy one
	// organization can satisfy is a single organization either way.
	excluded, err := chaincode.Discover().WithFilterByMSPIDs(v.ExcludedMSPIDs...).Call()
	assert.NoError(err, "failed discovering the endorsers of [%s] under %v", tmsID, v.ExcludedMSPIDs)
	assert.True(len(excluded) > 0,
		"selection %v excludes nothing: %v hosts no endorser of [%s]", selection.MSPIDs, v.ExcludedMSPIDs, tmsID)

	return mspIDsOf(selected), nil
}

// sortedEqual reports whether a and b hold the same MSP IDs, in any order.
func sortedEqual(a, b []string) bool {
	x, y := slices.Clone(a), slices.Clone(b)
	slices.Sort(x)
	slices.Sort(y)

	return slices.Equal(x, y)
}

// mspIDsOf returns the distinct MSP IDs of the given peers, sorted.
func mspIDsOf(peers []fabric.DiscoveredPeer) []string {
	mspIDs := make([]string, 0, len(peers))
	for _, peer := range peers {
		if !slices.Contains(mspIDs, peer.MSPID) {
			mspIDs = append(mspIDs, peer.MSPID)
		}
	}
	slices.Sort(mspIDs)

	return mspIDs
}

type EndorserSelectionCheckViewFactory struct{}

func (f *EndorserSelectionCheckViewFactory) NewView(in []byte) (view.View, error) {
	v := &EndorserSelectionCheckView{EndorserSelectionCheck: &EndorserSelectionCheck{}}
	err := json.Unmarshal(in, v.EndorserSelectionCheck)
	assert.NoError(err, "failed unmarshalling input")

	return v, nil
}
