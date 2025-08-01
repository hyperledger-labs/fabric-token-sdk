/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package transfer_test

import (
	"context"

	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	transfer3 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer/mock"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("Sender", func() {
	var (
		fakeSigningIdentity *mock.SigningIdentity
		signers             []driver.Signer
		pp                  *v1.PublicParams

		transfer *transfer3.Action
		sender   *transfer3.Sender

		invalues  []*math.Zr
		outvalues []uint64
		inBF      []*math.Zr
		tokens    []*token.Token

		owners [][]byte
		ids    []*token2.ID
	)
	BeforeEach(func() {
		var err error
		pp, err = v1.Setup(8, nil, math.FP256BN_AMCL)
		Expect(err).NotTo(HaveOccurred())
		owners = make([][]byte, 2)
		owners[0] = []byte("bob")
		owners[1] = []byte("charlie")
		signers = make([]driver.Signer, 3)
		fakeSigningIdentity = &mock.SigningIdentity{}
		signers[0] = fakeSigningIdentity
		signers[1] = fakeSigningIdentity
		signers[2] = fakeSigningIdentity

		fakeSigningIdentity.SignReturnsOnCall(0, []byte("signer[0]"), nil)
		fakeSigningIdentity.SignReturnsOnCall(1, []byte("signer[1]"), nil)
		fakeSigningIdentity.SignReturnsOnCall(2, []byte("signer[2]"), nil)

		c := math.Curves[pp.Curve]
		invalues = make([]*math.Zr, 3)
		invalues[0] = c.NewZrFromInt(50)
		invalues[1] = c.NewZrFromInt(20)
		invalues[2] = c.NewZrFromInt(30)

		inBF = make([]*math.Zr, 3)
		rand, err := c.Rand()
		Expect(err).NotTo(HaveOccurred())
		for i := range 3 {
			inBF[i] = c.NewRandomZr(rand)
		}
		outvalues = make([]uint64, 2)
		outvalues[0] = 65
		outvalues[1] = 35

		ids = make([]*token2.ID, 3)
		ids[0] = &token2.ID{TxId: "0"}
		ids[1] = &token2.ID{TxId: "1"}
		ids[2] = &token2.ID{TxId: "3"}

		inputs := PrepareTokens(invalues, inBF, "ABC", pp.PedersenGenerators, c)
		tokens = make([]*token.Token, 3)

		tokens[0] = &token.Token{Data: inputs[0], Owner: []byte("alice-1")}
		tokens[1] = &token.Token{Data: inputs[1], Owner: []byte("alice-2")}
		tokens[2] = &token.Token{Data: inputs[2], Owner: []byte("alice-3")}

		inputInf := make([]*token.Metadata, 3)
		inputInf[0] = &token.Metadata{Type: "ABC", Value: invalues[0], BlindingFactor: inBF[0]}
		inputInf[1] = &token.Metadata{Type: "ABC", Value: invalues[1], BlindingFactor: inBF[1]}
		inputInf[2] = &token.Metadata{Type: "ABC", Value: invalues[2], BlindingFactor: inBF[2]}

		sender, err = transfer3.NewSender(signers, tokens, ids, inputInf, pp)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Transfer", func() {
		When("transfer is computed correctly", func() {
			It("succeeds", func() {
				var err error
				transfer, _, err = sender.GenerateZKTransfer(context.TODO(), outvalues, owners)
				Expect(err).NotTo(HaveOccurred())
				Expect(transfer).NotTo(BeNil())
				raw, err := transfer.Serialize()
				Expect(err).NotTo(HaveOccurred())

				sig, err := sender.SignTokenActions(raw)
				Expect(fakeSigningIdentity.SignCallCount()).To(Equal(3))
				Expect(sig).To(HaveLen(3))
				Expect(err).NotTo(HaveOccurred())
			})
		})
		When("when signature fails", func() {
			BeforeEach(func() {
				fakeSigningIdentity.SignReturnsOnCall(2, nil, errors.New("banana republic"))
			})
			It("no signature is returned", func() {
				var err error
				transfer, _, err = sender.GenerateZKTransfer(context.TODO(), outvalues, owners)
				Expect(err).NotTo(HaveOccurred())
				Expect(transfer).NotTo(BeNil())
				raw, err := transfer.Serialize()
				Expect(err).NotTo(HaveOccurred())

				sig, err := sender.SignTokenActions(raw)
				Expect(err).To(HaveOccurred())
				Expect(sig).To(BeNil())
				Expect(fakeSigningIdentity.SignCallCount()).To(Equal(3))
				Expect(err.Error()).To(ContainSubstring("banana republic"))
			})
		})
	})
})

func PrepareTokens(values, bf []*math.Zr, ttype string, pp []*math.G1, curve *math.Curve) []*math.G1 {
	tokens := make([]*math.G1, len(values))
	for i := range values {
		tokens[i] = prepareToken(values[i], bf[i], ttype, pp, curve)
	}
	return tokens
}
