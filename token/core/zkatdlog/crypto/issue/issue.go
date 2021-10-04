/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package issue

import (
	"encoding/json"

	"github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	rp "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/range"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

// Issue specifies an issue of one or more tokens
type IssueAction struct {
	Issuer []byte
	// OutputTokens are the newly issued tokens
	OutputTokens []*token.Token `protobuf:"bytes,1,rep,name=outputs,proto3" json:"outputs,omitempty"`
	// ZK proof
	Proof []byte
	// flag to indicate type of issue
	Anonymous bool
}

func (i *IssueAction) GetProof() []byte {
	return i.Proof
}

func (i *IssueAction) GetMetadata() []byte {
	return nil
}

func (i *IssueAction) IsAnonymous() bool {
	return i.Anonymous
}

func (i *IssueAction) Serialize() ([]byte, error) {
	return json.Marshal(i)
}

func (i *IssueAction) NumOutputs() int {
	return len(i.OutputTokens)
}

func (i *IssueAction) GetOutputs() []driver.Output {
	var res []driver.Output
	for _, token := range i.OutputTokens {
		res = append(res, token)
	}
	return res
}

func (i *IssueAction) GetSerializedOutputs() ([][]byte, error) {
	var res [][]byte
	for _, token := range i.OutputTokens {
		r, err := token.Serialize()
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (i *IssueAction) GetIssuer() []byte {
	return i.Issuer
}

func (i *IssueAction) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, i)
}

func (i *IssueAction) GetCommitments() []*math.G1 {
	com := make([]*math.G1, len(i.OutputTokens))
	for j := 0; j < len(com); j++ {
		com[j] = i.OutputTokens[j].Data
	}
	return com
}

// Initialize Issue
func NewIssue(issuer []byte, coms []*math.G1, owners [][]byte, proof []byte, anonymous bool) (*IssueAction, error) {
	if len(owners) != len(coms) {
		return nil, errors.Errorf("number of owners does not match number of tokens")
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

// zkat proof in Issue
type Proof struct {
	WellFormedness   []byte // proof that issued tokens are well-formed
	RangeCorrectness []byte // proof that issued tokens have value in the authorized range
}

// verifier for zkat issue
type Verifier struct {
	WellFormedness   *WellFormednessVerifier
	RangeCorrectness common.Verifier
}

// prover for zkat issue
type Prover struct {
	WellFormedness   *WellFormednessProver
	RangeCorrectness common.Prover
}

// serialize
func (p *Proof) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

// deserialize
func (p *Proof) Deserialize(bytes []byte) error {
	return json.Unmarshal(bytes, p)
}

func NewProver(tw []*token.TokenDataWitness, tokens []*math.G1, anonymous bool, pp *crypto.PublicParams) *Prover {
	c := math.Curves[pp.Curve]
	p := &Prover{}
	p.WellFormedness = NewWellFormednessProver(tw, tokens, anonymous, pp.ZKATPedParams, c)

	p.RangeCorrectness = rp.NewProver(tw, tokens, pp.RangeProofParams.SignedValues, pp.RangeProofParams.Exponent, pp.ZKATPedParams, pp.RangeProofParams.SignPK, pp.P, pp.RangeProofParams.Q, math.Curves[pp.Curve])

	return p
}

func NewVerifier(tokens []*math.G1, anonymous bool, pp *crypto.PublicParams) *Verifier {
	v := &Verifier{}
	v.WellFormedness = NewWellFormednessVerifier(tokens, anonymous, pp.ZKATPedParams, math.Curves[pp.Curve])
	v.RangeCorrectness = rp.NewVerifier(tokens, uint64(len(pp.RangeProofParams.SignedValues)), pp.RangeProofParams.Exponent, pp.ZKATPedParams, pp.RangeProofParams.SignPK, pp.P, pp.RangeProofParams.Q, math.Curves[pp.Curve])
	return v
}

func (p *Prover) Prove() ([]byte, error) {
	// well-formedness proof
	wf, err := p.WellFormedness.Prove()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate issue proof")
	}

	// range proof
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

func (v *Verifier) Verify(proof []byte) error {
	ip := &Proof{}
	err := ip.Deserialize(proof)
	if err != nil {
		return err
	}
	// verify well-formedness proof
	err = v.WellFormedness.Verify(ip.WellFormedness)
	if err != nil {
		return err
	}

	// verify range proof
	return v.RangeCorrectness.Verify(ip.RangeCorrectness)
}
