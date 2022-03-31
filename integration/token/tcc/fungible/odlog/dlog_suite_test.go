/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/integration"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "Orion EndToEnd ZKAT CC (DLOG) Suite")
}

func StartPortDlog() int {
	return integration.OrionZKATDLogBasics.StartPortForNode()
}
