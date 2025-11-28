/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fsc_test

import (
	"encoding/json"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	mock2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/endorsement/fsc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/endorsement/fsc/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep/tokenapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockNewRequestApprovalResponderView struct {
	view      *fsc.RequestApprovalResponderView
	ctx       *mock.Context
	fabricTx  *mock.FabricTransaction
	tmsIDRaw  []byte
	validator *mock2.Validator
	rws       *mock.FabricRWSet
	es        *mock.EndorserService
}

func mockNewRequestApprovalResponderView(t *testing.T, overrideTMSID *token.TMSID) *MockNewRequestApprovalResponderView {
	t.Helper()

	ctx := &mock.Context{}
	ctx.ContextReturns(t.Context())
	es := &mock.EndorserService{}
	fabricTx := &mock.FabricTransaction{}
	fabricTx.IDReturns("a_tx_id")
	tmsID := token.TMSID{
		Network:   "a_network",
		Channel:   "a_channel",
		Namespace: "a_namespace",
	}
	if overrideTMSID != nil {
		tmsID = *overrideTMSID
	}
	tmsIDRaw, err := json.Marshal(tmsID)
	require.NoError(t, err)
	fabricTx.TransientReturns(map[string][]byte{
		fsc.TransientTMSIDKey:        tmsIDRaw,
		fsc.TransientTokenRequestKey: []byte("a_token_request"),
	})
	rws := &mock.FabricRWSet{}
	fabricTx.GetRWSetReturns(rws, nil)
	fabricTx.ChaincodeReturns("a_namespace")
	fabricTx.ChaincodeVersionReturns(fsc.ChaincodeVersion)
	fabricTx.FunctionReturns(fsc.InvokeFunction)

	es.ReceiveTxReturns(&endorser.Transaction{
		Provider:    ctx,
		Transaction: fabric.NewTransaction(nil, fabricTx),
	}, nil)
	tmsp := &mock.TokenManagementSystemProvider{}
	tms, validator := tokenapi.NewMockedManagementServiceWithValidation(t, tmsID)
	validator.VerifyTokenRequestFromRawReturns(nil, nil, nil)
	tmsp.GetManagementServiceReturns(tms, nil)

	storage := &mock.Storage{}
	storageProvider := &mock.StorageProvider{}
	storageProvider.GetStorageReturns(storage, nil)
	storage.AppendValidationRecordReturns(nil)

	view := fsc.NewRequestApprovalResponderView(
		nil,
		func(txID string, namespace string, rws *fabric.RWSet) (fsc.Translator, error) {
			return &mock.Translator{}, nil
		},
		es,
		tmsp,
		storageProvider,
	)
	return &MockNewRequestApprovalResponderView{
		view:      view,
		ctx:       ctx,
		fabricTx:  fabricTx,
		tmsIDRaw:  tmsIDRaw,
		validator: validator,
		rws:       rws,
		es:        es,
	}
}

func TestRequestApprovalResponderView(t *testing.T) {
	type testCase struct {
		name             string
		setup            func() *MockNewRequestApprovalResponderView
		verify           func(m *MockNewRequestApprovalResponderView, res any)
		expectError      bool
		expectErrorType  error
		expectErrContain string
	}

	testCases := []testCase{
		{
			name: "failed to receive proposal",
			setup: func() *MockNewRequestApprovalResponderView {
				m := mockNewRequestApprovalResponderView(t, nil)
				m.es.ReceiveTxReturns(nil, errors.New("pineapple"))
				return m
			},
			expectError:      true,
			expectErrorType:  fsc.ErrReceivedProposal,
			expectErrContain: "pineapple",
			verify: func(m *MockNewRequestApprovalResponderView, res any) {
				assert.Equal(t, 0, m.rws.DoneCallCount())
			},
		},
		{
			name: "invalid number of transient field",
			setup: func() *MockNewRequestApprovalResponderView {
				m := mockNewRequestApprovalResponderView(t, nil)
				m.fabricTx.TransientReturns(map[string][]byte{
					"transient": []byte("transient"),
				})
				return m
			},
			expectError:      true,
			expectErrorType:  fsc.ErrReceivedProposal,
			expectErrContain: "invalid number of transient field, expected 2, got 1",
			verify: func(m *MockNewRequestApprovalResponderView, res any) {
				assert.Equal(t, 0, m.rws.DoneCallCount())
			},
		},
		{
			name: "missing TransientTMSIDKey",
			setup: func() *MockNewRequestApprovalResponderView {
				m := mockNewRequestApprovalResponderView(t, nil)
				m.fabricTx.TransientReturns(map[string][]byte{
					"transient":  []byte("transient"),
					"transient2": []byte("transient2"),
				})
				return m
			},
			expectError:      true,
			expectErrorType:  fsc.ErrReceivedProposal,
			expectErrContain: "transient map key [tmsID] does not exists",
			verify: func(m *MockNewRequestApprovalResponderView, res any) {
				assert.Equal(t, 0, m.rws.DoneCallCount())
			},
		},
		{
			name: "tmsid network empty",
			setup: func() *MockNewRequestApprovalResponderView {
				m := mockNewRequestApprovalResponderView(t, &token.TMSID{})
				return m
			},
			expectError:      true,
			expectErrorType:  fsc.ErrReceivedProposal,
			expectErrContain: "invalid tms id [,,]",
			verify: func(m *MockNewRequestApprovalResponderView, res any) {
				assert.Equal(t, 0, m.rws.DoneCallCount())
			},
		},
		{
			name: "tmsid channel empty",
			setup: func() *MockNewRequestApprovalResponderView {
				m := mockNewRequestApprovalResponderView(t, &token.TMSID{
					Network: "a_network",
				})
				return m
			},
			expectError:      true,
			expectErrorType:  fsc.ErrReceivedProposal,
			expectErrContain: "invalid tms id [a_network,,]",
			verify: func(m *MockNewRequestApprovalResponderView, res any) {
				assert.Equal(t, 0, m.rws.DoneCallCount())
			},
		},
		{
			name: "tmsid namespace empty",
			setup: func() *MockNewRequestApprovalResponderView {
				m := mockNewRequestApprovalResponderView(t, &token.TMSID{
					Network: "a_network",
					Channel: "a_channel",
				})
				return m
			},
			expectError:      true,
			expectErrorType:  fsc.ErrReceivedProposal,
			expectErrContain: "invalid tms id [a_network,a_channel,]",
			verify: func(m *MockNewRequestApprovalResponderView, res any) {
				assert.Equal(t, 0, m.rws.DoneCallCount())
			},
		},
		{
			name: "empty token request",
			setup: func() *MockNewRequestApprovalResponderView {
				m := mockNewRequestApprovalResponderView(t, nil)
				m.fabricTx.TransientReturns(map[string][]byte{
					fsc.TransientTMSIDKey: m.tmsIDRaw,
					"transient2":          []byte("transient2"),
				})
				return m
			},
			expectError:      true,
			expectErrorType:  fsc.ErrReceivedProposal,
			expectErrContain: "empty token request",
			verify: func(m *MockNewRequestApprovalResponderView, res any) {
				assert.Equal(t, 0, m.rws.DoneCallCount())
			},
		},
		{
			name: "a namespace is already there",
			setup: func() *MockNewRequestApprovalResponderView {
				m := mockNewRequestApprovalResponderView(t, nil)
				m.rws.NamespacesReturns([]driver.Namespace{
					"a_namespace",
				})
				return m
			},
			expectError:      true,
			expectErrorType:  fsc.ErrInvalidProposal,
			expectErrContain: "non empty namespaces",
			verify: func(m *MockNewRequestApprovalResponderView, res any) {
				assert.Equal(t, 1, m.rws.DoneCallCount())
			},
		},
		{
			name: "invalid function name",
			setup: func() *MockNewRequestApprovalResponderView {
				m := mockNewRequestApprovalResponderView(t, nil)
				m.fabricTx.FunctionReturns("strawberry")
				return m
			},
			expectError:      true,
			expectErrorType:  fsc.ErrInvalidProposal,
			expectErrContain: "invalid function [strawberry]",
			verify: func(m *MockNewRequestApprovalResponderView, res any) {
				assert.Equal(t, 1, m.rws.DoneCallCount())
			},
		},
		{
			name: "validator returns an error",
			setup: func() *MockNewRequestApprovalResponderView {
				m := mockNewRequestApprovalResponderView(t, nil)
				m.validator.VerifyTokenRequestFromRawReturns(nil, nil, errors.New("pineapple"))
				return m
			},
			expectError:      true,
			expectErrorType:  fsc.ErrValidateProposal,
			expectErrContain: "pineapple",
			verify: func(m *MockNewRequestApprovalResponderView, res any) {
				assert.Equal(t, 1, m.rws.DoneCallCount())
			},
		},
		{
			name: "success",
			setup: func() *MockNewRequestApprovalResponderView {
				m := mockNewRequestApprovalResponderView(t, nil)
				return m
			},
			expectError: false,
			verify: func(m *MockNewRequestApprovalResponderView, res any) {
				assert.Equal(t, 1, m.rws.DoneCallCount())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := tc.setup()
			res, err := m.view.Call(m.ctx)
			if tc.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.expectErrorType)
				assert.Contains(t, err.Error(), tc.expectErrContain)
			} else {
				require.NoError(t, err)
			}
			if tc.verify != nil {
				tc.verify(m, res)
			}
		})
	}
}
