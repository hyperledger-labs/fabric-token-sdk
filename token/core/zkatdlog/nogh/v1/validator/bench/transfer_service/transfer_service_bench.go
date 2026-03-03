/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bench

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	fabtoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/driver"
	dlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/driver"
	v1setup "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	tk "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	defaultTestRoot = "../../regression/testdata/32-BLS12_381_BBS_GURVY/transfers_i2_o2"
)

var (
	once            sync.Once
	cachedValidator *token.Validator
)

type trandferServiceParams struct {
	OutputPath string     `json:"test_root_path,omitempty"`
	TokenData  *TokenData `json:"proof,omitempty"`
}

func (p *trandferServiceParams) PublicParamsRaw() ([]byte, error) {
	paramsTxt := filepath.Join(filepath.Dir(filepath.Dir(p.OutputPath)), "params.txt")
	raw, err := os.ReadFile(paramsTxt)
	if err != nil {
		return nil, fmt.Errorf("failed to read params file %s: %w", paramsTxt, err)
	}
	ppRaw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("failed to base64-decode params file: %w", err)
	}
	return ppRaw, nil
}

func (p *trandferServiceParams) PublicParams() (*v1setup.PublicParams, error) {
	ppRaw, err := p.PublicParamsRaw()
	if err != nil {
		return nil, err
	}
	return v1setup.NewPublicParamsFromBytes(ppRaw, v1setup.DLogNoGHDriverName, v1setup.ProtocolV1)
}

func (p *trandferServiceParams) NumInputs() int {
	subDir := filepath.Base(filepath.Dir(p.OutputPath))
	if m := regexp.MustCompile(`_i(\d+)_o(\d+)$`).FindStringSubmatch(subDir); len(m) == 3 {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

func (p *trandferServiceParams) NumOutputs() int {
	subDir := filepath.Base(filepath.Dir(p.OutputPath))
	if m := regexp.MustCompile(`_i(\d+)_o(\d+)$`).FindStringSubmatch(subDir); len(m) == 3 {
		n, _ := strconv.Atoi(m[2])
		return n
	}
	return 0
}

func (p *trandferServiceParams) CurveID() string {
	dirName := filepath.Base(filepath.Dir(filepath.Dir(p.OutputPath)))
	if parts := strings.SplitN(dirName, "-", 2); len(parts) == 2 {
		return parts[1]
	}
	return ""
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

	outPaths, err := os.ReadDir(TestRootPath)
	if err != nil {
		panic(err)
	}
	ret := make([]*trandferServiceParams, len(outPaths))
	for i, outPath := range outPaths {
		ret[i] = newTokenTransferVerifyParams(filepath.Join(TestRootPath, outPath.Name()), ppRaw)
	}

	return ret
}

func newTokenTransferVerifyParams(outputPath string, ppRaw []byte) *trandferServiceParams {
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
		TokenData: &TokenData{
			TokenRequestRaw: tokenData.ReqRaw,
			TxID:            tokenData.TXID,
		},
	}
}

type fakeLedger struct{}

func (*fakeLedger) GetState(_ tk.ID) ([]byte, error) {
	panic("not implemented")
}

func newTokenValidator(ppRaw []byte) (*token.Validator, error) {
	is := core.NewPPManagerFactoryService(fabtoken.NewPPMFactory(), dlog.NewPPMFactory())
	ppm, err := is.PublicParametersFromBytes(ppRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize public parameters: %w", err)
	}
	v, err := is.DefaultValidator(ppm)
	if err != nil {
		return nil, fmt.Errorf("failed to create default validator: %w", err)
	}

	return token.NewValidator(v), nil
}

type TransferServiceView struct {
	params    trandferServiceParams
	tokenData *TokenData
	validator *token.Validator
}

// Call runs the full token request validation pipeline matching the
// regression test's UnmarshallAndVerifyWithMetadata path: auditing,
// signatures, ZK proofs, HTLC, upgrade witnesses, and metadata checks.
// Call Chain:
//   1. token.Validator.UnmarshallAndVerifyWithMetadata -> driver.Validator.VerifyTokenRequestFromRaw
//   2. VerifyTokenRequestFromRaw (from token/core/common/validator.go)
//     - deserializes the raw bytes into a TokenRequest, prepares signed message + signatures
// 	   - calls VerifyTokenRequest
//   3. VerifyTokenRequest runs three stages:
//     - Auditing validation (VerifyAuditing) [verifies auditor signatures]
//     - Issue validation (verifyIssues) [verifies issue actions]
//     - Transfer validation (verifyTransfers):
//       a. TransferActionValidate [action.Validate()]
//       b. TransferSignatureValidate [verifies sender signatures (deserializes owner identity, checks signature)]
//       c. TransferUpgradeWitnessValidate
//       d. TransferZKProofValidate [transfer.NewVerifier(in, outputCommitments, pp).Verify(proof)]
//       e. TransferHTLCValidate
//       f. TransferApplicationDataValidate [validates metadata]
//   4. After all validators pass, it checks that all metadata have been validated

func (q *TransferServiceView) Call(viewCtx view.Context) (interface{}, error) {
	if q.tokenData == nil {
		return nil, errors.New("proof data is nil")
	}

	_, _, err := q.validator.UnmarshallAndVerifyWithMetadata(
		context.Background(),
		&fakeLedger{},
		token.RequestAnchor(q.tokenData.TxID),
		q.tokenData.TokenRequestRaw,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token request: %w", err)
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
	if f.params.TokenData != nil {

		f.tokenData = f.params.TokenData

		var initErr error
		once.Do(func() {
			ppRaw, err := f.params.PublicParamsRaw()
			if err != nil {
				initErr = err
			} else {
				cachedValidator, initErr = newTokenValidator(ppRaw)
			}
		})
		if initErr != nil {
			return nil, fmt.Errorf("failed to create token validator: %w", initErr)
		}
		f.validator = cachedValidator
	}

	return f, nil
}
