/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package rangeproof

import (
	"encoding/json"
	"math"
	"sync"
	"sync/atomic"

	mathlib "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/sigproof"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/pkg/errors"
)

// todo check lengths
type Proof struct {
	Challenge        *mathlib.Zr
	EqualityProofs   *EqualityProofs
	MembershipProofs []*MembershipProof
}

type EqualityProofs struct {
	Type                     *mathlib.Zr
	Value                    []*mathlib.Zr
	TokenBlindingFactor      []*mathlib.Zr
	CommitmentBlindingFactor []*mathlib.Zr
}

type MembershipProof struct {
	Commitments     []*mathlib.G1
	SignatureProofs [][]byte
}

type Prover struct {
	*Verifier
	tokenWitness []*token.TokenDataWitness
	Signatures   []*pssign.Signature
}

func NewProver(tw []*token.TokenDataWitness, token []*mathlib.G1, signatures []*pssign.Signature, exponent int, pp []*mathlib.G1, PK []*mathlib.G2, P *mathlib.G1, Q *mathlib.G2, c *mathlib.Curve) *Prover {
	return &Prover{
		tokenWitness: tw,
		Signatures:   signatures,
		Verifier: &Verifier{
			Token:          token,
			Base:           uint64(len(signatures)),
			Exponent:       exponent,
			PedersenParams: pp,
			PK:             PK,
			P:              P,
			Q:              Q,
			Curve:          c,
		},
	}
}

type Verifier struct {
	Token          []*mathlib.G1
	Base           uint64
	Exponent       int
	PedersenParams []*mathlib.G1
	Q              *mathlib.G2
	P              *mathlib.G1
	PK             []*mathlib.G2
	Curve          *mathlib.Curve
}

func NewVerifier(token []*mathlib.G1, base uint64, exponent int, pp []*mathlib.G1, PK []*mathlib.G2, P *mathlib.G1, Q *mathlib.G2, c *mathlib.Curve) *Verifier {
	return &Verifier{
		Token:          token,
		Base:           base,
		Exponent:       exponent,
		PedersenParams: pp,
		PK:             PK,
		P:              P,
		Q:              Q,
		Curve:          c,
	}
}

type Randomness struct {
	Type                     *mathlib.Zr
	Value                    []*mathlib.Zr
	TokenBlindingFactor      []*mathlib.Zr
	CommitmentBlindingFactor []*mathlib.Zr
}

type Commitment struct {
	Token             []*mathlib.G1
	CommitmentToValue []*mathlib.G1
}

type commitmentWitnessBlindingFactor struct {
	commitment     [][]*mathlib.G1
	witness        [][]*sigproof.MembershipWitness
	blindingFactor []*mathlib.Zr
}

func (p *Prover) Prove() ([]byte, error) {
	proof := &Proof{}
	var err error
	preProcessed, err := p.preProcess()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	wg.Add(len(p.Token) * p.Exponent)

	parallelErr := &atomic.Value{}

	proof.MembershipProofs = make([]*MembershipProof, len(p.Token))
	for k := 0; k < len(proof.MembershipProofs); k++ {
		proof.MembershipProofs[k] = &MembershipProof{}
		proof.MembershipProofs[k].Commitments = make([]*mathlib.G1, p.Exponent)
		proof.MembershipProofs[k].SignatureProofs = make([][]byte, p.Exponent)
		for i := 0; i < p.Exponent; i++ {
			proof.MembershipProofs[k].Commitments[i] = preProcessed.commitment[k][i]
			mp := sigproof.NewMembershipProver(preProcessed.witness[k][i], proof.MembershipProofs[k].Commitments[i], p.P, p.Q, p.PK, p.PedersenParams[:2], p.Curve)

			go func(k, i int) {
				defer wg.Done()
				var err error
				proof.MembershipProofs[k].SignatureProofs[i], err = mp.Prove()
				if err != nil {
					parallelErr.Store(err)
				}
			}(k, i)
		}
	}

	wg.Wait()

	if parallelErr.Load() != nil {
		return nil, parallelErr.Load().(error)
	}

	// show that value in token = value in the aggregate commitment
	commitment, randomness, err := p.computeCommitment()
	if err != nil {
		return nil, err
	}

	proof.Challenge = p.computeChallenge(commitment, preProcessed.commitment)

	// equality proof
	proof.EqualityProofs = &EqualityProofs{}
	for k := 0; k < len(p.Token); k++ {
		sp := &common.SchnorrProver{Challenge: proof.Challenge, Randomness: []*mathlib.Zr{randomness.Value[k], randomness.TokenBlindingFactor[k], randomness.CommitmentBlindingFactor[k]}, Witness: []*mathlib.Zr{p.tokenWitness[k].Value, p.tokenWitness[k].BlindingFactor, preProcessed.blindingFactor[k]}, SchnorrVerifier: &common.SchnorrVerifier{Curve: p.Curve}}
		proofs, err := sp.Prove()
		if err != nil {
			return nil, err
		}
		proof.EqualityProofs.Value = append(proof.EqualityProofs.Value, proofs[0])
		proof.EqualityProofs.TokenBlindingFactor = append(proof.EqualityProofs.TokenBlindingFactor, proofs[1])
		proof.EqualityProofs.CommitmentBlindingFactor = append(proof.EqualityProofs.CommitmentBlindingFactor, proofs[2])
	}
	proof.EqualityProofs.Type = p.Curve.ModMul(proof.Challenge, p.Curve.HashToZr([]byte(p.tokenWitness[0].Type)), p.Curve.GroupOrder)
	proof.EqualityProofs.Type = p.Curve.ModAdd(proof.EqualityProofs.Type, randomness.Type, p.Curve.GroupOrder)

	return json.Marshal(proof)
}

