/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bench

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

const (
	defaultTestRoot = "../../regression/testdata/32-BLS12_381_BBS_GURVY/transfers_i2_o2"
)

type trandferServiceParams struct {
	OutputPath string         `json:"test_root_path,omitempty"`
	CurveID    string         `json:"curve_id,omitempty"`
	NumInputs  int            `json:"num_inputs"`
	NumOutputs int            `json:"num_outputs"`
	Proof      *WireProofData `json:"proof,omitempty"`
}

func NewTokenTransferVerifyParamsSlice(TestRootPath string) []*trandferServiceParams {
	if TestRootPath == "" {
		TestRootPath = defaultTestRoot
	}
	paramsTxt := filepath.Join(filepath.Dir(TestRootPath), "params.txt")

	paramsRaw, err := os.ReadFile(paramsTxt)
	if err != nil {
		panic(err)
	}

	ppRaw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(paramsRaw)))
	if err != nil {
		panic(fmt.Errorf("failed to base64-decode params.txt: %w", err))
	}

	dirName := filepath.Base(filepath.Dir(TestRootPath))
	var curveID string
	if parts := strings.SplitN(dirName, "-", 2); len(parts) == 2 {
		curveID = parts[1]
	}

	var numInputs, numOutputs int
	subDir := filepath.Base(TestRootPath)
	if m := regexp.MustCompile(`_i(\d+)_o(\d+)$`).FindStringSubmatch(subDir); len(m) == 3 {
		numInputs, _ = strconv.Atoi(m[1])
		numOutputs, _ = strconv.Atoi(m[2])
	}

	outPaths, err := os.ReadDir(TestRootPath)
	if err != nil {
		panic(err)
	}
	ret := make([]*trandferServiceParams, len(outPaths))
	for i, outPath := range outPaths {
		ret[i] = newTokenTransferVerifyParams(filepath.Join(TestRootPath, outPath.Name()), curveID, numInputs, numOutputs, ppRaw)
	}

	return ret
}

func newTokenTransferVerifyParams(outputPath string,
	curveID string,
	numInputs int,
	numOutputs int,
	ppRaw []byte,
) *trandferServiceParams {
	outputRaw, err := os.ReadFile(outputPath)
	if err != nil {
		panic(fmt.Errorf("failed to read %s: %w", outputPath, err))
	}

	var tokenData struct {
		ReqRaw []byte `json:"req_raw"`
		TXID   string `json:"txid"`
	}
	if err := json.Unmarshal(outputRaw, &tokenData); err != nil {
		panic(fmt.Errorf("failed to unmarshal output file: %w", err))
	}

	return &trandferServiceParams{
		OutputPath: outputPath,
		CurveID:    curveID,
		NumInputs:  numInputs,
		NumOutputs: numOutputs,
		Proof: &WireProofData{
			PubParamsRaw:    ppRaw,
			TokenRequestRaw: tokenData.ReqRaw,
			TxID:            tokenData.TXID,
		},
	}
}

type TransferServiceView struct {
	params trandferServiceParams
	proof  *ProofData
}

// Call deserializes a token request, extracts its transfer actions,
// and verifies each ZK proof.
// To reflect token/core/zkatdlog/nogh/v1/transfer.go VerifyTransfer
func (q *TransferServiceView) Call(viewCtx view.Context) (interface{}, error) {
	if q.proof == nil {
		return nil, errors.New("proof data is nil")
	}

	tr := &driver.TokenRequest{}
	if err := tr.FromBytes(q.proof.TokenRequestRaw); err != nil {
		return nil, fmt.Errorf("failed to deserialize token request: %w", err)
	}

	pp := q.proof.PubParams
	for _, raw := range tr.Transfers {
		action := &transfer.Action{}
		if err := action.Deserialize(raw); err != nil {
			return nil, fmt.Errorf("failed to deserialize transfer action: %w", err)
		}
		if err := action.Validate(); err != nil {
			return nil, fmt.Errorf("invalid transfer action: %w", err)
		}

		inputTokens := action.InputTokens()
		in := make([]*math.G1, len(inputTokens))
		for i, tok := range inputTokens {
			in[i] = tok.Data
		}

		if err := transfer.NewVerifier(in, action.GetOutputCommitments(), pp).Verify(action.GetProof()); err != nil {
			return nil, fmt.Errorf("failed to verify transfer proof: %w", err)
		}
	}

	return nil, nil
}

type TransferServiceViewFactory struct{}

// NewView builds a verification view.
// Wire proof embedded in the JSON params (remote/gRPC path)
func (c *TransferServiceViewFactory) NewView(in []byte) (view.View, error) {
	f := &TransferServiceView{}

	if err := json.Unmarshal(in, &f.params); err != nil {
		return nil, err
	}
	var proof *ProofData
	if f.params.Proof != nil {
		var err error
		proof, err = f.params.Proof.Deserialize()
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal wire proof: %w", err)
		}
	}
	f.proof = proof

	return f, nil
}
