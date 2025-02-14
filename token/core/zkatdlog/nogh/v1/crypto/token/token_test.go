/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token_test

import (
	math "github.com/IBM/mathlib"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/token"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Token", func() {
	var (
		inf   *token2.Metadata
		token *token2.Token
		pp    *v1.PublicParams
	)

	BeforeEach(func() {
		var err error
		pp, err = v1.Setup(64, nil, math.FP256BN_AMCL)
		Expect(err).NotTo(HaveOccurred())
		c := math.Curves[pp.Curve]
		rand, err := c.Rand()
		Expect(err).NotTo(HaveOccurred())
		inf = &token2.Metadata{
			Value:          c.NewZrFromInt(50),
			Type:           "ABC",
			BlindingFactor: c.NewRandomZr(rand),
		}
		token = &token2.Token{}
		token.Data = c.NewG1()
		token.Data.Add(pp.PedersenGenerators[1].Mul(inf.Value))
		token.Data.Add(pp.PedersenGenerators[2].Mul(inf.BlindingFactor))
		token.Data.Add(pp.PedersenGenerators[0].Mul(c.HashToZr([]byte("ABC"))))
	})
	Describe("get token in the clear", func() {
		When("token is computed correctly", func() {
			It("succeeds", func() {
				t, err := token.ToClear(inf, pp)
				Expect(err).NotTo(HaveOccurred())
				Expect(t.Type).To(Equal(token3.Type("ABC")))
				Expect(t.Quantity).To(Equal("0x" + inf.Value.String()))
			})
		})
	})
})
