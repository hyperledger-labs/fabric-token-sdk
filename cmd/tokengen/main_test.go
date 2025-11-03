/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen/cobra/pp/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/crypto/fabtokenv1"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/crypto/zkatdlognoghv1"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/slices"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

// To run this command, first make sure to install the tokengen tool. To do that run `make tokengen`.
//go:generate tokengen gen zkatdlognogh.v1 --idemix "./testdata/idemix" --issuers "./testdata/issuers/msp" --auditors "./testdata/auditors/msp" --output "./testdata"

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

	tempOutput := t.TempDir()
	defer utils.IgnoreErrorWithOneArg(os.RemoveAll, tempOutput)

	testGenRun(
		gt,
		tokengen,
		[]string{
			"gen",
			zkatdlognoghv1.DriverIdentifier,
			"--idemix",
			"./testdata/idemix",
			"--issuers",
			"./testdata/issuers/msp",
			"--auditors",
			"./testdata/auditors/msp",
			"--output",
			tempOutput,
			"--extra",
			"k1=./testdata/extras/f1.txt",
			"--extra",
			"k2=./testdata/extras/f2.txt",
		},
	)

	validateOutputEquivalent(
		gt,
		tempOutput,
		"./testdata/auditors/msp",
		"./testdata/issuers/msp",
		"./testdata/idemix/msp/IssuerPublicKey",
		v1.ProtocolV1,
		true,
	)
}

func TestGenFullSuccessWithVersionOverrideAndNewExtras(t *testing.T) {
	gt := NewGomegaWithT(t)
	tokengen, err := gexec.Build("github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen")
	gt.Expect(err).NotTo(HaveOccurred())
	defer gexec.CleanupBuildArtifacts()

	tempOutput := t.TempDir()
	defer utils.IgnoreErrorWithOneArg(os.RemoveAll, tempOutput)

	testGenRun(
		gt,
		tokengen,
		[]string{
			"gen",
			zkatdlognoghv1.DriverIdentifier,
			"--idemix",
			"./testdata/idemix",
			"--issuers",
			"./testdata/issuers/msp",
			"--auditors",
			"./testdata/auditors/msp",
			"--output",
			tempOutput,
			"--version",
			"2",
			"--extra",
			"k1=./testdata/extras/f1.txt",
			"--extra",
			"k2=./testdata/extras/f2.txt",
		},
	)

	validateOutputEquivalent(
		gt,
		tempOutput,
		"./testdata/auditors/msp",
		"./testdata/issuers/msp",
		"./testdata/idemix/msp/IssuerPublicKey",
		driver.TokenDriverVersion(2),
		true,
	)
}

func TestFullUpdate(t *testing.T) {
	gt := NewWithT(t)
	tokengen, err := gexec.Build("github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen")
	gt.Expect(err).NotTo(HaveOccurred())
	defer gexec.CleanupBuildArtifacts()

	tempOutput := t.TempDir()
	defer utils.IgnoreErrorWithOneArg(os.RemoveAll, tempOutput)

	// Switching the auditor and issuer certs to test the update function
	testGenRun(
		gt,
		tokengen,
		[]string{
			"update",
			zkatdlognoghv1.DriverIdentifier,
			"--issuers",
			"./testdata/auditors/msp",
			"--auditors",
			"./testdata/issuers/msp",
			"--input",
			"./testdata/zkatdlognoghv1_pp.json",
			"--output",
			tempOutput,
			"--extra",
			"k1=./testdata/extras/f1.txt",
			"--extra",
			"k2=./testdata/extras/f2.txt",
		},
	)

	validateOutputEquivalent(
		gt,
		tempOutput,
		"./testdata/issuers/msp",
		"./testdata/auditors/msp",
		"./testdata/idemix/msp/IssuerPublicKey",
		v1.ProtocolV1,
		true,
	)
}

