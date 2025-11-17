/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fsc_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/endorsement/fsc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/endorsement/fsc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRequestApprovalResponderView(t *testing.T) {
	type testCase struct {
		name             string
		setup            func() (*fsc.RequestApprovalResponderView, view.Context)
		expectError      bool
		expectErrorType  error
		expectErrContain string
	}

	testCases := []testCase{
		{
			name: "failed to receive proposal",
			setup: func() (*fsc.RequestApprovalResponderView, view.Context) {
				ctx := &mock.Context{}
				es := &mock.EndorserService{}
				es.ReceiveTxReturns(nil, errors.New("pineapple"))
				view := fsc.NewRequestApprovalResponderView(nil, nil, es)
				return view, ctx
			},
			expectError:      true,
			expectErrorType:  fsc.ErrReceivedProposal,
			expectErrContain: "pineapple",
		},
		{
			name: "failed to receive proposal",
			setup: func() (*fsc.RequestApprovalResponderView, view.Context) {
				ctx := &mock.Context{}
				es := &mock.EndorserService{}
				es.ReceiveTxReturns(&endorser.Transaction{
					Provider:    ctx,
					Transaction: fabric.NewTransaction(nil, nil),
				}, nil)
				view := fsc.NewRequestApprovalResponderView(nil, nil, es)
				return view, ctx
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			view, ctx := tc.setup()
			_, err := view.Call(ctx)
			if tc.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.expectErrorType)
				assert.Contains(t, err.Error(), tc.expectErrContain)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
