/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package sigproof_test

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	sigproof "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/sigproof"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ZK proof of PS signature", func() {
	var (
		w        *sigproof.POKWitness
		prover   *sigproof.POKProver
		verifier *sigproof.POKVerifier
		signer   *pssign.Signer
	)
	BeforeEach(func() {
		signer = getSigner(3)
		w = prepareWitness(signer)
		P := bn256.G1Gen()
		verifier = &sigproof.POKVerifier{PK: signer.PK, Q: signer.Q, P: P}
		prover = &sigproof.POKProver{Witness: w, POKVerifier: verifier}
	})
	Describe("Prove", func() {
		Context("parameters and witness are initialized correctly", func() {
			It("Succeeds", func() {
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				Expect(proof).NotTo(BeNil())
				psp := &sigproof.POK{}
				err = json.Unmarshal(proof, psp)
				Expect(err).NotTo(HaveOccurred())
				Expect(psp.Challenge).NotTo(BeNil())
				Expect(psp.Signature).NotTo(BeNil())
				Expect(psp.Messages).NotTo(BeNil())
				Expect(len(psp.Messages)).To(Equal(3))
			})
		})
	})
	Describe("Verify", func() {
		It("Succeeds", func() {
			proof, err := prover.Prove()
			Expect(err).NotTo(HaveOccurred())
			err = verifier.Verify(proof)
			Expect(err).NotTo(HaveOccurred())
		})
		When("POK is not valid", func() {
			It("fails", func() {
				rand, err := bn256.GetRand()
				Expect(err).NotTo(HaveOccurred())
				// tamper with signed hidden
				prover.Witness.Messages[1] = bn256.RandModOrder(rand)
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				// proof verification fails
				err = verifier.Verify(proof)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("proof of PS signature is not valid"))
			})
		})
	})

})

func prepareWitness(s *pssign.Signer) *sigproof.POKWitness {
	w := &sigproof.POKWitness{}
	w.Messages = make([]*bn256.Zr, len(s.SK)-2)
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())

	for i := 0; i < len(w.Messages); i++ {
		w.Messages[i] = bn256.RandModOrder(rand)
	}
	w.Signature, err = s.Sign(w.Messages)
	Expect(err).NotTo(HaveOccurred())

	return w
}

func getSigner(length int) *pssign.Signer {
	s := &pssign.Signer{}
	s.KeyGen(length)
	return s
}
