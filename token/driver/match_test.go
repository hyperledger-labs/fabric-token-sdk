/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueMetadataMatch(t *testing.T) {
	makeMetadata := func() *driver.IssueMetadata {
		return &driver.IssueMetadata{
			Issuer:       driver.AuditableIdentity{Identity: driver.Identity("issuer")},
			Inputs:       []*driver.IssueInputMetadata{{}, {}},
			Outputs:      []*driver.IssueOutputMetadata{{}, {}},
			ExtraSigners: []driver.Identity{driver.Identity("signer1")},
		}
	}
	makeAction := func() *mock.IssueAction {
		ia := &mock.IssueAction{}
		ia.NumInputsReturns(2)
		ia.NumOutputsReturns(2)
		ia.ExtraSignersReturns([]driver.Identity{driver.Identity("signer1")})
		ia.GetIssuerReturns([]byte("issuer"))

		return ia
	}

	tests := []struct {
		name      string
		setupMeta func(*driver.IssueMetadata)
		setupAct  func(*mock.IssueAction)
		wantErr   string
	}{
		{
			name: "success",
		},
		{
			name:     "input count mismatch",
			setupAct: func(a *mock.IssueAction) { a.NumInputsReturns(3) },
			wantErr:  "expected [2] inputs but got [3]",
		},
		{
			name:     "output count mismatch",
			setupAct: func(a *mock.IssueAction) { a.NumOutputsReturns(1) },
			wantErr:  "expected [2] outputs but got [1]",
		},
		{
			name: "extra signer count mismatch",
			setupAct: func(a *mock.IssueAction) {
				a.ExtraSignersReturns([]driver.Identity{driver.Identity("s1"), driver.Identity("s2")})
			},
			wantErr: "expected [2] extra signers but got [1]",
		},
		{
			name: "extra signer value mismatch",
			setupAct: func(a *mock.IssueAction) {
				a.ExtraSignersReturns([]driver.Identity{driver.Identity("other")})
			},
			wantErr: "expected extra signer",
		},
		{
			name:     "issuer mismatch",
			setupAct: func(a *mock.IssueAction) { a.GetIssuerReturns([]byte("other")) },
			wantErr:  "expected issuer",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			meta := makeMetadata()
			action := makeAction()
			if tc.setupMeta != nil {
				tc.setupMeta(meta)
			}
			if tc.setupAct != nil {
				tc.setupAct(action)
			}
			err := meta.Match(action)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}

func TestTransferMetadataMatch(t *testing.T) {
	makeMetadata := func() *driver.TransferMetadata {
		return &driver.TransferMetadata{
			Inputs:       []*driver.TransferInputMetadata{{}, {}},
			Outputs:      []*driver.TransferOutputMetadata{{}, {}},
			ExtraSigners: []driver.Identity{driver.Identity("signer1")},
			Issuer:       driver.Identity("issuer"),
		}
	}
	makeAction := func() *mock.TransferAction {
		ta := &mock.TransferAction{}
		ta.NumInputsReturns(2)
		ta.NumOutputsReturns(2)
		ta.ExtraSignersReturns([]driver.Identity{driver.Identity("signer1")})
		ta.GetIssuerReturns(driver.Identity("issuer"))

		return ta
	}

	tests := []struct {
		name      string
		setupMeta func(*driver.TransferMetadata)
		setupAct  func(*mock.TransferAction)
		wantErr   string
	}{
		{
			name: "success",
		},
		{
			name:     "input count mismatch",
			setupAct: func(a *mock.TransferAction) { a.NumInputsReturns(3) },
			wantErr:  "expected [2] inputs but got [3]",
		},
		{
			name:     "output count mismatch",
			setupAct: func(a *mock.TransferAction) { a.NumOutputsReturns(1) },
			wantErr:  "expected [2] outputs but got [1]",
		},
		{
			name: "extra signer count mismatch",
			setupAct: func(a *mock.TransferAction) {
				a.ExtraSignersReturns([]driver.Identity{driver.Identity("s1"), driver.Identity("s2")})
			},
			wantErr: "expected [1] extra signers but got [2]",
		},
		{
			name: "extra signer value mismatch",
			setupAct: func(a *mock.TransferAction) {
				a.ExtraSignersReturns([]driver.Identity{driver.Identity("other")})
			},
			wantErr: "expected extra signer",
		},
		{
			name:     "issuer mismatch",
			setupAct: func(a *mock.TransferAction) { a.GetIssuerReturns(driver.Identity("other")) },
			wantErr:  "expected issuer",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			meta := makeMetadata()
			action := makeAction()
			if tc.setupMeta != nil {
				tc.setupMeta(meta)
			}
			if tc.setupAct != nil {
				tc.setupAct(action)
			}
			err := meta.Match(action)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}

func TestTransferMetadataMatchInputs(t *testing.T) {
	tests := []struct {
		name         string
		actionInputs [][]byte
		ledgerTokens [][]byte
		wantErr      string
	}{
		{
			name:         "success",
			actionInputs: [][]byte{[]byte("token0"), []byte("token1")},
			ledgerTokens: [][]byte{[]byte("token0"), []byte("token1")},
		},
		{
			name:         "length mismatch",
			actionInputs: [][]byte{[]byte("token0")},
			ledgerTokens: [][]byte{[]byte("token0"), []byte("token1")},
			wantErr:      "action has [1] inputs but [2] tokens provided",
		},
		{
			name:         "nil action input skipped",
			actionInputs: [][]byte{nil, []byte("token1")},
			ledgerTokens: [][]byte{[]byte("anything"), []byte("token1")},
		},
		{
			name:         "byte mismatch at index",
			actionInputs: [][]byte{[]byte("token0"), []byte("token1")},
			ledgerTokens: [][]byte{[]byte("token0"), []byte("different")},
			wantErr:      "input token at index [1]: does not match the transfer action",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			meta := &driver.TransferMetadata{}
			err := meta.MatchInputs(tc.actionInputs, tc.ledgerTokens)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}

func TestTransferOutputMetadataValidateReceivers(t *testing.T) {
	tests := []struct {
		name      string
		receivers []*driver.AuditableIdentity
		wantErr   string
	}{
		{
			name:      "success single receiver",
			receivers: []*driver.AuditableIdentity{{Identity: driver.Identity("bob")}},
		},
		{
			name: "success multiple receivers",
			receivers: []*driver.AuditableIdentity{
				{Identity: driver.Identity("bob")},
				{Identity: driver.Identity("carol")},
			},
		},
		{
			name:    "no receivers empty slice",
			wantErr: "has no receivers",
		},
		{
			name:      "nil receivers slice",
			receivers: nil,
			wantErr:   "has no receivers",
		},
		{
			name: "nil receiver at index",
			receivers: []*driver.AuditableIdentity{
				{Identity: driver.Identity("bob")},
				nil,
			},
			wantErr: "receiver at index [1] is nil",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output := &driver.TransferOutputMetadata{Receivers: tc.receivers}
			err := output.ValidateReceivers()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}
