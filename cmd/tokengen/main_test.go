/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen/cobra/pp/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	fabtoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/driver"
	dlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/slices"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

// To run this command, first make sure to install the tokengen tool. To do that run `make tokengen`.
//go:generate tokengen gen dlog --idemix "./testdata/idemix" --issuers "./testdata/issuers/msp" --auditors "./testdata/auditors/msp" --output "./testdata"

func TestCompile(t *testing.T) {
	gt := NewGomegaWithT(t)
	_, err := gexec.Build("github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen")
	gt.Expect(err).NotTo(HaveOccurred())
	defer gexec.CleanupBuildArtifacts()
}

func TestGenFullSuccess(t *testing.T) {
	gt := NewGomegaWithT(t)
	tokengen, err := gexec.Build("github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen")
	gt.Expect(err).NotTo(HaveOccurred())
	defer gexec.CleanupBuildArtifacts()

	tempOutput, err := os.MkdirTemp("", "tokengen-test")
	gt.Expect(err).NotTo(HaveOccurred())
	defer os.RemoveAll(tempOutput)

	testGenRun(
		gt,
		tokengen,
		[]string{
			"gen",
			"dlog",
			"--idemix",
			"./testdata/idemix",
			"--issuers",
			"./testdata/issuers/msp",
			"--auditors",
			"./testdata/auditors/msp",
			"--output",
			tempOutput,
		},
	)

	validateOutputEquivalent(
		gt,
		tempOutput,
		"./testdata/auditors/msp",
		"./testdata/issuers/msp",
		"./testdata/idemix/msp/IssuerPublicKey",
	)
}

func TestFullUpdate(t *testing.T) {
	gt := NewWithT(t)
	tokengen, err := gexec.Build("github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen")
	gt.Expect(err).NotTo(HaveOccurred())
	defer gexec.CleanupBuildArtifacts()

	tempOutput, err := os.MkdirTemp("", "tokengen-update-test")
	gt.Expect(err).NotTo(HaveOccurred())
	defer os.RemoveAll(tempOutput)

	// Switching the auditor and issuer certs to test the update function
	testGenRun(
		gt,
		tokengen,
		[]string{
			"update",
			"dlog",
			"--issuers",
			"./testdata/auditors/msp",
			"--auditors",
			"./testdata/issuers/msp",
			"--input",
			"./testdata/zkatdlog_pp.json",
			"--output",
			tempOutput,
		},
	)

	validateOutputEquivalent(
		gt,
		tempOutput,
		"./testdata/issuers/msp",
		"./testdata/auditors/msp",
		"./testdata/idemix/msp/IssuerPublicKey",
	)
}

func TestPartialUpdate(t *testing.T) {
	gt := NewWithT(t)
	tokengen, err := gexec.Build("github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen")
	gt.Expect(err).NotTo(HaveOccurred())
	defer gexec.CleanupBuildArtifacts()

	tempOutput, err := os.MkdirTemp("", "tokengen-update-test")
	gt.Expect(err).NotTo(HaveOccurred())
	defer os.RemoveAll(tempOutput)

	// Only changing the issuer cert to also use the auditor cert.
	// The other auditor cert should stay the same.
	testGenRun(
		gt,
		tokengen,
		[]string{
			"update",
			"dlog",
			"--issuers",
			"./testdata/auditors/msp",
			"--input",
			"./testdata/zkatdlog_pp.json",
			"--output",
			tempOutput,
		},
	)

	validateOutputEquivalent(
		gt,
		tempOutput,
		"./testdata/auditors/msp",
		"./testdata/auditors/msp",
		"./testdata/idemix/msp/IssuerPublicKey",
	)
}