func (v *Verifier) Verify(raw []byte) error {
	proof := &Proof{}
	err := json.Unmarshal(raw, proof)
	if err != nil {
		return err
	}
	if len(proof.MembershipProofs) != len(v.Token) {
		return errors.Errorf("failed to verify range proofz")
	}

	var verifications []func()
	parallelErr := &atomic.Value{}

	//  verify membership
	for k := 0; k < len(v.Token); k++ {
		if len(proof.MembershipProofs[k].Commitments) != len(proof.MembershipProofs[k].SignatureProofs) {
			return errors.Errorf("failed to verify range proof")
		}
		for i := 0; i < len(proof.MembershipProofs[k].Commitments); i++ {
			mv := sigproof.NewMembershipVerifier(proof.MembershipProofs[k].Commitments[i], v.P, v.Q, v.PK, v.PedersenParams[:2], v.Curve)
			proofToVerify := proof.MembershipProofs[k].SignatureProofs[i]
			verifications = append(verifications, func() {
				err := mv.Verify(proofToVerify)
				if err != nil {
					parallelErr.Store(errors.Wrapf(err, "failed to verify range proof"))
				}
			})
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(verifications))

	for _, verification := range verifications {
		go func(f func()) {
			defer wg.Done()
			f()
		}(verification)
	}

	wg.Wait()

	if parallelErr.Load() != nil {
		return parallelErr.Load().(error)
	}

	//  verify equality
	com := v.recomputeCommitments(proof)
	coms := make([][]*mathlib.G1, len(proof.MembershipProofs))
	for i := 0; i < len(proof.MembershipProofs); i++ {
		for k := 0; k < len(proof.MembershipProofs[i].Commitments); k++ {
			coms[i] = append(coms[i], proof.MembershipProofs[i].Commitments[k])
		}
	}
	chal := v.computeChallenge(com, coms)
	if !chal.Equals(proof.Challenge) {
		return errors.Errorf("failed to verify range proof")
	}

	return nil
}

func (p *Prover) preProcess() (*commitmentWitnessBlindingFactor, error) {
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, err
	}
	membershipWitness := make([][]*sigproof.MembershipWitness, len(p.tokenWitness))
	commitmentBlindingFactor := make([]*mathlib.Zr, len(p.tokenWitness))
	coms := make([][]*mathlib.G1, len(p.tokenWitness))

	for k := 0; k < len(p.tokenWitness); k++ {
		values := make([]int, p.Exponent)
		v, err := p.tokenWitness[k].Value.Int()
		if err != nil {
			return nil, err
		}
		if v >= int64(math.Pow(float64(p.Base), float64(p.Exponent))) {
			return nil, errors.Errorf("can't compute range proof: value of token outside authorized range")
		}
		values[0] = int(v % int64(p.Base))
		for i := 0; i < p.Exponent-1; i++ {
			values[p.Exponent-1-i] = int(v / int64(math.Pow(float64(p.Base), float64(p.Exponent-1-i)))) // quotient
			v = v % int64(math.Pow(float64(p.Base), float64(p.Exponent-1-i)))                           // remainder
		}

		membershipWitness[k] = make([]*sigproof.MembershipWitness, p.Exponent)
		commitmentBlindingFactor[k] = p.Curve.NewZrFromInt(0)
		coms[k] = make([]*mathlib.G1, p.Exponent)
		for i := 0; i < p.Exponent; i++ {
			bf := p.Curve.NewRandomZr(rand)
			coms[k][i], err = common.ComputePedersenCommitment([]*mathlib.Zr{p.Curve.NewZrFromInt(int64(values[i])), bf}, p.PedersenParams[:2], p.Curve)
			if err != nil {
				return nil, err
			}

			membershipWitness[k][i] = sigproof.NewMembershipWitness(p.Signatures[values[i]], p.Curve.NewZrFromInt(int64(values[i])), bf)
			pow := p.Curve.NewZrFromInt(int64(math.Pow(float64(p.Base), float64(i))))
			commitmentBlindingFactor[k] = p.Curve.ModAdd(commitmentBlindingFactor[k], p.Curve.ModMul(bf, pow, p.Curve.GroupOrder), p.Curve.GroupOrder)
		}
	}
	return &commitmentWitnessBlindingFactor{
		commitment:     coms,
		blindingFactor: commitmentBlindingFactor,
		witness:        membershipWitness,
	}, nil
}

