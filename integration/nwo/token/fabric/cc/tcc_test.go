/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cc

import (
	"testing"

	"github.com/LFDT-Panurus/panurus/integration/nwo/token/fabric"
	topology3 "github.com/LFDT-Panurus/panurus/integration/nwo/token/topology"
	"github.com/stretchr/testify/assert"
)

func newTMS() *topology3.TMS {
	return &topology3.TMS{
		Network:       "testnet",
		Channel:       "testchannel",
		Namespace:     "tns",
		BackendParams: map[string]any{},
	}
}

func TestNamespacePolicyDefaultsToUnanimity(t *testing.T) {
	assert.Equal(t, "AND ('Org1MSP.member')", namespacePolicy(newTMS(), []string{"Org1"}))
	assert.Equal(t, "AND ('Org1MSP.member','Org2MSP.member')", namespacePolicy(newTMS(), []string{"Org1", "Org2"}))
}

func TestNamespacePolicyHonoursConfiguredPolicy(t *testing.T) {
	// A policy either organization can satisfy on its own, which is what an endorser
	// selection narrowing discovery to one of them needs.
	const policy = "OR ('Org1MSP.member','Org2MSP.member')"
	tms := fabric.WithNamespacePolicy(newTMS(), policy)

	assert.Equal(t, policy, namespacePolicy(tms, []string{"Org1", "Org2"}))
}
