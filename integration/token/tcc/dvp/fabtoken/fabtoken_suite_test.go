/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/hyperledger-labs/fabric-token-sdk/integration"
)

func TestEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "EndToEnd DVP (FabToken) Suite")
}

func StartPort() int {
	return integration.FabTokenDVP.StartPortForNode()
}