func (p *Prover) computeCommitment() (*Commitment, *Randomness, error) {
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, nil, err
	}
	// generate randomness
	randomness := &Randomness{}
	randomness.Type = p.Curve.NewRandomZr(rand)
	for i := 0; i < len(p.Token); i++ {
		randomness.Value = append(randomness.Value, p.Curve.NewRandomZr(rand))
		randomness.CommitmentBlindingFactor = append(randomness.CommitmentBlindingFactor, p.Curve.NewRandomZr(rand))
		randomness.TokenBlindingFactor = append(randomness.TokenBlindingFactor, p.Curve.NewRandomZr(rand))
	}

	// compute commitment
	commitment := &Commitment{}
	for i := 0; i < len(p.tokenWitness); i++ {
		tok := p.PedersenParams[0].Mul(randomness.Type)
		tok.Add(p.PedersenParams[1].Mul(randomness.Value[i]))
		tok.Add(p.PedersenParams[2].Mul(randomness.TokenBlindingFactor[i]))
		commitment.Token = append(commitment.Token, tok)

		com := p.PedersenParams[0].Mul(randomness.Value[i])
		com.Add(p.PedersenParams[1].Mul(randomness.CommitmentBlindingFactor[i]))
		commitment.CommitmentToValue = append(commitment.CommitmentToValue, com)
	}

	return commitment, randomness, nil
}

func (v *Verifier) computeChallenge(commitment *Commitment, comToValue [][]*mathlib.G1) *mathlib.Zr {
	g1array := common.GetG1Array([]*mathlib.G1{v.P}, v.Token, commitment.Token, commitment.CommitmentToValue, v.PedersenParams)
	g2array := common.GetG2Array([]*mathlib.G2{v.Q}, v.PK)
	bytes := append(g1array.Bytes(), g2array.Bytes()...)
	for i := 0; i < len(comToValue); i++ {
		bytes = append(bytes, common.GetG1Array(comToValue[i]).Bytes()...)
	}
	return v.Curve.HashToZr(bytes)
}

func (v *Verifier) recomputeCommitments(p *Proof) *Commitment {
	c := &Commitment{}
	// recompute commitments for verification
	for j := 0; j < len(v.Token); j++ {
		ver := &common.SchnorrVerifier{PedParams: v.PedersenParams, Curve: v.Curve}
		zkp := &common.SchnorrProof{Statement: v.Token[j], Proof: []*mathlib.Zr{p.EqualityProofs.Type, p.EqualityProofs.Value[j], p.EqualityProofs.TokenBlindingFactor[j]}, Challenge: p.Challenge}
		c.Token = append(c.Token, ver.RecomputeCommitment(zkp))
	}

	for j := 0; j < len(v.Token); j++ {
		com := v.Curve.NewG1()
		for i := 0; i < v.Exponent; i++ {
			pow := v.Curve.NewZrFromInt(int64(math.Pow(float64(v.Base), float64(i))))
			com.Add(p.MembershipProofs[j].Commitments[i].Mul(pow))
		}

		ver := &common.SchnorrVerifier{PedParams: v.PedersenParams[:2], Curve: v.Curve}
		zkp := &common.SchnorrProof{Statement: com, Proof: []*mathlib.Zr{p.EqualityProofs.Value[j], p.EqualityProofs.CommitmentBlindingFactor[j]}, Challenge: p.Challenge}
		c.CommitmentToValue = append(c.CommitmentToValue, ver.RecomputeCommitment(zkp))
	}
	return c
}
