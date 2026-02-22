/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmarking

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

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

type ProofData struct {
	PubParams *v1.PublicParams
	ActionRaw []byte
}

type TokenTxVerifyMetadata struct {
	NumOutputTokens int    `json:"num_outputs"`
	BitLength       uint64 `json:"bit_length,omitempty"`
	TokenType       string `json:"token_type,omitempty"`
	CurveID         int    `json:"curve_id,omitempty"`
}

func (t *TokenTxVerifyMetadata) applyDefaults() *TokenTxVerifyMetadata {
	if t.NumOutputTokens <= 0 {
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

type TokenTxVerifyView struct {
	meta      TokenTxVerifyMetadata
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

	return nil, issue.NewVerifier(coms, q.pubParams).Verify(action.GetProof())
}

// GenerateProofData creates a ZK issue proof andpublic parameters.
func GenerateProofData(meta *TokenTxVerifyMetadata) (*ProofData, error) {
	meta.applyDefaults()

	pubParams, err := v1.Setup(meta.BitLength, nil, math.CurveID(meta.CurveID))
	if err != nil {
		return nil, fmt.Errorf("failed to set up public parameters: %w", err)
	}

	outputValues := make([]uint64, meta.NumOutputTokens)
	outputOwners := make([][]byte, meta.NumOutputTokens)
	var val uint64
	for i := range outputValues {
		val += 10
		outputValues[i] = val
		outputOwners[i] = []byte("alice_" + strconv.Itoa(i))
	}

	issuer := issue.NewIssuer(token2.Type(meta.TokenType), &mock.SigningIdentity{}, pubParams)
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

type TokenTxVerifyViewFactory struct {
	mu     sync.Mutex
	proofs []*ProofData
}

func (c *TokenTxVerifyViewFactory) SetupProofs(n int, meta *TokenTxVerifyMetadata) {
	if meta == nil {
		meta = &TokenTxVerifyMetadata{}
	}
	meta.applyDefaults()
	proofs := make([]*ProofData, n)
	for i := range proofs {
		proof, err := GenerateProofData(meta)
		if err != nil {
			panic(err)
		}
		proofs[i] = proof
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.proofs = proofs
}

// AddProofs enqueues pre-computed proofs for subsequent NewView calls.
func (c *TokenTxVerifyViewFactory) AddProofs(proofs ...*ProofData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.proofs = append(c.proofs, proofs...)
}

func (c *TokenTxVerifyViewFactory) popProof() *ProofData {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.proofs) == 0 {
		return nil
	}
	p := c.proofs[len(c.proofs)-1]
	c.proofs = c.proofs[:len(c.proofs)-1]

	return p
}

// NewView builds a verification view. If proofs were added via AddProofs,
// one is consumed and the expensive generation step is skipped.
// Otherwise a fresh proof is generated from the JSON-encoded metadata.
func (c *TokenTxVerifyViewFactory) NewView(in []byte) (view.View, error) {
	f := &TokenTxVerifyView{}

	if err := json.Unmarshal(in, &f.meta); err != nil {
		return nil, err
	}
	f.meta.applyDefaults()

	proof := c.popProof()
	if proof == nil {
		var err error
		proof, err = GenerateProofData(&f.meta)
		if err != nil {
			return nil, err
		}
	}

	f.pubParams = proof.PubParams
	f.actionRaw = proof.ActionRaw

	return f, nil
}