func TestGenFailure(t *testing.T) {
	gt := NewGomegaWithT(t)
	tokengen, err := gexec.Build("github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen")
	gt.Expect(err).NotTo(HaveOccurred())
	defer gexec.CleanupBuildArtifacts()

	type T struct {
		Args   []string
		ErrMsg string
	}
	var tests []T
	for _, driver := range []string{"fabtoken"} {
		tests = append(tests, []T{
			{
				Args: []string{
					"gen",
					driver,
					"--auditors", "a:Org1MSP,b:Org2MSP"},
				ErrMsg: "Error: failed to generate public parameters: failed to get auditor identity [a:Org1MSP]",
			},
			{
				Args: []string{
					"gen",
					driver,
					"--auditors", "aOrg1MSP,b:Org2MSP",
				},
				ErrMsg: "Error: failed to generate public parameters: failed to get auditor identity [aOrg1MSP]: failed to load certificates from aOrg1MSP/signcerts: stat aOrg1MSP/signcerts: no such file or directory",
			},
			{
				Args: []string{
					"gen",
					driver,
					"--issuers", "a:Org1MSP,b:Org2MSP",
				},
				ErrMsg: "Error: failed to generate public parameters: failed to get issuer identity [a:Org1MSP]",
			},
			{
				Args: []string{
					"gen",
					driver,
					"--issuers", "aOrg1MSP,b:Org2MSP",
				},
				ErrMsg: "Error: failed to generate public parameters: failed to get issuer identity [aOrg1MSP]: failed to load certificates from aOrg1MSP/signcerts: stat aOrg1MSP/signcerts: no such file or directory",
			},
		}...,
		)
	}
	tests = append(tests, []T{
		{
			Args: []string{
				"gen",
				"dlog",
				"--issuers", "aOrg1MSP,b:Org2MSP",
			},
			ErrMsg: "Error: failed to generate public parameters: failed reading idemix issuer public key [msp/IssuerPublicKey]: open msp/IssuerPublicKey: no such file or directory",
		},
		{
			Args: []string{
				"gen",
				"dlog",
				"--idemix", "./testdata/idemix",
				"--issuers", "Error: failed to generate public parameters: failed to get issuer identity [aOrg1MSP]: invalid input [aOrg1MSP]",
			},
			ErrMsg: "Error: failed to generate public parameters: failed to get issuer identity [Error: failed to generate public parameters: failed to get issuer identity [aOrg1MSP]: invalid input [aOrg1MSP]]: invalid input [Error: failed to generate public parameters: failed to get issuer identity [aOrg1MSP]: invalid input [aOrg1MSP]]",
		},
	}...,
	)

	for _, test := range tests {
		testGenRunWithError(gt, tokengen, test.Args, test.ErrMsg)
	}

	tempOutput, err := os.MkdirTemp("", "tokengen-test")
	gt.Expect(err).NotTo(HaveOccurred())

	defer os.RemoveAll(tempOutput)
	testGenRun(gt, tokengen, []string{"gen", "fabtoken", "--output", tempOutput})
	raw, err := os.ReadFile(filepath.Join(tempOutput, "fabtoken_pp.json"))
	gt.Expect(err).NotTo(HaveOccurred())
	is := core.NewPPManagerFactoryService(fabtoken.NewPPMFactory(), dlog.NewPPMFactory())
	pp, err := is.PublicParametersFromBytes(raw)
	gt.Expect(err).NotTo(HaveOccurred())
	_, err = is.DefaultValidator(pp)
	gt.Expect(err).NotTo(HaveOccurred())

	testGenRun(gt, tokengen, []string{"gen", "dlog", "--idemix", "./testdata/idemix", "--output", tempOutput})
	raw, err = os.ReadFile(filepath.Join(tempOutput, "zkatdlog_pp.json"))
	gt.Expect(err).NotTo(HaveOccurred())
	pp, err = is.PublicParametersFromBytes(raw)
	gt.Expect(err).NotTo(HaveOccurred())
	_, err = is.DefaultValidator(pp)
	gt.Expect(err).NotTo(HaveOccurred())
}

func validateOutputEquivalent(gt *WithT, tempOutput, auditorsMSPdir, issuersMSPdir, idemixMSPdir string) {
	ppRaw, err := os.ReadFile(filepath.Join(tempOutput, "zkatdlog_pp.json"))
	gt.Expect(err).NotTo(HaveOccurred())

	pp, err := v1.NewPublicParamsFromBytes(ppRaw, v1.DLogPublicParameters)
	gt.Expect(err).NotTo(HaveOccurred())
	gt.Expect(pp.Validate()).NotTo(HaveOccurred())

	auditors := pp.Auditors()
	auditor, err := common.GetMSPIdentity(auditorsMSPdir, msp.AuditorMSPID)
	gt.Expect(err).NotTo(HaveOccurred())
	gt.Expect(auditors[0]).To(Equal(auditor))

	issuers := pp.IssuerIDs
	issuer, err := common.GetMSPIdentity(issuersMSPdir, msp.IssuerMSPID)
	gt.Expect(err).NotTo(HaveOccurred())
	gt.Expect(issuers[0]).To(BeEquivalentTo(issuer))

	idemixPK, err := os.ReadFile(idemixMSPdir)
	gt.Expect(err).NotTo(HaveOccurred())
	gt.Expect(idemixPK).To(BeEquivalentTo(slices.GetUnique(pp.IdemixIssuerPublicKeys).PublicKey))
}

func testGenRunWithError(gt *WithT, tokengen string, args []string, errMsg string) {
	b, err := exec.Command(tokengen, args...).CombinedOutput()
	gt.Expect(err).To(HaveOccurred())
	gt.Expect(string(b)).To(ContainSubstring(errMsg))
}

func testGenRun(gt *WithT, tokengen string, args []string) {
	_, err := exec.Command(tokengen, args...).CombinedOutput()
	gt.Expect(err).ToNot(HaveOccurred(), "failed running tokengen [%s:%v]", tokengen, args)
}
