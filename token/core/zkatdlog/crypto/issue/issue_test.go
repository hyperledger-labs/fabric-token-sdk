/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package issue_test

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

func prepareInputsForZKIssue(pp *crypto.PublicParams) ([]*token.TokenDataWitness, []*math.G1) {
	values := make([]*math.Zr, 2)
	values[0] = math.Curves[pp.Curve].NewZrFromInt(120)
	values[1] = math.Curves[pp.Curve].NewZrFromInt(190)

	rand, _ := math.Curves[pp.Curve].Rand()
	bF := make([]*math.Zr, len(values))
	for i := 0; i < len(values); i++ {
		bF[i] = math.Curves[pp.Curve].NewRandomZr(rand)
	}
	ttype := "ABC"

	tokens := PrepareTokens(values, bF, ttype, pp.ZKATPedParams)
	return issue.NewTokenDataWitness(ttype, values, bF), tokens
}

func prepareZKIssue() (*issue.Prover, *issue.Verifier) {
	pp, err := crypto.Setup(100, 2, nil, math.FP256BN_AMCL)
	Expect(err).NotTo(HaveOccurred())

	tw, tokens := prepareInputsForZKIssue(pp)

	prover := issue.NewProver(tw, tokens, true, pp)
	verifier := issue.NewVerifier(tokens, true, pp)

	return prover, verifier
}
