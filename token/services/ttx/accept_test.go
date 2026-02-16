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
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/endpoint"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	mock2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep/tokenapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCounter struct {
	addCallCount int
}

func (m *mockCounter) With(labelValues ...string) metrics.Counter {
	return m
}

func (m *mockCounter) Add(delta float64) {
	m.addCallCount++
}

type TestAcceptViewContext struct {
	ctx                     *mock2.Context
	tx                      *ttx.Transaction
	storageProvider         *mock2.StorageProvider
	storage                 *mock2.Storage
	session                 *mock2.Session
	networkIdentityProvider *mock2.NetworkIdentityProvider
	networkIdentitySigner   *mock2.NetworkIdentitySigner
	metrics                 *ttx.Metrics
	acceptedCounter         *mockCounter
}

func newTestAcceptViewContext(t *testing.T) *TestAcceptViewContext {
	t.Helper()

	session := &mock2.Session{}
	ch := make(chan *view.Message, 2)
	session.ReceiveReturns(ch)

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
	tokenDes := &mock.Deserializer{}
	tokenIP := &mock.IdentityProvider{}
	tokenIP.IsMeReturns(true)
	tokenSigner := &mock.Signer{}
	tokenSigner.SignReturns([]byte("a_token_sigma"+seed), nil)
	tokenIP.GetSignerReturns(tokenSigner, nil)
	tms.SigServiceReturns(token.NewSignatureService(tokenDes, tokenIP))
	tokenAPITMS := tokenapi.NewMockedManagementService(t, tmsID)
	tms.SetTokenManagementServiceStub = func(arg1 *token.Request) error {
		arg1.SetTokenService(tokenAPITMS)

		return nil
	}
	tmsp := &mock2.TokenManagementServiceProvider{}
	tmsp.TokenManagementServiceReturns(tms, nil)
	network := &mock2.Network{}
	network.ComputeTxIDReturns("an_anchor" + seed)
	np := &mock2.NetworkProvider{}
	np.GetNetworkReturns(network, nil)

	req := token.NewRequest(nil, token.RequestAnchor("an_anchor"+seed))
	req.Metadata.Issues = []*driver.IssueMetadata{
		{
			Issuer: driver.AuditableIdentity{
				Identity: []byte("an_issuer" + seed),
			},
		},
	}
	tms.NewRequestReturns(req, nil)

	ctx := &mock2.Context{}
	ctx.SessionReturns(session)
	ctx.ContextReturns(t.Context())
	ctx.GetServiceReturnsOnCall(0, tmsp, nil)
	ctx.GetServiceReturnsOnCall(1, np, nil)
	ctx.GetServiceReturnsOnCall(2, &endpoint.Service{}, nil)
	ctx.GetServiceReturnsOnCall(3, np, nil)
	ctx.GetServiceReturnsOnCall(4, tmsp, nil)
	tx, err := ttx.NewTransaction(ctx, []byte("a_signer"))
	require.NoError(t, err)

	storage := &mock2.Storage{}
	storage.AppendReturns(nil)
	storageProvider := &mock2.StorageProvider{}
	storageProvider.GetStorageReturns(storage, nil)

	networkIdentityProvider := &mock2.NetworkIdentityProvider{}
	nis := &mock2.NetworkIdentitySigner{}
	nis.SignReturns([]byte("an_ack_signature"+seed), nil)
	networkIdentityProvider.GetSignerReturns(nis, nil)
	networkIdentityProvider.DefaultIdentityReturns([]byte("default_identity" + seed))

	acceptedCounter := &mockCounter{}
	metrics := &ttx.Metrics{
		AcceptedTransactions: acceptedCounter,
	}

	ctx = &mock2.Context{}
	ctx.SessionReturns(session)
	ctx.ContextReturns(t.Context())
	ctx.GetServiceReturnsOnCall(0, storageProvider, nil)
	ctx.GetServiceReturnsOnCall(1, networkIdentityProvider, nil)
	ctx.GetServiceReturnsOnCall(2, metrics, nil)
	ctx.GetServiceReturnsOnCall(3, storageProvider, nil)

	return &TestAcceptViewContext{
		ctx:                     ctx,
		tx:                      tx,
		storage:                 storage,
		storageProvider:         storageProvider,
		session:                 session,
		networkIdentityProvider: networkIdentityProvider,
		networkIdentitySigner:   nis,
		metrics:                 metrics,
		acceptedCounter:         acceptedCounter,
	}
}

func TestAcceptView(t *testing.T) {
	testCases := []struct {
		name          string
		prepare       func(*TestAcceptViewContext)
		expectError   bool
		errorContains string
		expectErr     error
		verify        func(*TestAcceptViewContext, any)
	}{
		{
			name: "transaction is nil",
			prepare: func(c *TestAcceptViewContext) {
				c.tx = nil
			},
			expectError:   true,
			errorContains: "transaction is nil",
			expectErr:     ttx.ErrInvalidInput,
		},
		{
			name: "success",
			prepare: func(c *TestAcceptViewContext) {
			},
			expectError: false,
			verify: func(c *TestAcceptViewContext, res any) {
				assert.Equal(t, 1, c.session.SendWithContextCallCount())
				_, sigma := c.session.SendWithContextArgsForCall(0)
				assert.Contains(t, string(sigma), "an_ack_signature")

				assert.Equal(t, 1, c.storage.AppendCallCount())
				assert.Equal(t, 1, c.storageProvider.CacheRequestCallCount())
				assert.Equal(t, 1, c.acceptedCounter.addCallCount)
			},
		},
		{
			name: "failed storing transaction records",
			prepare: func(c *TestAcceptViewContext) {
				c.storage.AppendReturns(errors.New("storage error"))
			},
			expectError:   true,
			errorContains: "failed storing transaction records",
		},
		{
			name: "failed acknowledgement (identity provider)",
			prepare: func(c *TestAcceptViewContext) {
				c.ctx.GetServiceReturnsOnCall(1, nil, errors.New("id provider error"))
			},
			expectError:   true,
			errorContains: "failed to get identity provider",
		},
		{
			name: "failed acknowledgement (signer)",
			prepare: func(c *TestAcceptViewContext) {
				c.networkIdentityProvider.GetSignerReturns(nil, errors.New("signer error"))
			},
			expectError:   true,
			errorContains: "failed to get signer for default identity",
		},
		{
			name: "failed acknowledgement (sign)",
			prepare: func(c *TestAcceptViewContext) {
				c.networkIdentitySigner.SignReturns(nil, errors.New("sign error"))
			},
			expectError:   true,
			errorContains: "failed to sign ack response",
		},
		{
			name: "failed acknowledgement (send)",
			prepare: func(c *TestAcceptViewContext) {
				c.session.SendWithContextReturns(errors.New("send error"))
			},
			expectError:   true,
			errorContains: "failed sending ack",
		},
		{
			name: "failed caching request (storage provider)",
			prepare: func(c *TestAcceptViewContext) {
				c.ctx.GetServiceReturnsOnCall(3, nil, errors.New("storage provider error"))
			},
			expectError:   true,
			errorContains: "failed to get storage provider",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestAcceptViewContext(t)
			tc.prepare(c)
			v := ttx.NewAcceptView(c.tx)
			res, err := v.Call(c.ctx)
			if tc.expectError {
				require.Error(t, err)
				if len(tc.errorContains) > 0 {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				if tc.expectErr != nil {
					require.ErrorIs(t, err, tc.expectErr)
				}
			} else {
				require.NoError(t, err)
				if tc.verify != nil {
					tc.verify(c, res)
				}
			}
		})
	}
}
