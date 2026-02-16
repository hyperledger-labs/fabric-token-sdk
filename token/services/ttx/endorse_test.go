/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx_test

import (
	"context"
	"math/rand"
	"strconv"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/endpoint"
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

type TestEndorseViewContextInput struct {
	IssuerIdentity    token.Identity
	AnotherReceivedTx bool
}

type TestEndorseViewContext struct {
	ctx                     *mock2.Context
	tx                      *ttx.Transaction
	options                 []ttx.TxOption
	storageProvider         *mock2.StorageProvider
	storage                 *mock2.Storage
	session                 *mock2.Session
	tokenSigner             *mock.Signer
	networkIdentitySigner   *mock2.NetworkIdentitySigner
	ch                      chan *view.Message
	tokenIP                 *mock.IdentityProvider
	networkIdentityProvider *mock2.NetworkIdentityProvider
}

func newTransaction(t *testing.T) *ttx.Transaction {
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

	return tx
}

func newTestEndorseViewContext(t *testing.T, input *TestEndorseViewContextInput) *TestEndorseViewContext {
	t.Helper()
	if input == nil {
		input = &TestEndorseViewContextInput{}
	}
	if len(input.IssuerIdentity) == 0 {
		input.IssuerIdentity = []byte("an_issuer")
	}

	session := &mock2.Session{}
	ch := make(chan *view.Message, 2)
	session.ReceiveReturns(ch)

	tms := &mock2.TokenManagementServiceWithExtensions{}
	tms.NetworkReturns("a_network")
	tms.ChannelReturns("a_channel")
	tmsID := token.TMSID{
		Network:   "a_network",
		Channel:   "a_channel",
		Namespace: "a_namespace",
	}
	tms.IDReturns(tmsID)
	tokenDes := &mock.Deserializer{}
	tokenIP := &mock.IdentityProvider{}
	tokenIP.IsMeReturns(true)
	tokenSigner := &mock.Signer{}
	tokenSigner.SignReturns([]byte("a_token_sigma"), nil)
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
	network.ComputeTxIDReturns("an_anchor")
	np := &mock2.NetworkProvider{}
	np.GetNetworkReturns(network, nil)

	req := token.NewRequest(nil, "an_anchor")
	req.Metadata.Issues = []*driver.IssueMetadata{
		{
			Issuer: driver.AuditableIdentity{
				Identity: []byte("an_issuer"),
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
	nis.SignReturns([]byte("an_ack_signature"), nil)
	networkIdentityProvider.GetSignerReturns(nis, nil)

	ctx = &mock2.Context{}
	ctx.SessionReturns(session)
	ctx.ContextReturns(t.Context())
	ctx.GetServiceReturnsOnCall(0, storageProvider, nil)
	ctx.GetServiceReturnsOnCall(1, np, nil)
	ctx.GetServiceReturnsOnCall(2, tmsp, nil)
	ctx.GetServiceReturnsOnCall(3, networkIdentityProvider, nil)
	ctx.GetServiceReturnsOnCall(4, storageProvider, nil)

	txRaw, err := tx.Bytes()
	require.NoError(t, err)

	// first the signature request
	signatureRequest := &ttx.SignatureRequest{
		Signer: input.IssuerIdentity,
	}
	signatureRequestRaw, err := signatureRequest.Bytes()
	require.NoError(t, err)
	ch <- &view.Message{
		Payload: signatureRequestRaw,
	}
	// then the transaction
	if input.AnotherReceivedTx {
		tx2 := newTransaction(t)
		txRaw2, err := tx2.Bytes()
		require.NoError(t, err)
		ch <- &view.Message{
			Payload: txRaw2,
		}
	} else {
		ch <- &view.Message{
			Payload: txRaw,
		}
	}

	ctx.RunViewStub = func(v view.View, option ...view.RunViewOption) (interface{}, error) {
		return v.Call(ctx)
	}

	c := &TestEndorseViewContext{
		ctx:                     ctx,
		tx:                      tx,
		storage:                 storage,
		storageProvider:         storageProvider,
		session:                 session,
		tokenSigner:             tokenSigner,
		networkIdentitySigner:   nis,
		ch:                      ch,
		tokenIP:                 tokenIP,
		networkIdentityProvider: networkIdentityProvider,
	}

	return c
}

func TestEndorseView(t *testing.T) {
	testCases := []struct {
		name          string
		prepare       func() *TestEndorseViewContext
		expectError   bool
		errorContains string
		expectErr     error
		verify        func(*TestEndorseViewContext, any)
	}{
		{
			name: "transaction is nil",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, nil)
				c.tx = nil
				c.options = nil

				return c
			},
			expectError:   true,
			errorContains: "transaction is nil",
			expectErr:     ttx.ErrInvalidInput,
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 0, ctx.session.SendWithContextCallCount())
			},
		},
		{
			name: "success",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, nil)

				return c
			},
			expectError: false,
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 2, ctx.session.SendWithContextCallCount())
				_, msg := ctx.session.SendWithContextArgsForCall(0)
				assert.Equal(t, []byte("a_token_sigma"), msg)
				_, msg = ctx.session.SendWithContextArgsForCall(1)
				assert.Equal(t, []byte("an_ack_signature"), msg)
			},
		},
		{
			name: "failed storage append",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, nil)
				c.storage.AppendReturns(errors.Errorf("pineapple"))

				return c
			},
			expectError:   true,
			errorContains: "pineapple",
			expectErr:     ttx.ErrStorage,
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 0, ctx.session.SendWithContextCallCount())
			},
		},
		{
			name: "wrong identity in signature request",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, &TestEndorseViewContextInput{
					IssuerIdentity: []byte("another_issuer"),
				})

				return c
			},
			expectError:   true,
			errorContains: "signature request's signer does not match the expected signer",
			expectErr:     ttx.ErrSignerIdentityMismatch,
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 0, ctx.session.SendWithContextCallCount())
			},
		},
		{
			name: "received transaction is different",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, &TestEndorseViewContextInput{
					IssuerIdentity: []byte("another_issuer"),
				})

				return c
			},
			expectError:   true,
			errorContains: "signature request's signer does not match the expected signer",
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 1, ctx.session.SendWithContextCallCount())
			},
		},
		{
			name: "failed unmarshalling signature request",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, nil)
				for len(c.ch) > 0 {
					<-c.ch
				}
				c.ch <- &view.Message{Payload: []byte("garbage")}

				return c
			},
			expectError:   true,
			errorContains: "failed unmarshalling signature request",
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 0, ctx.session.SendWithContextCallCount())
			},
		},
		{
			name: "failed signing request",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, nil)
				c.tokenSigner.SignReturns(nil, errors.New("sign error"))

				return c
			},
			expectError:   true,
			errorContains: "failed signing request",
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 0, ctx.session.SendWithContextCallCount())
			},
		},
		{
			name: "failed sending signature back",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, nil)
				c.session.SendWithContextReturns(errors.New("send error"))

				return c
			},
			expectError:   true,
			errorContains: "failed sending signature back",
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 1, ctx.session.SendWithContextCallCount())
			},
		},
		{
			name: "failed receiving transaction",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, nil)
				sigReq := <-c.ch
				<-c.ch
				c.ch <- sigReq
				c.ch <- &view.Message{Payload: []byte("garbage transaction")}

				return c
			},
			expectError:   true,
			errorContains: "failed receiving transaction",
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 1, ctx.session.SendWithContextCallCount())
			},
		},
		{
			name: "failed acknowledging transaction (signing)",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, nil)
				c.networkIdentitySigner.SignReturns(nil, errors.New("ack sign error"))

				return c
			},
			expectError:   true,
			errorContains: "failed to sign ack response",
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 1, ctx.session.SendWithContextCallCount())
			},
		},
		{
			name: "failed acknowledging transaction (sending)",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, nil)
				count := 0
				c.session.SendWithContextStub = func(ctx context.Context, payload []byte) error {
					count++
					if count == 2 {
						return errors.New("ack send error")
					}

					return nil
				}

				return c
			},
			expectError:   true,
			errorContains: "failed sending ack",
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 2, ctx.session.SendWithContextCallCount())
			},
		},
		{
			name: "failed getting storage provider",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, nil)
				c.ctx.GetServiceReturnsOnCall(4, nil, errors.New("storage provider error"))

				return c
			},
			expectError:   true,
			expectErr:     ttx.ErrDepNotAvailableInContext,
			errorContains: "storage provider",
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 2, ctx.session.SendWithContextCallCount())
			},
		},
		{
			name: "failed getting identity provider",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, nil)
				c.ctx.GetServiceReturnsOnCall(3, nil, errors.New("identity provider error"))

				return c
			},
			expectError:   true,
			expectErr:     ttx.ErrDepNotAvailableInContext,
			errorContains: "identity provider",
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 1, ctx.session.SendWithContextCallCount())
			},
		},
		{
			name: "failed getting signer",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, nil)
				c.tokenIP.GetSignerReturns(nil, errors.New("signer error"))

				return c
			},
			expectError:   true,
			errorContains: "cannot find signer for",
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 0, ctx.session.SendWithContextCallCount())
			},
		},
		{
			name: "success with FromSignatureRequest",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, nil)
				c.tx.FromSignatureRequest = &ttx.SignatureRequest{
					Signer: []byte("an_issuer"),
				}
				// Clear channel and put only transaction
				for len(c.ch) > 0 {
					<-c.ch
				}
				txRaw, _ := c.tx.Bytes()
				c.ch <- &view.Message{Payload: txRaw}

				return c
			},
			expectError: false,
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 2, ctx.session.SendWithContextCallCount())
			},
		},
		{
			name: "failed cache request",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, nil)
				c.storageProvider.CacheRequestReturns(errors.New("cache error"))

				return c
			},
			expectError: false,
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 2, ctx.session.SendWithContextCallCount())
				assert.Equal(t, 1, ctx.storageProvider.CacheRequestCallCount())
			},
		},
		{
			name: "failed getting signer for ack",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, nil)
				c.networkIdentityProvider.GetSignerReturns(nil, errors.New("ack signer error"))

				return c
			},
			expectError:   true,
			errorContains: "failed to get signer for default identity",
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 1, ctx.session.SendWithContextCallCount())
			},
		},
		{
			name: "multiple signers",
			prepare: func() *TestEndorseViewContext {
				c := newTestEndorseViewContext(t, nil)
				c.tx.TokenRequest.Metadata.Issues = append(c.tx.TokenRequest.Metadata.Issues, &driver.IssueMetadata{
					Issuer: driver.AuditableIdentity{
						Identity: []byte("another_issuer"),
					},
				})

				req1 := <-c.ch
				tx := <-c.ch

				c.ch = make(chan *view.Message, 3)
				c.session.ReceiveReturns(c.ch)

				c.ch <- req1

				sigReq := &ttx.SignatureRequest{
					Signer: []byte("another_issuer"),
				}
				sigReqRaw, _ := sigReq.Bytes()
				c.ch <- &view.Message{Payload: sigReqRaw}

				c.ch <- tx

				return c
			},
			expectError: false,
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 3, ctx.session.SendWithContextCallCount())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testCtx := tc.prepare()
			v := ttx.NewEndorseView(testCtx.tx)
			txBoxed, err := v.Call(testCtx.ctx)
			if tc.expectError {
				require.Error(t, err)
				if len(tc.errorContains) != 0 {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				if tc.expectErr != nil {
					require.ErrorIs(t, err, tc.expectErr)
				}
			} else {
				require.NoError(t, err)
				if tc.verify != nil {
					tc.verify(testCtx, txBoxed)
				}
			}
		})
	}
}
