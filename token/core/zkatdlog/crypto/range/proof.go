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

// RangeProof is a range proof that  show that an array of token have value < max_value
// max_value = base^exponent - 1
// token_value = \sum_{i=0}^exponent v_i base^i and 0=<v_i =<base-1
type RangeProof struct {
	// Challenge used to compute the proof
	Challenge *mathlib.Zr
	// EqualityProofs show that for each token in an array of tokens,
	// token_value = \sum_{i=0}^exponent v_i base^i
	EqualityProofs *EqualityProofs
	// MembershipProofs show that  0=<v_i =<base-1
	MembershipProofs []*MembershipProof
}

// EqualityProofs show that for each token in an array of tokens,
// token_value = \sum_{i=0}^exponent v_i base^i
type EqualityProofs struct {
	// Type of tokens
	Type *mathlib.Zr
	// Value is an array of elements in Zr such that Value[i] is the value of the i^th token
	Value []*mathlib.Zr
	// TokenBlindingFactor is an array of elements in Zr such that
	// TokenBlindingFactor[i] is the blinding factor of the i^th token
	TokenBlindingFactor []*mathlib.Zr
	// CommitmentBlindingFactor is an array of elements in Zr such that
	// CommitmentBlindingFactor[i] is the blinding factor of the i^th commitment
	CommitmentBlindingFactor []*mathlib.Zr
}

// MembershipProof shows that committed values 0=<v_i =<max_value, for 1 =< i =< n
type MembershipProof struct {
	// Commitments is an array of Pedersen commitments
	Commitments []*mathlib.G1
	// SignatureProofs is ZK proof that each committed value is signed
	// using Pointcheval-Sanders signature
	SignatureProofs []*sigproof.MembershipProof
}

// Prover produces a proof that show that values of tokens is < max_value
type Prover struct {
	*Verifier
	// tokenWitness is the opening of a TokenData (type, value, blinding factor)
	tokenWitness []*token.TokenDataWitness
	// Signatures are an array of Pointcheval-Sanders signatures
	Signatures []*pssign.Signature
}

