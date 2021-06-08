/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func TestCompile(t *testing.T) {
	gt := NewGomegaWithT(t)
	_, err := gexec.Build("github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen")
	gt.Expect(err).NotTo(HaveOccurred())
	defer gexec.CleanupBuildArtifacts()
}
