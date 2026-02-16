/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"fmt"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	zktoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
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

type tokenEntry struct {
	pub *math.G1
	msg []byte
}

type TokenTxValidateView struct {
	params    TokenTxValidateParams
	inputs    []tokenEntry
	outputs   []tokenEntry
	pubParams *v1.PublicParams
	proof     []byte
}

// generateEntry creates a tokenEntry from a Pedersen commitment and a label.
func generateEntry(commitment *math.G1, msg []byte) tokenEntry {
	return tokenEntry{
		pub: commitment,
		msg: msg,
	}
}

// Call verifies the zkatdlog zero-knowledge transfer proof that demonstrates:
//   - inputs and outputs encode the same token type
//   - the sum of input values equals the sum of output values
//   - all output values are within the authorized range (bulletproof)
//
// This benchmarks the ZKP verification path used by the
// fabric-token-sdk validator for zkatdlog transfer actions.
func (q *TokenTxValidateView) Call(viewCtx view.Context) (interface{}, error) {
	in := make([]*math.G1, len(q.inputs))
	for i := range q.inputs {
		in[i] = q.inputs[i].pub
	}
	out := make([]*math.G1, len(q.outputs))
	for i := range q.outputs {
		out[i] = q.outputs[i].pub
	}

	// fmt.Println("============")
	// fmt.Println("Pre-computed params")
	// fmt.Printf("%s\n", q.pubParams.String())
	// fmt.Println("============")

	return nil, transfer.NewVerifier(in, out, q.pubParams).Verify(q.proof)
}

type TokenTxValidateViewFactory struct{}

func (c *TokenTxValidateViewFactory) NewView(in []byte) (view.View, error) {
	f := &TokenTxValidateView{}
	if err := json.Unmarshal(in, &f.params); err != nil {
		return nil, err
	}

	// Default to 1 input / 1 output when not specified.
	if f.params.NumInputs <= 0 {
		f.params.NumInputs = 1
	}
	if f.params.NumOutputs <= 0 {
		f.params.NumOutputs = 1
	}

	// Set up public parameters (Pedersen generators + range proof params).
	var err error
	f.pubParams, err = v1.Setup(benchBitLength, nil, benchCurveID)
	if err != nil {
		return nil, fmt.Errorf("failed to set up public parameters: %w", err)
	}
	curve := math.Curves[f.pubParams.Curve]

	// Build inputs and compute their total.
	inValues := make([]uint64, f.params.NumInputs)

	var totalIn uint64
	for i := range inValues {
		value := 500 + uint64(i)*10
		inValues[i] = value
		totalIn += value
	}

	// Split total evenly across outputs.
	outValues := make([]uint64, f.params.NumOutputs)
	perOutput := totalIn / uint64(len(outValues))
	remainder := totalIn % uint64(len(outValues))

	for i := range outValues {
		outValues[i] = perOutput
	}

	// Add leftover to first output (if any).
	outValues[0] += remainder

	// Generate input token commitments (Pedersen) and their witnesses.
	inCommitments, inMeta, err := zktoken.GetTokensWithWitness(
		inValues, benchTokenType, f.pubParams.PedersenGenerators, curve,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate input tokens: %w", err)
	}

	// Generate output token commitments and witnesses.
	outCommitments, outMeta, err := zktoken.GetTokensWithWitness(
		outValues, benchTokenType, f.pubParams.PedersenGenerators, curve,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate output tokens: %w", err)
	}

	// Generate the ZK transfer proof:
	//   - TypeAndSum proof: inputs/outputs have same type and equal total value
	//   - Range proof (bulletproof): output values lie in [0, 2^bitLength)
	prover, err := transfer.NewProver(
		inMeta,
		outMeta,
		inCommitments,
		outCommitments,
		f.pubParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create ZK prover: %w", err)
	}

	f.proof, err = prover.Prove()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ZK transfer proof: %w", err)
	}

	// Store token commitments as entries.
	f.inputs = make([]tokenEntry, f.params.NumInputs)
	for i := range f.params.NumInputs {
		msg := []byte(fmt.Sprintf("token-tx-input:%d:transfer:%d:alice:bob", i, inValues[i]))
		f.inputs[i] = generateEntry(inCommitments[i], msg)
	}
	f.outputs = make([]tokenEntry, f.params.NumOutputs)
	for i := range f.params.NumOutputs {
		msg := []byte(fmt.Sprintf("token-tx-output:%d:transfer:%d:alice:bob", i, outValues[i]))
		f.outputs[i] = generateEntry(outCommitments[i], msg)
	}

	return f, nil
}
