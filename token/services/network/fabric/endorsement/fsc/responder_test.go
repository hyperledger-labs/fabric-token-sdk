/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fsc_test

import (
	"encoding/json"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
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
			name: "invalid number of transient field",
			setup: func() (*fsc.RequestApprovalResponderView, view.Context) {
				ctx := &mock.Context{}
				ctx.ContextReturns(t.Context())
				es := &mock.EndorserService{}
				fabricTx := &mock.FabricTransaction{}
				fabricTx.IDReturns("a_tx_id")
				fabricTx.TransientReturns(map[string][]byte{
					"transient": []byte("transient"),
				})
				es.ReceiveTxReturns(&endorser.Transaction{
					Provider:    ctx,
					Transaction: fabric.NewTransaction(nil, fabricTx),
				}, nil)
				view := fsc.NewRequestApprovalResponderView(nil, nil, es)
				return view, ctx
			},
			expectError:      true,
			expectErrorType:  fsc.ErrReceivedProposal,
			expectErrContain: "invalid number of transient field, expected 2, got 1",
		},
		{
			name: "missing TransientTMSIDKey",
			setup: func() (*fsc.RequestApprovalResponderView, view.Context) {
				ctx := &mock.Context{}
				ctx.ContextReturns(t.Context())
				es := &mock.EndorserService{}
				fabricTx := &mock.FabricTransaction{}
				fabricTx.IDReturns("a_tx_id")
				fabricTx.TransientReturns(map[string][]byte{
					"transient":  []byte("transient"),
					"transient2": []byte("transient2"),
				})
				es.ReceiveTxReturns(&endorser.Transaction{
					Provider:    ctx,
					Transaction: fabric.NewTransaction(nil, fabricTx),
				}, nil)
				view := fsc.NewRequestApprovalResponderView(nil, nil, es)
				return view, ctx
			},
			expectError:      true,
			expectErrorType:  fsc.ErrReceivedProposal,
			expectErrContain: "transient map key [tmsID] does not exists",
		},
		{
			name: "tmsid network empty",
			setup: func() (*fsc.RequestApprovalResponderView, view.Context) {
				ctx := &mock.Context{}
				ctx.ContextReturns(t.Context())
				es := &mock.EndorserService{}
				fabricTx := &mock.FabricTransaction{}
				fabricTx.IDReturns("a_tx_id")
				tmsID := token.TMSID{}
				tmsIDRaw, err := json.Marshal(tmsID)
				require.NoError(t, err)
				fabricTx.TransientReturns(map[string][]byte{
					fsc.TransientTMSIDKey: tmsIDRaw,
					"transient2":          []byte("transient2"),
				})
				es.ReceiveTxReturns(&endorser.Transaction{
					Provider:    ctx,
					Transaction: fabric.NewTransaction(nil, fabricTx),
				}, nil)
				view := fsc.NewRequestApprovalResponderView(nil, nil, es)
				return view, ctx
			},
			expectError:      true,
			expectErrorType:  fsc.ErrReceivedProposal,
			expectErrContain: "invalid tms id [,,]",
		},
		{
			name: "tmsid channel empty",
			setup: func() (*fsc.RequestApprovalResponderView, view.Context) {
				ctx := &mock.Context{}
				ctx.ContextReturns(t.Context())
				es := &mock.EndorserService{}
				fabricTx := &mock.FabricTransaction{}
				fabricTx.IDReturns("a_tx_id")
				tmsID := token.TMSID{
					Network: "a_network",
				}
				tmsIDRaw, err := json.Marshal(tmsID)
				require.NoError(t, err)
				fabricTx.TransientReturns(map[string][]byte{
					fsc.TransientTMSIDKey: tmsIDRaw,
					"transient2":          []byte("transient2"),
				})
				es.ReceiveTxReturns(&endorser.Transaction{
					Provider:    ctx,
					Transaction: fabric.NewTransaction(nil, fabricTx),
				}, nil)
				view := fsc.NewRequestApprovalResponderView(nil, nil, es)
				return view, ctx
			},
			expectError:      true,
			expectErrorType:  fsc.ErrReceivedProposal,
			expectErrContain: "invalid tms id [a_network,,]",
		},
		{
			name: "tmsid namespace empty",
			setup: func() (*fsc.RequestApprovalResponderView, view.Context) {
				ctx := &mock.Context{}
				ctx.ContextReturns(t.Context())
				es := &mock.EndorserService{}
				fabricTx := &mock.FabricTransaction{}
				fabricTx.IDReturns("a_tx_id")
				tmsID := token.TMSID{
					Network: "a_network",
					Channel: "a_channel",
				}
				tmsIDRaw, err := json.Marshal(tmsID)
				require.NoError(t, err)
				fabricTx.TransientReturns(map[string][]byte{
					fsc.TransientTMSIDKey: tmsIDRaw,
					"transient2":          []byte("transient2"),
				})
				es.ReceiveTxReturns(&endorser.Transaction{
					Provider:    ctx,
					Transaction: fabric.NewTransaction(nil, fabricTx),
				}, nil)
				view := fsc.NewRequestApprovalResponderView(nil, nil, es)
				return view, ctx
			},
			expectError:      true,
			expectErrorType:  fsc.ErrReceivedProposal,
			expectErrContain: "invalid tms id [a_network,a_channel,]",
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
