/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ppm_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/audit"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/ecdsa"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue/anonym"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/ppm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	bn256 "github.ibm.com/fabric-research/mathlib"
)

var _ = Describe("PublicParamsManager", func() {
	var (
		engine *ppm.PublicParamsManager
		pp     *crypto.PublicParams

		auditor *audit.Auditor
		ipk     []byte
	)

	BeforeEach(func() {
		var err error
		// prepare public parameters
		ipk, err = ioutil.ReadFile("./testdata/idemix/msp/IssuerPublicKey")
		Expect(err).NotTo(HaveOccurred())
		pp, err = crypto.Setup(100, 2, ipk)
		Expect(err).NotTo(HaveOccurred())

		//prepare issuers' public keys
		sk, _, err := anonym.GenerateKeyPair("ABC", pp)
		Expect(sk).NotTo(BeNil())
		Expect(err).NotTo(HaveOccurred())

		asigner, _ := prepareECDSASigner()
		auditor = &audit.Auditor{Signer: asigner, PedersenParams: pp.ZKATPedParams, NYMParams: pp.IdemixPK}
		araw, err := asigner.Serialize()
		Expect(err).NotTo(HaveOccurred())
		pp.Auditor = araw

		// initialize enginw with pp
		engine = ppm.New(pp)
	})

	Describe("Add Auditor", func() {
		var (
			raw []byte
			err error
		)
		BeforeEach(func() {
			raw, err = auditor.Signer.Serialize()
			Expect(err).NotTo(HaveOccurred())
		})
		When("addAuditor is called correctly", func() {
			It("succeeds", func() {
				ppbytes, err := engine.SetAuditor(raw)
				Expect(err).NotTo(HaveOccurred())
				pp := &crypto.PublicParams{}
				pp.Label = crypto.DLogPublicParameters
				err = pp.Deserialize(ppbytes)
				Expect(err).NotTo(HaveOccurred())
				Expect(bytes.Equal(pp.Auditor, raw)).To(Equal(true))
			})
		})
		When("addAuditor is called with invalid identity", func() {
			BeforeEach(func() {
				raw = []byte("invalid auditor")
			})
			It("succeeds", func() {
				ppbytes, err := engine.SetAuditor(raw)
				Expect(err).To(HaveOccurred())
				Expect(ppbytes).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("failed to retrieve auditor's identity"))
			})
		})
	})

	Describe("Add Issuer", func() {
		Context("AddIssuer is called correctly to add a new anonymissuer", func() {
			var (
				issuer *bn256.G1
				raw    []byte
			)
			BeforeEach(func() {
				var err error
				issuer = bn256.Curves[pp.Curve].GenG1
				raw, err = json.Marshal(issuer)
				Expect(err).NotTo(HaveOccurred())
			})
			It("succeeds", func() {
				ppbytes, err := engine.AddIssuer(raw)
				Expect(err).NotTo(HaveOccurred())
				Expect(ppbytes).NotTo(BeNil())
			})
		})
	})
})

func prepareECDSASigner() (*ecdsa.ECDSASigner, *ecdsa.ECDSAVerifier) {
	signer, err := ecdsa.NewECDSASigner()
	Expect(err).NotTo(HaveOccurred())
	return signer, signer.ECDSAVerifier
}
