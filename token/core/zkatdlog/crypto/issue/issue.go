/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issue

import (
	"encoding/json"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	rp "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/rp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

// IssueAction specifies an issue of one or more tokens
type IssueAction struct {
	// Issuer is the identity of issuer
	Issuer []byte
	// OutputTokens are the newly issued tokens
	OutputTokens []*token.Token `protobuf:"bytes,1,rep,name=outputs,proto3" json:"outputs,omitempty"`
	// Proof carries the ZKP of IssueAction validity
	Proof []byte
	// flag to indicate whether the Issuer is anonymous or not
	Anonymous bool
	// Metadata of the issue action
	Metadata map[string][]byte
}

// GetProof returns IssueAction ZKP
func (i *IssueAction) GetProof() []byte {
	return i.Proof
}

// GetMetadata returns IssueAction metadata if there is any.
func (i *IssueAction) GetMetadata() map[string][]byte {
	return i.Metadata
}

// IsAnonymous returns a Boolean. True if IssueAction is anonymous, and False otherwise.
func (i *IssueAction) IsAnonymous() bool {
	return i.Anonymous
}

// Serialize marshal IssueAction
func (i *IssueAction) Serialize() ([]byte, error) {
	return json.Marshal(i)
}

// NumOutputs returns the number of outputs in IssueAction
func (i *IssueAction) NumOutputs() int {
	return len(i.OutputTokens)
}

// GetOutputs returns the OutputTokens in IssueAction
func (i *IssueAction) GetOutputs() []driver.Output {
	var res []driver.Output
	for _, token := range i.OutputTokens {
		res = append(res, token)
	}
	return res
}

// GetSerializedOutputs returns the serialization of OutputTokens
func (i *IssueAction) GetSerializedOutputs() ([][]byte, error) {
	var res [][]byte
	for _, token := range i.OutputTokens {
		if token == nil {
			return nil, errors.New("invalid issue: there is a nil output")
		}
		r, err := token.Serialize()
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

// GetIssuer returns the Issuer of IssueAction
func (i *IssueAction) GetIssuer() []byte {
	return i.Issuer
}

// Deserialize un-marshals IssueAction
func (i *IssueAction) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, i)
}

// GetCommitments return the Pedersen commitment of (type, value) in the OutputTokens
func (i *IssueAction) GetCommitments() ([]*math.G1, error) {
	com := make([]*math.G1, len(i.OutputTokens))
	for j := 0; j < len(com); j++ {
		if i.OutputTokens[j] == nil {
			return nil, errors.New("invalid issue: there is a nil output")
		}
		com[j] = i.OutputTokens[j].Data
	}
	return com, nil
}

// NewIssue instantiates an IssueAction given the passed arguments
func NewIssue(issuer []byte, coms []*math.G1, owners [][]byte, proof []byte, anonymous bool) (*IssueAction, error) {
	if len(owners) != len(coms) {
		return nil, errors.New("number of owners does not match number of tokens")
	}

	outputs := make([]*token.Token, len(coms))
	for i, c := range coms {
		outputs[i] = &token.Token{Owner: owners[i], Data: c}
	}

	return &IssueAction{
		Issuer:       issuer,
		OutputTokens: outputs,
		Proof:        proof,
		Anonymous:    anonymous,
	}, nil
}

// Proof poves that an IssueAction is valid
type Proof struct {
	// SameType is the proof that issued tokens are well-formed
	// tokens contain a commitment to type and value
	SameType *SameType
	// RangeCorrectness is the proof that issued tokens have value in the authorized range
	RangeCorrectness *rp.RangeCorrectness
}

// Serialize marshals Proof
func (p *Proof) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

// Deserialize un-marshals Proof
func (p *Proof) Deserialize(bytes []byte) error {
	return json.Unmarshal(bytes, p)
}

// Prover produces a proof of validity of an IssueAction
type Prover struct {
	// SameType encodes the SameType Prover
	SameType *SameTypeProver
	// RangeCorrectness encodes the range proof Prover
	RangeCorrectness *rp.RangeCorrectnessProver
}

func NewProver(tw []*token.TokenDataWitness, tokens []*math.G1, anonymous bool, pp *crypto.PublicParams) (*Prover, error) {
	c := math.Curves[pp.Curve]
	p := &Prover{}
	tokenType := c.HashToZr([]byte(tw[0].Type))
	com := pp.PedParams[0].Mul(tokenType)
	if anonymous {
		rand, err := c.Rand()
		if err != nil {
			return nil, errors.Wrapf(err, "cannot get issue prover")
		}
		typeBF := c.NewRandomZr(rand)
		com.Add(pp.PedParams[2].Mul(typeBF))
		p.SameType = NewSameTypeProver(tw[0].Type, typeBF, com, anonymous, pp.PedParams, c)

	} else {
		p.SameType = NewSameTypeProver(tw[0].Type, c.NewZrFromInt(0), com, anonymous, pp.PedParams, c)

	}
	var values []uint64
	var blindingFactors []*math.Zr
	for i := 0; i < len(tw); i++ {
		if tw[i] == nil || tw[i].BlindingFactor == nil {
			return nil, errors.New("invalid token witness")
		}
		tw[i] = tw[i].Clone()
		values = append(values, tw[i].Value)
		blindingFactors = append(blindingFactors, c.ModSub(tw[i].BlindingFactor, p.SameType.blindingFactor, c.GroupOrder))
	}
	var coms []*math.G1
	for i := 0; i < len(tokens); i++ {
		token := tokens[i].Copy()
		token.Sub(com)
		coms = append(coms, token)
	}
	p.RangeCorrectness = rp.NewRangeCorrectnessProver(coms, values, blindingFactors, pp.PedParams[1:], pp.RangeProofParams.LeftGenerators, pp.RangeProofParams.RightGenerators, pp.RangeProofParams.P, pp.RangeProofParams.Q, pp.RangeProofParams.BitLength, pp.RangeProofParams.NumberOfRounds, math.Curves[pp.Curve])

	return p, nil
}

// Prove produces a Proof for an IssueAction
func (p *Prover) Prove() ([]byte, error) {
	// TypeAndSum proof
	st, err := p.SameType.Prove()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate issue proof")
	}

	// RangeCorrectness proof
	rc, err := p.RangeCorrectness.Prove()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate range proof for issue")
	}

	proof := &Proof{
		SameType:         st,
		RangeCorrectness: rc,
	}
	return proof.Serialize()
}

