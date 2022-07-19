/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/cmd/pp/common"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/driver"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

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

	tempOutput, err := ioutil.TempDir("", "tokengen-test")
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

	// Check output
	ppRaw, err := ioutil.ReadFile(filepath.Join(tempOutput, "zkatdlog_pp.json"))
	gt.Expect(err).NotTo(HaveOccurred())

	pp, err := crypto.NewPublicParamsFromBytes(ppRaw, crypto.DLogPublicParameters)
	gt.Expect(err).NotTo(HaveOccurred())

	auditors := pp.Auditors()
	auditor, err := common.GetMSPIdentity("./testdata/auditors/msp", msp.AuditorMSPID)
	gt.Expect(err).NotTo(HaveOccurred())
	gt.Expect(auditors[0]).To(Equal(auditor))

	issuers := pp.Issuers
	issuer, err := common.GetMSPIdentity("./testdata/issuers/msp", msp.IssuerMSPID)
	gt.Expect(err).NotTo(HaveOccurred())
	gt.Expect(issuers[0]).To(BeEquivalentTo(issuer))

	idemixPK, err := ioutil.ReadFile("./testdata/idemix/msp/IssuerPublicKey")
	gt.Expect(err).NotTo(HaveOccurred())
	gt.Expect(idemixPK).To(BeEquivalentTo(pp.IdemixPK))
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
				ErrMsg: "Error: failed to generate public parameters: failed to get auditor identity [aOrg1MSP]: failed to create x509 provider for [aOrg1MSP]: could not load a valid signer certificate from directory aOrg1MSP/signcerts: stat aOrg1MSP/signcerts: no such file or directory",
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
				ErrMsg: "Error: failed to generate public parameters: failed to get issuer identity [aOrg1MSP]: failed to create x509 provider for [aOrg1MSP]: could not load a valid signer certificate from directory aOrg1MSP/signcerts: stat aOrg1MSP/signcerts: no such file or directory",
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

	tempOutput, err := ioutil.TempDir("", "tokengen-test")
	gt.Expect(err).NotTo(HaveOccurred())

	defer os.RemoveAll(tempOutput)
	testGenRun(gt, tokengen, []string{"gen", "fabtoken", "--output", tempOutput})
	raw, err := ioutil.ReadFile(filepath.Join(tempOutput, "fabtoken_pp.json"))
	gt.Expect(err).NotTo(HaveOccurred())
	_, _, err = token.NewServicesFromPublicParams(raw)
	gt.Expect(err).NotTo(HaveOccurred())

	testGenRun(gt, tokengen, []string{"gen", "dlog", "--idemix", "./testdata/idemix", "--output", tempOutput})
	raw, err = ioutil.ReadFile(filepath.Join(tempOutput, "zkatdlog_pp.json"))
	gt.Expect(err).NotTo(HaveOccurred())
	_, _, err = token.NewServicesFromPublicParams(raw)
	gt.Expect(err).NotTo(HaveOccurred())
}

func testGenRunWithError(gt *WithT, tokengen string, args []string, errMsg string) {
	b, err := exec.Command(tokengen, args...).CombinedOutput()
	gt.Expect(err).To(HaveOccurred())
	gt.Expect(string(b)).To(ContainSubstring(errMsg))
}

func testGenRun(gt *WithT, tokengen string, args []string) {
	_, err := exec.Command(tokengen, args...).CombinedOutput()
	gt.Expect(err).ToNot(HaveOccurred())
}
