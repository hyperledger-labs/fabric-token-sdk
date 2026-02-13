/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx_test

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	mock2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestOrderingViewContext struct {
	ctx             *mock2.Context
	tx              *ttx.Transaction
	networkProvider *mock2.NetworkProvider
	network         *mock2.Network
	storageProvider *mock2.StorageProvider
	storage         *mock2.Storage
}

func newTestOrderingViewContext(t *testing.T) *TestOrderingViewContext {
	t.Helper()

	seed := strconv.Itoa(rand.Int())

	tms := &mock2.TokenManagementServiceWithExtensions{}
	tms.NetworkReturns("a_network" + seed)
	tms.ChannelReturns("a_channel" + seed)
	tmsID := token.TMSID{
		Network:   "a_network" + seed,
		Channel:   "a_channel" + seed,
		Namespace: "a_namespace" + seed,
	}
	tms.IDReturns(tmsID)

	tmsp := &mock2.TokenManagementServiceProvider{}
	tmsp.TokenManagementServiceReturns(tms, nil)

	network := &mock2.Network{}
	network.ComputeTxIDReturns("an_anchor" + seed)
	networkProvider := &mock2.NetworkProvider{}
	networkProvider.GetNetworkReturns(network, nil)

	storage := &mock2.Storage{}
	storageProvider := &mock2.StorageProvider{}
	storageProvider.GetStorageReturns(storage, nil)

	tx := &ttx.Transaction{
		Payload: &ttx.Payload{
			Envelope:     nil,
			TokenRequest: token.NewRequest(nil, token.RequestAnchor("an_anchor"+seed)),
		},
	}

	ctx := &mock2.Context{}
	ctx.ContextReturns(t.Context())
	ctx.GetServiceReturnsOnCall(0, networkProvider, nil)
	ctx.GetServiceReturnsOnCall(1, storageProvider, nil)

	return &TestOrderingViewContext{
		ctx:             ctx,
		tx:              tx,
		networkProvider: networkProvider,
		network:         network,
		storageProvider: storageProvider,
		storage:         storage,
	}
}

func TestOrderingView(t *testing.T) {
	testCases := []struct {
		name          string
		prepare       func(*TestOrderingViewContext)
		options       []ttx.TxOption
		expectError   bool
		errorContains string
	}{
		{
			name: "success",
			prepare: func(c *TestOrderingViewContext) {
			},
			expectError: false,
		},
		{
			name: "success without caching",
			prepare: func(c *TestOrderingViewContext) {
			},
			options:     []ttx.TxOption{ttx.WithNoCachingRequest()},
			expectError: false,
		},
		{
			name: "failed to get network provider",
			prepare: func(c *TestOrderingViewContext) {
				c.ctx.GetServiceReturnsOnCall(0, nil, errors.New("network provider error"))
			},
			expectError:   true,
			errorContains: "network provider error",
		},
		{
			name: "failed to get network",
			prepare: func(c *TestOrderingViewContext) {
				c.networkProvider.GetNetworkReturns(nil, errors.New("network error"))
			},
			expectError:   true,
			errorContains: "failed to get network",
		},
		{
			name: "failed to broadcast",
			prepare: func(c *TestOrderingViewContext) {
				c.network.BroadcastReturns(errors.New("broadcast error"))
			},
			expectError:   true,
			errorContains: "failed to broadcast token transaction",
		},
		{
			name: "failed to get storage provider",
			prepare: func(c *TestOrderingViewContext) {
				c.ctx.GetServiceReturnsOnCall(1, nil, errors.New("storage provider error"))
			},
			expectError:   true,
			errorContains: "failed to get storage provider",
		},
		{
			name: "failed to cache request",
			prepare: func(c *TestOrderingViewContext) {
				c.storageProvider.CacheRequestReturns(errors.New("cache error"))
			},
			expectError: false, // warning only
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestOrderingViewContext(t)
			if tc.prepare != nil {
				tc.prepare(c)
			}
			v := ttx.NewOrderingView(c.tx, tc.options...)
			_, err := v.Call(c.ctx)
			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, 1, c.network.BroadcastCallCount())
				if tc.options == nil {
					assert.Equal(t, 1, c.storageProvider.CacheRequestCallCount())
				} else {
					// check if NoCachingRequest was used
					opts, _ := ttx.CompileOpts(tc.options...)
					if opts.NoCachingRequest {
						assert.Equal(t, 0, c.storageProvider.CacheRequestCallCount())
					} else {
						assert.Equal(t, 1, c.storageProvider.CacheRequestCallCount())
					}
				}
			}
		})
	}
}

func TestOrderingAndFinalityView(t *testing.T) {
	testCases := []struct {
		name          string
		prepare       func(*mock2.Context)
		expectError   bool
		errorContains string
	}{
		{
			name: "success",
			prepare: func(ctx *mock2.Context) {
				ctx.RunViewReturnsOnCall(0, nil, nil)
				ctx.RunViewReturnsOnCall(1, nil, nil)
			},
			expectError: false,
		},
		{
			name: "failed ordering",
			prepare: func(ctx *mock2.Context) {
				ctx.RunViewReturnsOnCall(0, nil, errors.New("ordering error"))
			},
			expectError:   true,
			errorContains: "ordering error",
		},
		{
			name: "failed finality",
			prepare: func(ctx *mock2.Context) {
				ctx.RunViewReturnsOnCall(0, nil, nil)
				ctx.RunViewReturnsOnCall(1, nil, errors.New("finality error"))
			},
			expectError:   true,
			errorContains: "finality error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &mock2.Context{}
			if tc.prepare != nil {
				tc.prepare(ctx)
			}
			tx := &ttx.Transaction{}
			v := ttx.NewOrderingAndFinalityView(tx)
			_, err := v.Call(ctx)
			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, 2, ctx.RunViewCallCount())
			}
		})
	}
}
