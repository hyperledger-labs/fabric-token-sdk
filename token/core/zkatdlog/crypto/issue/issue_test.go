/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package issue_test

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	bn256 "github.ibm.com/fabric-research/mathlib"
)

var _ = Describe("Issue Correctness", func() {
	var (
		prover   *issue.Prover
		verifier *issue.Verifier
	)
	BeforeEach(func() {
		prover, verifier = prepareZKIssue()
	})
	Describe("Prove", func() {
		Context("parameters and witness are initialized correctly", func() {
			It("Succeeds", func() {
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				Expect(proof).NotTo(BeNil())
				err = verifier.Verify(proof)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

func prepareInputsForZKIssue(pp *crypto.PublicParams) ([]*token.TokenDataWitness, []*bn256.G1) {
	values := make([]*bn256.Zr, 2)
	values[0] = bn256.Curves[pp.Curve].NewZrFromInt(120)
	values[1] = bn256.Curves[pp.Curve].NewZrFromInt(190)

	rand, _ := bn256.Curves[pp.Curve].Rand()
	bF := make([]*bn256.Zr, len(values))
	for i := 0; i < len(values); i++ {
		bF[i] = bn256.Curves[pp.Curve].NewRandomZr(rand)
	}
	ttype := "ABC"

	tokens := PrepareTokens(values, bF, ttype, pp.ZKATPedParams)
	return issue.NewTokenDataWitness(ttype, values, bF), tokens
}

func prepareZKIssue() (*issue.Prover, *issue.Verifier) {
	pp, err := crypto.Setup(100, 2, nil)
	Expect(err).NotTo(HaveOccurred())

	tw, tokens := prepareInputsForZKIssue(pp)

	prover := issue.NewProver(tw, tokens, true, pp)
	verifier := issue.NewVerifier(tokens, true, pp)

	return prover, verifier
}
