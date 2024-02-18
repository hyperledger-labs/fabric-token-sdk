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
	. "github.com/onsi/ginkgo/v2"
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
	values := make([]uint64, 2)
	values[0] = 120
	values[1] = 190
	curve := math.Curves[pp.Curve]
	rand, _ := curve.Rand()
	bf := make([]*math.Zr, len(values))
	for i := 0; i < len(values); i++ {
		bf[i] = math.Curves[pp.Curve].NewRandomZr(rand)
	}

	tokens := make([]*math.G1, len(values))
	for i := 0; i < len(values); i++ {
		tokens[i] = NewToken(curve.NewZrFromInt(int64(values[i])), bf[i], "ABC", pp.PedParams, curve)
	}
	return token.NewTokenDataWitness("ABC", values, bf), tokens
}

func prepareZKIssue() (*issue.Prover, *issue.Verifier) {
	pp, err := crypto.Setup(32, nil, math.BN254)
	Expect(err).NotTo(HaveOccurred())

	tw, tokens := prepareInputsForZKIssue(pp)

	prover, err := issue.NewProver(tw, tokens, true, pp)
	Expect(err).NotTo(HaveOccurred())
	verifier := issue.NewVerifier(tokens, true, pp)

	return prover, verifier
}
