/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token_test

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Token", func() {
	var (
		inf   *token2.TokenInformation
		token *token2.Token
		pp    *crypto.PublicParams
	)

	BeforeEach(func() {
		var err error
		pp, err = crypto.Setup(100, 2, nil, math.FP256BN_AMCL)
		Expect(err).NotTo(HaveOccurred())
		c := math.Curves[pp.Curve]
		rand, err := c.Rand()
		Expect(err).NotTo(HaveOccurred())
		inf = &token2.TokenInformation{
			Value:          c.NewZrFromInt(50),
			Type:           "ABC",
			BlindingFactor: c.NewRandomZr(rand),
		}
		token = &token2.Token{}
		token.Data = c.NewG1()
		token.Data.Add(pp.ZKATPedParams[1].Mul(inf.Value))
		token.Data.Add(pp.ZKATPedParams[2].Mul(inf.BlindingFactor))
		token.Data.Add(pp.ZKATPedParams[0].Mul(c.HashToZr([]byte("ABC"))))
	})
	Describe("get token in the clear", func() {
		When("token is computed correctly", func() {
			It("succeeds", func() {
				t, err := token.GetTokenInTheClear(inf, pp)
				Expect(err).NotTo(HaveOccurred())
				Expect(t.Type).To(Equal("ABC"))
				Expect(t.Quantity).To(Equal("0x" + inf.Value.String()))
			})
		})
	})
})
