/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"fmt"
	"strconv"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue/mock"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
)

const (
	benchBitLength = 32
	benchCurveID   = math.BLS12_381_BBS_GURVY
	benchTokenType = "benchmark-token"
)

type TokenTxValidateParams struct {
	NumInputs  int `json:"num_inputs"`
	NumOutputs int `json:"num_outputs"`
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

	if f.params.NumOutputs <= 0 {
		f.params.NumOutputs = 1
	}

	var err error
	f.pubParams, err = v1.Setup(benchBitLength, nil, benchCurveID)
	if err != nil {
		return nil, fmt.Errorf("failed to set up public parameters: %w", err)
	}

	// Generate issue action following the same approach as
	// createIssuerProofVerificationEnv in issuer_test.go.
	outputValues := make([]uint64, f.params.NumOutputs)
	outputOwners := make([][]byte, f.params.NumOutputs)
	for i := range outputValues {
		outputValues[i] = uint64(i*10 + 10)
		outputOwners[i] = []byte("alice_" + strconv.Itoa(i))
	}

	issuer := issue.NewIssuer(benchTokenType, &mock.SigningIdentity{}, f.pubParams)
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