// Verifier checks if Proof is valid
type Verifier struct {
	// SameType encodes the SameType Verifier
	SameType *SameTypeVerifier
	// RangeCorrectness encodes the range proof verifier
	RangeCorrectness *rp.RangeCorrectnessVerifier
}

func NewVerifier(tokens []*math.G1, anonymous bool, pp *crypto.PublicParams) *Verifier {
	v := &Verifier{}
	v.SameType = NewSameTypeVerifier(tokens, anonymous, pp.PedParams, math.Curves[pp.Curve])
	v.RangeCorrectness = rp.NewRangeCorrectnessVerifier(pp.PedParams[1:], pp.RangeProofParams.LeftGenerators, pp.RangeProofParams.RightGenerators, pp.RangeProofParams.P, pp.RangeProofParams.Q, pp.RangeProofParams.BitLength, pp.RangeProofParams.NumberOfRounds, math.Curves[pp.Curve])
	return v
}

// Verify returns an error if Proof of an IssueAction is invalid
func (v *Verifier) Verify(proof []byte) error {
	tp := &Proof{}
	// unmarshal proof
	err := tp.Deserialize(proof)
	if err != nil {
		return err
	}
	// verify TypeAndSum proof
	err = v.SameType.Verify(tp.SameType)
	if err != nil {
		return errors.Wrapf(err, "invalid issue proof")
	}
	// verify RangeCorrectness proof
	commitmentToType := tp.SameType.CommitmentToType.Copy()
	var coms []*math.G1
	for i := 0; i < len(v.SameType.Tokens); i++ {
		token := v.SameType.Tokens[i].Copy()
		token.Sub(commitmentToType)
		coms = append(coms, token)
	}
	v.RangeCorrectness.Commitments = coms
	err = v.RangeCorrectness.Verify(tp.RangeCorrectness)
	if err != nil {
		return errors.Wrapf(err, "invalid issue proof")
	}
	return nil
}
