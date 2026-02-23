/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmarking

import (
	"encoding/json"
	"fmt"
	"strconv"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue/mock"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	deafultNumOutputs = 2
	defaultBitLength  = 32
	defaultTokenType  = "benchmark-token"
	defaultCurveID    = math.BLS12_381_BBS_GURVY
)

var dummyIdemixPK = []byte("benchmark-dummy-idemix-pk")

type ProofData struct {
	PubParams *v1.PublicParams
	ActionRaw []byte
}

func (p *ProofData) ToWire() (*WireProofData, error) {
	ppRaw, err := p.PubParams.Serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize public parameters: %w", err)
	}

	return &WireProofData{PubParamsRaw: ppRaw, ActionRaw: p.ActionRaw}, nil
}

// WireProofData is the JSON-safe representation of ProofData for transport.
type WireProofData struct {
	PubParamsRaw []byte `json:"pub_params_raw"`
	ActionRaw    []byte `json:"action_raw"`
}

func (w *WireProofData) UnWire() (*ProofData, error) {
	pp, err := v1.NewPublicParamsFromBytes(w.PubParamsRaw, v1.DLogNoGHDriverName, v1.ProtocolV1)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize public parameters: %w", err)
	}

	return &ProofData{PubParams: pp, ActionRaw: w.ActionRaw}, nil
}

type TokenTxVerifyParams struct {
	NumOutputTokens int            `json:"num_outputs"`
	BitLength       uint64         `json:"bit_length,omitempty"`
	TokenType       string         `json:"token_type,omitempty"`
	CurveID         int            `json:"curve_id,omitempty"`
	Proof           *WireProofData `json:"proof,omitempty"`
}

func (t *TokenTxVerifyParams) applyDefaults() *TokenTxVerifyParams {
	if t.NumOutputTokens < 0 {
		t.NumOutputTokens = deafultNumOutputs
	}
	if t.BitLength <= 0 {
		t.BitLength = defaultBitLength
	}
	if t.TokenType == "" {
		t.TokenType = defaultTokenType
	}
	if t.CurveID <= 0 {
		t.CurveID = int(defaultCurveID)
	}

	return t
}

// SetupProof pre-generates a ZK proof and embeds it in the metadata so it
// travels over the wire. Call this on the client before sending the workload.
func (t *TokenTxVerifyParams) SetupProof() error {
	proof, err := GenerateProofData(t)
	if err != nil {
		return err
	}
	t.Proof, err = proof.ToWire()

	return err
}

type TokenTxVerifyView struct {
	params    TokenTxVerifyParams // TODO remove duplicates
	pubParams *v1.PublicParams
	actionRaw []byte
}

// Call verifies a pre-computed ZK proof by deserializing the
// issue action and checking the proof against the token commitments.
func (q *TokenTxVerifyView) Call(viewCtx view.Context) (interface{}, error) {
	action := &issue.Action{}
	if err := action.Deserialize(q.actionRaw); err != nil {
		return nil, fmt.Errorf("failed to deserialize issue action: %w", err)
	}

	coms := make([]*math.G1, len(action.Outputs))
	for i := range action.Outputs {
		coms[i] = action.Outputs[i].Data
	}

	err := issue.NewVerifier(coms, q.pubParams).Verify(action.GetProof())
	if err != nil {
		return nil, fmt.Errorf("failed to Verify Proof %w", err)
	}

	return nil, nil
}

// GenerateProofData creates a ZK issue proof and associated public parameters.
func GenerateProofData(params *TokenTxVerifyParams) (*ProofData, error) {
	params.applyDefaults()

	pubParams, err := v1.Setup(params.BitLength, dummyIdemixPK, math.CurveID(params.CurveID))
	if err != nil {
		return nil, fmt.Errorf("failed to set up public parameters: %w", err)
	}

	outputValues := make([]uint64, params.NumOutputTokens)
	outputOwners := make([][]byte, params.NumOutputTokens)
	var val uint64
	for i := range outputValues {
		val += 10
		outputValues[i] = val
		outputOwners[i] = []byte("alice_" + strconv.Itoa(i))
	}

	issuer := issue.NewIssuer(token2.Type(params.TokenType), &mock.SigningIdentity{}, pubParams)
	action, _, err := issuer.GenerateZKIssue(outputValues, outputOwners)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ZK issue: %w", err)
	}

	actionRaw, err := action.Serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize issue action: %w", err)
	}

	return &ProofData{
		PubParams: pubParams,
		ActionRaw: actionRaw,
	}, nil
}

type TokenTxVerifyViewFactory struct{}

// NewView builds a verification view. Proof source priority:
//  1. Wire proof embedded in the JSON params (remote/gRPC path)
//  2. Fresh generation (fallback)
func (c *TokenTxVerifyViewFactory) NewView(in []byte) (view.View, error) {
	f := &TokenTxVerifyView{}

	if err := json.Unmarshal(in, &f.params); err != nil {
		return nil, err
	}
	f.params.applyDefaults()

	var proof *ProofData
	if f.params.Proof != nil {
		var err error
		proof, err = f.params.Proof.UnWire()
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal wire proof: %w", err)
		}
	} else {
		var err error
		proof, err = GenerateProofData(&f.params)
		if err != nil {
			return nil, err
		}
	}

	f.pubParams = proof.PubParams
	f.actionRaw = proof.ActionRaw

	return f, nil
}
