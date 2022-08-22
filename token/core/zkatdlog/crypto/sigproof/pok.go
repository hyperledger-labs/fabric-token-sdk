/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package sigproof

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"
	"github.com/pkg/errors"
)

// POK is a zero-knowledge proof of knowledge of a Pointcheval-Sanders Signature
type POK struct {
	// Challenge is the challenge used in the proof
	Challenge *math.Zr
	// Signature is an obfuscated Pointcheval-Sanders signature
	Signature *pssign.Signature
	// Proof of signed messages
	Messages []*math.Zr
	// BlindingFactor is randomness used to obfuscate Pointcheval-Sadners signature
	BlindingFactor *math.Zr
	// Proof of hash (hash is computed as the hash of the signed messages)
	Hash *math.Zr
}

// POKWitness encodes the witness of POK proof
type POKWitness struct {
	// Messages corresponds to signed messages
	Messages []*math.Zr
	// Signature is Pointcheval-Sanders signature
	Signature *pssign.Signature
	// BlindingFactor is the randomness used to obfuscate Pointcheval-Sanders signature
	// for the POK proof
	BlindingFactor *math.Zr
}

// POKRandomness is the Randomness used during the POK proof
type POKRandomness struct {
	messages       []*math.Zr
	hash           *math.Zr
	blindingFactor *math.Zr
}

// POKProver produces a POK proof
type POKProver struct {
	*POKVerifier
	Witness    *POKWitness
	randomness *POKRandomness
}

// POKVerifier validates if a POK proof is valid
type POKVerifier struct {
	// PK is the public key under which the signature should be valid
	PK    []*math.G2
	Q     *math.G2
	P     *math.G1
	Curve *math.Curve
}

// Prove returns a POK proof
func (p *POKProver) Prove() (*POK, error) {
	// obfuscate signature
	sig, err := p.obfuscateSignature()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate proof of knowledge of Pointcheval-Sanders signature")
	}

	// generate and compute commitment to randomness that will be used in the proof
	com, err := p.computeCommitment()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof of knowledge of Pointcheval-Sanders signature")
	}

	// compute challenge
	chal, err := p.computeChallenge(com, sig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate proof of knowledge of Pointcheval-Sanders signature")
	}
	hash, err := HashMessages(p.Witness.Messages, p.Curve)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate proof of knowledge of Pointcheval-Sanders signature")
	}
	// generate schnorr proof
	sprover := &common.SchnorrProver{Witness: common.GetZrArray(p.Witness.Messages, []*math.Zr{hash, p.Witness.BlindingFactor}), Randomness: common.GetZrArray(p.randomness.messages, []*math.Zr{p.randomness.hash, p.randomness.blindingFactor}), Challenge: chal, SchnorrVerifier: &common.SchnorrVerifier{Curve: p.Curve}}
	sp, err := sprover.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute proof of knowledge of Pointcheval-Sanders signature")
	}
	// return proof of knowledge
	return &POK{
		Challenge:      chal,
		Signature:      sig,
		Messages:       sp[:len(p.Witness.Messages)],
		Hash:           sp[len(p.Witness.Messages)],
		BlindingFactor: sp[len(p.Witness.Messages)+1]}, nil
}
func (p *POKProver) computeCommitment() (*math.Gt, error) {
	if p.Curve == nil {
		return nil, errors.New("please initialize curve")
	}
	// Get Random number generator
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, errors.New("failed to get random number generator")
	}
	// compute commitment
	p.randomness = &POKRandomness{}
	p.randomness.hash = p.Curve.NewRandomZr(rand)
	if p.PK[len(p.Witness.Messages)+1] == nil || p.randomness.hash == nil {
		return nil, errors.New("please initialize prover correctly")
	}
	t := p.PK[len(p.Witness.Messages)+1].Mul(p.randomness.hash)
	p.randomness.messages = make([]*math.Zr, len(p.Witness.Messages))
	for i := 0; i < len(p.Witness.Messages); i++ {
		p.randomness.messages[i] = p.Curve.NewRandomZr(rand)
		if p.PK[i+1] == nil {
			return nil, errors.New("please initialize prover correctly")
		}
		t.Add(p.PK[i+1].Mul(p.randomness.messages[i]))
	}

	p.randomness.blindingFactor = p.Curve.NewRandomZr(rand)
	if p.Witness == nil || p.Witness.Signature == nil || p.Witness.Signature.R == nil {
		return nil, errors.New("please initialize prover's witness correctly")
	}
	if p.Q == nil || p.P == nil {
		return nil, errors.New("please initialize prover correctly")
	}
	com := p.Curve.Pairing2(t, p.Witness.Signature.R, p.Q, p.P.Mul(p.randomness.blindingFactor))

	return p.Curve.FExp(com), nil
}