func TestPartialUpdate(t *testing.T) {
	gt := NewWithT(t)
	tokengen, err := gexec.Build("github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen")
	gt.Expect(err).NotTo(HaveOccurred())
	defer gexec.CleanupBuildArtifacts()

	tempOutput := t.TempDir()
	defer utils.IgnoreErrorWithOneArg(os.RemoveAll, tempOutput)

	// Only changing the issuer cert to also use the auditor cert.
	// The other auditor cert should stay the same.
	testGenRun(
		gt,
		tokengen,
		[]string{
			"update",
			zkatdlognoghv1.DriverIdentifier,
			"--issuers",
			"./testdata/auditors/msp",
			"--input",
			"./testdata/zkatdlognoghv1_pp.json",
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
		v1.ProtocolV1,
		false,
	)
}

func TestPartialUpdateWithVersion(t *testing.T) {
	gt := NewWithT(t)
	tokengen, err := gexec.Build("github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen")
	gt.Expect(err).NotTo(HaveOccurred())
	defer gexec.CleanupBuildArtifacts()

	tempOutput := t.TempDir()
	defer utils.IgnoreErrorWithOneArg(os.RemoveAll, tempOutput)

	// Only changing the issuer cert to also use the auditor cert.
	// The other auditor cert should stay the same.
	testGenRun(
		gt,
		tokengen,
		[]string{
			"update",
			zkatdlognoghv1.DriverIdentifier,
			"--issuers",
			"./testdata/auditors/msp",
			"--input",
			"./testdata/zkatdlognoghv1_pp.json",
			"--output",
			tempOutput,
			"--version",
			"2",
		},
	)

	validateOutputEquivalent(
		gt,
		tempOutput,
		"./testdata/auditors/msp",
		"./testdata/auditors/msp",
		"./testdata/idemix/msp/IssuerPublicKey",
		driver.TokenDriverVersion(2),
		false,
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
	for _, driver := range []string{fabtokenv1.DriverIdentifier} {
		tests = append(tests, []T{
			{
				Args: []string{
					"gen",
					driver,
					"--auditors", "a,b"},
				ErrMsg: "Error: failed to generate public parameters: failed to get auditor identity [a]",
			},
			{
				Args: []string{
					"gen",
					driver,
					"--auditors", "aOrg1MSP,b",
				},
				ErrMsg: "Error: failed to generate public parameters: failed to get auditor identity [aOrg1MSP]: failed to load certificates from aOrg1MSP/signcerts: stat aOrg1MSP/signcerts: no such file or directory",
			},
			{
				Args: []string{
					"gen",
					driver,
					"--issuers", "a,b",
				},
				ErrMsg: "Error: failed to generate public parameters: failed to get issuer identity [a]",
			},
			{
				Args: []string{
					"gen",
					driver,
					"--issuers", "aOrg1MSP,b",
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
				zkatdlognoghv1.DriverIdentifier,
				"--issuers", "aOrg1MSP,b",
			},
			ErrMsg: "failed to generate public parameters: failed to load issuer public key: failed reading idemix issuer public key [msp/IssuerPublicKey]: open msp/IssuerPublicKey: no such file or directory",
		},
		{
			Args: []string{
				"gen",
				zkatdlognoghv1.DriverIdentifier,
				"--idemix", "./testdata/idemix",
				"--issuers", "Error: failed to generate public parameters: failed to get issuer identity [aOrg1MSP]: invalid input [aOrg1MSP]",
			},
			ErrMsg: "Error: failed to generate public parameters: failed to setup issuer and auditors: failed to get issuer identity [Error: failed to generate public parameters: failed to get issuer identity [aOrg1MSP]: invalid input [aOrg1MSP]]: failed to load certificates from Error: failed to generate public parameters: failed to get issuer identity [aOrg1MSP]: invalid input [aOrg1MSP]/signcerts: stat Error: failed to generate public parameters: failed to get issuer identity [aOrg1MSP]: invalid input [aOrg1MSP]/signcerts: no such file or directory",
		},
	}...,
	)

	for i, test := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			testGenRunWithError(gt, tokengen, test.Args, test.ErrMsg)
		})
	}
}

func validateOutputEquivalent(
	gt *WithT,
	tempOutput, auditorsMSPdir, issuersMSPdir, idemixMSPdir string,
	version driver.TokenDriverVersion,
	checkExtra bool,
) {
	ppRaw, err := os.ReadFile(filepath.Join(
		tempOutput,
		fmt.Sprintf("zkatdlognoghv%d_pp.json", version),
	))
	gt.Expect(err).NotTo(HaveOccurred())

	pp, err := v1.NewPublicParamsFromBytes(ppRaw, v1.DLogNoGHDriverName, version)
	gt.Expect(err).NotTo(HaveOccurred())
	gt.Expect(pp.Validate()).NotTo(HaveOccurred())

	auditors := pp.Auditors()
	auditor, err := common.GetX509Identity(auditorsMSPdir)
	gt.Expect(err).NotTo(HaveOccurred())
	gt.Expect(auditors[0]).To(Equal(auditor))

	issuers := pp.Issuers()
	issuer, err := common.GetX509Identity(issuersMSPdir)
	gt.Expect(err).NotTo(HaveOccurred())
	gt.Expect(issuers[0]).To(BeEquivalentTo(issuer))

	idemixPK, err := os.ReadFile(idemixMSPdir)
	gt.Expect(err).NotTo(HaveOccurred())
	gt.Expect(idemixPK).To(BeEquivalentTo(slices.GetUnique(pp.IdemixIssuerPublicKeys).PublicKey))

	extras := pp.Extras()
	if checkExtra {
		gt.Expect(extras).To(HaveLen(2))
		gt.Expect(extras["k1"]).To(BeEquivalentTo("f1"))
		gt.Expect(extras["k2"]).To(BeEquivalentTo("f2"))
	} else {
		gt.Expect(extras).To(BeEmpty())
	}
}

func testGenRunWithError(gt *WithT, tokengen string, args []string, errMsg string) {
	b, err := exec.Command(tokengen, args...).CombinedOutput()
	gt.Expect(err).To(HaveOccurred())
	gt.Expect(string(b)).To(ContainSubstring(errMsg))
}

func testGenRun(gt *WithT, tokengen string, args []string) {
	output, err := exec.Command(tokengen, args...).CombinedOutput()
	gt.Expect(err).ToNot(HaveOccurred(), "failed running tokengen [%s:%v] with output \n[%s]", tokengen, args, string(output))
}