// NewProver returns a Prover
func NewProver(tw []*token.TokenDataWitness, token []*mathlib.G1, signatures []*pssign.Signature, exponent int, pp []*mathlib.G1, PK []*mathlib.G2, P *mathlib.G1, Q *mathlib.G2, c *mathlib.Curve) *Prover {
	return &Prover{
		tokenWitness: tw,
		Signatures:   signatures,
		Verifier: &Verifier{
			Tokens:         token,
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

// Verifier checks the validity of range proofs produced by Prover
type Verifier struct {
	// Tokens is an array of TokenData - commitment to (value, type)
	Tokens []*mathlib.G1
	// max_value = Base^Exponent
	Base     uint64
	Exponent int
	// PedersenParams corresponds to the Pedersen commitment generators
	PedersenParams []*mathlib.G1
	// Q is a random G2 generator
	Q *mathlib.G2
	// P is a random G1 generator
	P *mathlib.G1
	// PK is the public key of Pointcheval-Sanders signature
	PK []*mathlib.G2
	// Curve is an elliptic curve
	Curve *mathlib.Curve
}

// NewVerifier returns a range proof Verifier
func NewVerifier(token []*mathlib.G1, base uint64, exponent int, pp []*mathlib.G1, PK []*mathlib.G2, P *mathlib.G1, Q *mathlib.G2, c *mathlib.Curve) *Verifier {
	return &Verifier{
		Tokens:         token,
		Base:           base,
		Exponent:       exponent,
		PedersenParams: pp,
		PK:             PK,
		P:              P,
		Q:              Q,
		Curve:          c,
	}
}

// randomness is the randomness used in the range proof
type randomness struct {
	tokenType                         *mathlib.Zr
	values                            []*mathlib.Zr
	tokensBlindingFactors             []*mathlib.Zr
	commitmentsToValueBlindingFactors []*mathlib.Zr
}

// Commitment is the commitment to the randomness used in the range proof
type Commitment struct {
	Tokens              []*mathlib.G1
	CommitmentsToValues []*mathlib.G1
}

// rangeProofWitness is the secret information needed to generate a range proof
type rangeProofWitness struct {
	commitmentsToValues                [][]*mathlib.G1
	membershipWitnesses                [][]*sigproof.MembershipWitness
	commitmentsToValuesBlindingFactors []*mathlib.Zr
}

// Prove generates a range proof
func (p *Prover) Prove() ([]byte, error) {
	proof := &RangeProof{}
	var err error
	preProcessed, err := p.preProcess()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	wg.Add(len(p.Tokens) * p.Exponent)

	parallelErr := &atomic.Value{}

	// produce proof that each committed value is signed
	proof.MembershipProofs = make([]*MembershipProof, len(p.Tokens))
	for k := 0; k < len(proof.MembershipProofs); k++ {
		proof.MembershipProofs[k] = &MembershipProof{}
		proof.MembershipProofs[k].Commitments = make([]*mathlib.G1, p.Exponent)
		proof.MembershipProofs[k].SignatureProofs = make([]*sigproof.MembershipProof, p.Exponent)
		for i := 0; i < p.Exponent; i++ {
			proof.MembershipProofs[k].Commitments[i] = preProcessed.commitmentsToValues[k][i]
			mp := sigproof.NewMembershipProver(preProcessed.membershipWitnesses[k][i], proof.MembershipProofs[k].Commitments[i], p.P, p.Q, p.PK, p.PedersenParams[:2], p.Curve)

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

	// show that value in token = \prod_{i=0}^Exponent com_i^{Base^i}
	commitment, randomness, err := p.computeCommitment()
	if err != nil {
		return nil, err
	}

	proof.Challenge, err = p.computeChallenge(commitment, preProcessed.commitmentsToValues)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compute range proof")
	}

	// equality proof
	// value in token = \prod_{i=0}^Exponent com_i^{Base^i}
	proof.EqualityProofs = &EqualityProofs{}
	for k := 0; k < len(p.Tokens); k++ {
		sp := &common.SchnorrProver{Challenge: proof.Challenge, Randomness: []*mathlib.Zr{randomness.values[k], randomness.tokensBlindingFactors[k], randomness.commitmentsToValueBlindingFactors[k]}, Witness: []*mathlib.Zr{p.tokenWitness[k].Value, p.tokenWitness[k].BlindingFactor, preProcessed.commitmentsToValuesBlindingFactors[k]}, SchnorrVerifier: &common.SchnorrVerifier{Curve: p.Curve}}
		proofs, err := sp.Prove()
		if err != nil {
			return nil, err
		}
		proof.EqualityProofs.Value = append(proof.EqualityProofs.Value, proofs[0])
		proof.EqualityProofs.TokenBlindingFactor = append(proof.EqualityProofs.TokenBlindingFactor, proofs[1])
		proof.EqualityProofs.CommitmentBlindingFactor = append(proof.EqualityProofs.CommitmentBlindingFactor, proofs[2])
	}
	proof.EqualityProofs.Type = p.Curve.ModMul(proof.Challenge, p.Curve.HashToZr([]byte(p.tokenWitness[0].Type)), p.Curve.GroupOrder)
	proof.EqualityProofs.Type = p.Curve.ModAdd(proof.EqualityProofs.Type, randomness.tokenType, p.Curve.GroupOrder)

	return json.Marshal(proof)
}

func (v *Verifier) Verify(raw []byte) error {
	// todo check length of public parameters
	proof := &RangeProof{}
	err := json.Unmarshal(raw, proof)
	if err != nil {
		return err
	}

	if len(proof.MembershipProofs) != len(v.Tokens) {
		return errors.Errorf("range proof not well formed")
	}

	var verifications []func()
	parallelErr := &atomic.Value{}

	// verify membership
	// each committed value v_i is signed (i.e., v_i < Base)
	for k := 0; k < len(v.Tokens); k++ {
		if proof.MembershipProofs[k] == nil {
			return errors.Errorf("range proof not well formed")
		}
		if len(proof.MembershipProofs[k].Commitments) != len(proof.MembershipProofs[k].SignatureProofs) {
			return errors.Errorf("range proof not well formed")
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

	// verify equality proof
	// value in token = \prod_{i=0}^Exponent com_i^{Base^i}
	com, err := v.recomputeCommitments(proof)
	if err != nil {
		return err
	}
	coms := make([][]*mathlib.G1, len(proof.MembershipProofs))
	for i := 0; i < len(proof.MembershipProofs); i++ {
		for k := 0; k < len(proof.MembershipProofs[i].Commitments); k++ {
			coms[i] = append(coms[i], proof.MembershipProofs[i].Commitments[k])
		}
	}
	chal, err := v.computeChallenge(com, coms)
	if err != nil {
		return errors.Wrap(err, "failed to verify range proof")
	}
	if !chal.Equals(proof.Challenge) {
		return errors.Errorf("invalid range proof")
	}

	return nil
}

// preProcess computes commitment to values v_i such that v = v = \sum_{i=0}^Exponent values[i] Base^i
// preProcess returns the corresponding rangeProofWitness
func (p *Prover) preProcess() (*rangeProofWitness, error) {
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
		// v = \sum_{i=0}^Exponent values[i] Base^i
		values[0] = int(v % int64(p.Base))
		for i := 0; i < p.Exponent-1; i++ {
			values[p.Exponent-1-i] = int(v / int64(math.Pow(float64(p.Base), float64(p.Exponent-1-i)))) // quotient
			v = v % int64(math.Pow(float64(p.Base), float64(p.Exponent-1-i)))                           // remainder
		}

		// generate witness for membership of values[i]
		// membership corresponds to a Pointcheval-Sanders signature
		membershipWitness[k] = make([]*sigproof.MembershipWitness, p.Exponent)
		commitmentBlindingFactor[k] = p.Curve.NewZrFromInt(0)
		coms[k] = make([]*mathlib.G1, p.Exponent)
		for i := 0; i < p.Exponent; i++ {
			bf := p.Curve.NewRandomZr(rand)
			// compute Pedersen commitment to values[i]
			coms[k][i], err = common.ComputePedersenCommitment([]*mathlib.Zr{p.Curve.NewZrFromInt(int64(values[i])), bf}, p.PedersenParams[:2], p.Curve)
			if err != nil {
				return nil, err
			}
			// membershipWitness contains Pointcheval-Sanders signature of values[i], values[i] and the blinding factor used to compute the commitment to values[i]
			membershipWitness[k][i] = sigproof.NewMembershipWitness(p.Signatures[values[i]], p.Curve.NewZrFromInt(int64(values[i])), bf)
			pow := p.Curve.NewZrFromInt(int64(math.Pow(float64(p.Base), float64(i))))
			// this is the blinding factor to commitment  c = \prod_{i=0}^Exponent com[k][i]^{Base^i}
			commitmentBlindingFactor[k] = p.Curve.ModAdd(commitmentBlindingFactor[k], p.Curve.ModMul(bf, pow, p.Curve.GroupOrder), p.Curve.GroupOrder)
		}
	}
	return &rangeProofWitness{
		commitmentsToValues:                coms,
		commitmentsToValuesBlindingFactors: commitmentBlindingFactor,
		membershipWitnesses:                membershipWitness,
	}, nil
}

// computeCommitment generates range proof randomness and computes the corresponding commitments
func (p *Prover) computeCommitment() (*Commitment, *randomness, error) {
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, nil, err
	}
	// generate randomness
	randomness := &randomness{}
	randomness.tokenType = p.Curve.NewRandomZr(rand)
	for i := 0; i < len(p.Tokens); i++ {
		randomness.values = append(randomness.values, p.Curve.NewRandomZr(rand))
		randomness.commitmentsToValueBlindingFactors = append(randomness.commitmentsToValueBlindingFactors, p.Curve.NewRandomZr(rand))
		randomness.tokensBlindingFactors = append(randomness.tokensBlindingFactors, p.Curve.NewRandomZr(rand))
	}

	// compute commitment
	commitment := &Commitment{}
	for i := 0; i < len(p.tokenWitness); i++ {
		tok := p.PedersenParams[0].Mul(randomness.tokenType)
		tok.Add(p.PedersenParams[1].Mul(randomness.values[i]))
		tok.Add(p.PedersenParams[2].Mul(randomness.tokensBlindingFactors[i]))
		commitment.Tokens = append(commitment.Tokens, tok)

		com := p.PedersenParams[0].Mul(randomness.values[i])
		com.Add(p.PedersenParams[1].Mul(randomness.commitmentsToValueBlindingFactors[i]))
		commitment.CommitmentsToValues = append(commitment.CommitmentsToValues, com)
	}

	return commitment, randomness, nil
}

// computeChallenge computes the challenge for the range proof
func (v *Verifier) computeChallenge(commitment *Commitment, comToValue [][]*mathlib.G1) (*mathlib.Zr, error) {
	g1array, err := common.GetG1Array([]*mathlib.G1{v.P}, v.Tokens, commitment.Tokens, commitment.CommitmentsToValues, v.PedersenParams).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute challenge")
	}
	g2array, err := common.GetG2Array([]*mathlib.G2{v.Q}, v.PK).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute challenge")
	}
	bytes := append(g1array, g2array...)
	for i := 0; i < len(comToValue); i++ {
		raw, err := common.GetG1Array(comToValue[i]).Bytes()
		if err != nil {
			return nil, errors.Wrap(err, "failed to compute challenge")
		}
		bytes = append(bytes, raw...)
	}
	return v.Curve.HashToZr(bytes), nil
}

// recomputeCommitments computes the commitment to randomness used in the proof as a function
// of the proof
func (v *Verifier) recomputeCommitments(p *RangeProof) (*Commitment, error) {
	if p.EqualityProofs == nil {
		return nil, errors.Errorf("range proof not well formed")
	}
	if len(p.EqualityProofs.Value) != len(v.Tokens) {
		return nil, errors.Errorf("range proof not well formed")
	}
	if len(p.EqualityProofs.TokenBlindingFactor) != len(v.Tokens) {
		return nil, errors.Errorf("range proof not well formed")
	}
	if len(p.EqualityProofs.CommitmentBlindingFactor) != len(v.Tokens) {
		return nil, errors.Errorf("range proof not well formed")
	}

	c := &Commitment{}
	// recompute commitments for verification
	for j := 0; j < len(v.Tokens); j++ {
		ver := &common.SchnorrVerifier{PedParams: v.PedersenParams, Curve: v.Curve}
		zkp := &common.SchnorrProof{Statement: v.Tokens[j], Proof: []*mathlib.Zr{p.EqualityProofs.Type, p.EqualityProofs.Value[j], p.EqualityProofs.TokenBlindingFactor[j]}, Challenge: p.Challenge}
		com, err := ver.RecomputeCommitment(zkp)
		if err != nil {
			return nil, err
		}
		c.Tokens = append(c.Tokens, com)
	}

	for j := 0; j < len(v.Tokens); j++ {
		com := v.Curve.NewG1()
		if p.MembershipProofs[j] == nil {
			return nil, errors.Errorf("range proof not well formed")
		}
		if len(p.MembershipProofs[j].Commitments) != v.Exponent {
			return nil, errors.Errorf("range proof not well formed")
		}
		for i := 0; i < v.Exponent; i++ {
			pow := v.Curve.NewZrFromInt(int64(math.Pow(float64(v.Base), float64(i))))
			if p.MembershipProofs[j].Commitments == nil {
				return nil, errors.Errorf("range proof not well formed")
			}
			com.Add(p.MembershipProofs[j].Commitments[i].Mul(pow))
		}

		ver := &common.SchnorrVerifier{PedParams: v.PedersenParams[:2], Curve: v.Curve}
		zkp := &common.SchnorrProof{Statement: com, Proof: []*mathlib.Zr{p.EqualityProofs.Value[j], p.EqualityProofs.CommitmentBlindingFactor[j]}, Challenge: p.Challenge}
		com, err := ver.RecomputeCommitment(zkp)
		if err != nil {
			return nil, err
		}
		c.CommitmentsToValues = append(c.CommitmentsToValues, com)
	}
	return c, nil
}