// Verify checks if the passed POK is valid; if not, Verify returns an error
func (v *POKVerifier) Verify(proof *POK) error {

	// recompute commitments to randomness used in the POK proof
	com, err := v.recomputeCommitment(proof)
	if err != nil {
		return errors.Wrapf(err, "failed to verify POK of PS signature")
	}

	// recompute challenge
	chal, err := v.computeChallenge(com, proof.Signature)
	if err != nil {
		return errors.Wrapf(err, "failed to verify POK of PS signature")
	}

	// check if proof is valid
	if !proof.Challenge.Equals(chal) {
		return errors.Errorf("proof of PS signature is not valid")
	}
	return nil
}

// recomputeCommitment returns the commitment to the randomness used in the passed POK proof
func (v *POKVerifier) recomputeCommitment(p *POK) (*math.Gt, error) {
	if v.Curve == nil {
		return nil, errors.New("please initialize curve")
	}
	if len(v.PK) != len(p.Messages)+2 {
		return nil, errors.New("length of signature public key does not match size of proof")
	}
	t := v.Curve.NewG2()
	for i := 0; i < len(p.Messages); i++ {
		if p.Messages[i] == nil {
			return nil, errors.New("nil elements")
		}
		if v.PK[i+1] == nil {
			return nil, errors.New("please initialize verifier correctly")
		}
		t.Add(v.PK[i+1].Mul(p.Messages[i]))
	}
	if p.Hash == nil {
		return nil, errors.New("nil hash")
	}
	if v.PK[len(p.Messages)+1] == nil {
		return nil, errors.New("please initialize verifier correctly")
	}
	t.Add(v.PK[len(p.Messages)+1].Mul(p.Hash))

	pk := v.Curve.NewG2()
	if v.PK[0] == nil {
		return nil, errors.New("please initialize verifier correctly")
	}
	pk.Sub(v.PK[0])
	if p.Signature == nil || p.Signature.R == nil || p.Signature.S == nil {
		return nil, errors.New("nil elements")
	}
	if p.Challenge == nil || p.BlindingFactor == nil {
		return nil, errors.New("nil elements")
	}
	if v.Q == nil || v.P == nil {
		return nil, errors.New("please initialize verifier correctly")
	}
	com := v.Curve.Pairing2(v.Q, p.Signature.S.Mul(p.Challenge), pk, p.Signature.R.Mul(p.Challenge))
	com.Inverse()
	com.Mul(v.Curve.Pairing2(t, p.Signature.R, v.Q, v.P.Mul(p.BlindingFactor)))

	return v.Curve.FExp(com), nil
}

// HashMessages returns a hash of the passed array of messages
func HashMessages(m []*math.Zr, c *math.Curve) (*math.Zr, error) {
	var bytesToHash []byte
	for i := 0; i < len(m); i++ {
		if m[i] == nil {
			return nil, errors.New("can't compute hash: nil elements")
		}
		bytes := m[i].Bytes()
		bytesToHash = append(bytesToHash, bytes...)
	}

	return c.HashToZr(bytesToHash), nil
}

// obfuscateSignature returns an obfuscated Pointcheval-Sanders signature
// this is a pair (R', S') = (R, S*PedGen^r) where (R, S) is a Pointcheval-Sanders signature
// of the POKProver
func (p *POKProver) obfuscateSignature() (*pssign.Signature, error) {
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, errors.Errorf("failed to get RNG")
	}

	p.Witness.BlindingFactor = p.Curve.NewRandomZr(rand)
	v := pssign.NewVerifier(nil, nil, p.Curve)
	err = v.Randomize(p.Witness.Signature)
	if err != nil {
		return nil, err
	}
	sig := &pssign.Signature{}
	sig.Copy(p.Witness.Signature)
	if sig.S == nil {
		return nil, errors.New("invalid witness signature")
	}
	if p.P == nil {
		return nil, errors.New("please initialize prover correctly")
	}
	sig.S.Add(p.P.Mul(p.Witness.BlindingFactor))

	return sig, nil
}

// computeChallenge returns the challenge to be used in the POK proof
func (v *POKVerifier) computeChallenge(com *math.Gt, signature *pssign.Signature) (*math.Zr, error) {
	// serialize public inputs
	g2a, err := common.GetG2Array(v.PK, []*math.G2{v.Q}).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute challenge")
	}
	bytes, err := signature.Serialize()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compute challenge")
	}
	// compute challenge
	return v.Curve.HashToZr(common.GetBytesArray(v.P.Bytes(), g2a, bytes, com.Bytes())), nil

}
