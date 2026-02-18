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
	defaultNumInputs  = 1
	deafultNumOutputs = 1
	defaultBitLength  = 32
	defaultTokenType  = "benchmark-token"
	defaultCurveID    = math.BLS12_381_BBS_GURVY
)

type TokenTxValidateParams struct {
	NumInputs  int    `json:"num_inputs"`
	NumOutputs int    `json:"num_outputs"`
	BitLength  uint64 `json:"bit_length,omitempty"`
	TokenType  string `json:"token_type,omitempty"`
	CurveID    int    `json:"curve_id,omitempty"`
}

func (t *TokenTxValidateParams) applyDefaults() {
	if t.NumInputs <= 0 {
		t.NumInputs = defaultNumInputs
	}
	if t.NumOutputs <= 0 {
		t.NumOutputs = deafultNumOutputs
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
}

type TokenTxValidateView struct {
	params    TokenTxValidateParams
	pubParams *v1.PublicParams
	actionRaw []byte
}

// Call verifies a pre-computed ZK issue proof by deserializing the
// issue action and checking the proof against the token commitments.
//
// This benchmarks the ZKP verification path used by the
// fabric-token-sdk validator for zkatdlog issue actions.
func (q *TokenTxValidateView) Call(viewCtx view.Context) (interface{}, error) {
	action := &issue.Action{}
	if err := action.Deserialize(q.actionRaw); err != nil {
		return nil, fmt.Errorf("failed to deserialize issue action: %w", err)
	}

	coms := make([]*math.G1, len(action.Outputs))
	for i := range action.Outputs {
		coms[i] = action.Outputs[i].Data
	}

	return nil, issue.NewVerifier(coms, q.pubParams).Verify(action.GetProof())
}

type TokenTxValidateViewFactory struct{}

func (c *TokenTxValidateViewFactory) NewView(in []byte) (view.View, error) {
	f := &TokenTxValidateView{}

	if err := json.Unmarshal(in, &f.params); err != nil {
		return nil, err
	}

	f.params.applyDefaults()

	var err error
	f.pubParams, err = v1.Setup(f.params.BitLength, nil, math.CurveID(f.params.CurveID))
	if err != nil {
		return nil, fmt.Errorf("failed to set up public parameters: %w", err)
	}

	// Generate issue action following the same approach as
	// createIssuerProofVerificationEnv in issuer_test.go.
	outputValues := make([]uint64, f.params.NumOutputs)
	outputOwners := make([][]byte, f.params.NumOutputs)
	var val uint64
	for i := range outputValues {
		val += 10
		outputValues[i] = val
		outputOwners[i] = []byte("alice_" + strconv.Itoa(i))
	}

	issuer := issue.NewIssuer(token2.Type(f.params.TokenType), &mock.SigningIdentity{}, f.pubParams)
	action, _, err := issuer.GenerateZKIssue(outputValues, outputOwners)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ZK issue: %w", err)
	}

	f.actionRaw, err = action.Serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize issue action: %w", err)
	}

	return f, nil
}
