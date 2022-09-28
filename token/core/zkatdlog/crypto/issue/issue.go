/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package issue

import (
	"encoding/json"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	rp "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/range"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

// IssueAction specifies an issue of one or more tokens
type IssueAction struct {
	// Identity of issuer
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
	// proof that issued tokens are well-formed
	// tokens contain a commitment to type and value
	WellFormedness []byte
	// proof that issued tokens have value in the authorized range
	RangeCorrectness []byte
}

// Serialize marshals Proof
func (p *Proof) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

// Deserialize unmarshals Proof
func (p *Proof) Deserialize(bytes []byte) error {
	return json.Unmarshal(bytes, p)
}

// Prover produces a proof of validity of an IssueAction
type Prover struct {
	// WellFormedness encodes the WellFormedness Prover
	WellFormedness *WellFormednessProver
	// RangeCorrectness encodes the range proof Prover
	RangeCorrectness *rp.Prover
}

func NewProver(tw []*token.TokenDataWitness, tokens []*math.G1, anonymous bool, pp *crypto.PublicParams) *Prover {
	c := math.Curves[pp.Curve]
	p := &Prover{}
	p.WellFormedness = NewWellFormednessProver(tw, tokens, anonymous, pp.PedParams, c)

	p.RangeCorrectness = rp.NewProver(tw, tokens, pp.RangeProofParams.SignedValues, pp.RangeProofParams.Exponent, pp.PedParams, pp.RangeProofParams.SignPK, pp.PedGen, pp.RangeProofParams.Q, math.Curves[pp.Curve])

	return p
}

// Prove produces a Proof for an IssueAction
func (p *Prover) Prove() ([]byte, error) {
	// checks
	if p.WellFormedness == nil || p.RangeCorrectness == nil {
		return nil, errors.New("please initialize issue action prover correctly")
	}
	// WellFormedness proof
	wf, err := p.WellFormedness.Prove()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate issue proof")
	}

	// RangeCorrectness proof
	rc, err := p.RangeCorrectness.Prove()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate range proof for issue")
	}

	proof := &Proof{
		WellFormedness:   wf,
		RangeCorrectness: rc,
	}
	return proof.Serialize()
}

// Verifier checks if Proof is valid
type Verifier struct {
	// WellFormedness encodes the WellFormedness Verifier
	WellFormedness *WellFormednessVerifier
	// RangeCorrectness encodes the range proof verifier
	RangeCorrectness *rp.Verifier
}

func NewVerifier(tokens []*math.G1, anonymous bool, pp *crypto.PublicParams) *Verifier {
	v := &Verifier{}
	v.WellFormedness = NewWellFormednessVerifier(tokens, anonymous, pp.PedParams, math.Curves[pp.Curve])
	v.RangeCorrectness = rp.NewVerifier(tokens, uint64(len(pp.RangeProofParams.SignedValues)), pp.RangeProofParams.Exponent, pp.PedParams, pp.RangeProofParams.SignPK, pp.PedGen, pp.RangeProofParams.Q, math.Curves[pp.Curve])
	return v
}

// Verify returns an error if Proof of an IssueAction is invalid
func (v *Verifier) Verify(proof []byte) error {
	if v.RangeCorrectness == nil || v.WellFormedness == nil {
		return errors.New("please initialize issue action verifier correctly")
	}
	ip := &Proof{}
	// unmarshal proof
	err := ip.Deserialize(proof)
	if err != nil {
		return err
	}
	// verify WellFormedness proof
	err = v.WellFormedness.Verify(ip.WellFormedness)
	if err != nil {
		return errors.Wrapf(err, "invalid issue proof")
	}
	// verify RangeCorrectness proof
	err = v.RangeCorrectness.Verify(ip.RangeCorrectness)
	if err != nil {
		return errors.Wrapf(err, "invalid issue proof")
	}
	return nil
}
