/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"os/exec"
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

func TestGen(t *testing.T) {
	gt := NewGomegaWithT(t)
	tokengen, err := gexec.Build("github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen")
	gt.Expect(err).NotTo(HaveOccurred())
	defer gexec.CleanupBuildArtifacts()

	// auditors
	args := []string{
		"gen",
		"-d", "fabtoken",
		"--auditors", "a:Org1MSP,b:Org2MSP",
	}
	b, err := exec.Command(tokengen, args...).CombinedOutput()
	gt.Expect(err).To(HaveOccurred())
	gt.Expect(string(b)).To(ContainSubstring("Error: failed to create x509 provider for auditor [a:Org1MSP]: could not load a valid signer certificate from directory a/signcerts: stat a/signcerts: no such file or directory"))

	args = []string{
		"gen",
		"-d", "fabtoken",
		"--auditors", "aOrg1MSP,b:Org2MSP",
	}
	b, err = exec.Command(tokengen, args...).CombinedOutput()
	gt.Expect(err).To(HaveOccurred())
	gt.Expect(string(b)).To(ContainSubstring("Error: invalid auditor [aOrg1MSP]"))

	// issuers
	args = []string{
		"gen",
		"-d", "fabtoken",
		"--issuers", "a:Org1MSP,b:Org2MSP",
	}
	b, err = exec.Command(tokengen, args...).CombinedOutput()
	gt.Expect(err).To(HaveOccurred())
	gt.Expect(string(b)).To(ContainSubstring("Error: failed to create x509 provider for issuer [a:Org1MSP]: could not load a valid signer certificate from directory a/signcerts: stat a/signcerts: no such file or directory"))

	args = []string{
		"gen",
		"-d", "fabtoken",
		"--issuers", "aOrg1MSP,b:Org2MSP",
	}
	b, err = exec.Command(tokengen, args...).CombinedOutput()
	gt.Expect(err).To(HaveOccurred())
	gt.Expect(string(b)).To(ContainSubstring("Error: invalid issuer [aOrg1MSP]"))
}
