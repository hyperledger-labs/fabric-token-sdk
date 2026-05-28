/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx_test

import (
	"encoding/json"
	"reflect"
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
	ctx             *mock2.Context
	tx              *ttx.Transaction
	options         []ttx.TxOption
	storageProvider *mock2.StorageProvider
	storage         *mock2.Storage
	session         *mock2.Session
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

	storage := &mock2.Storage{}
	storage.AppendReturns(nil)
	storageProvider := &mock2.StorageProvider{}
	storageProvider.GetStorageReturns(storage, nil)

	networkIdentityProvider := &mock2.NetworkIdentityProvider{}
	nis := &mock2.NetworkIdentitySigner{}
	nis.SignReturns([]byte("an_ack_signature"), nil)
	networkIdentityProvider.GetSignerReturns(nis, nil)

	// Resolve services by their requested type rather than by call order, so the
	// stub is robust to additional GetService lookups (e.g. envelope metrics).
	getService := func(v any) (any, error) {
		rt, ok := v.(reflect.Type)
		if !ok {
			return nil, errors.Errorf("unexpected service request [%T]", v)
		}
		switch rt.String() {
		case "*dep.TokenManagementServiceProvider":
			return tmsp, nil
		case "*dep.NetworkProvider":
			return np, nil
		case "*dep.NetworkIdentityProvider":
			return networkIdentityProvider, nil
		case "*endpoint.Service":
			return &endpoint.Service{}, nil
		case "*ttx.StorageProvider":
			return storageProvider, nil
		case "*session.EnvelopeMetrics":
			return nil, errors.New("envelope metrics not registered in test")
		default:
			return nil, errors.Errorf("unexpected service request [%s]", rt.String())
		}
	}

	ctx := &mock2.Context{}
	ctx.SessionReturns(session)
	ctx.ContextReturns(t.Context())
	ctx.GetServiceStub = getService
	tx, err := ttx.NewTransaction(ctx, []byte("a_signer"))
	require.NoError(t, err)

	ctx = &mock2.Context{}
	ctx.SessionReturns(session)
	ctx.ContextReturns(t.Context())
	ctx.GetServiceStub = getService

	txRaw, err := tx.Bytes()
	require.NoError(t, err)

	// first the signature request
	signatureRequest := &ttx.SignatureRequest{
		Signer: input.IssuerIdentity,
	}
	ch <- &view.Message{
		Payload: mustEnvelopeBytes(t, ttx.TypeSignatureRequest, signatureRequest),
	}
	// then the transaction
	ch <- &view.Message{
		Payload: mustEnvelopeBytes(t, ttx.TypeTransaction, &ttx.TransactionPayload{Raw: txRaw}),
	}

	ctx.RunViewStub = func(v view.View, option ...view.RunViewOption) (any, error) {
		return v.Call(ctx)
	}

	c := &TestEndorseViewContext{
		ctx:             ctx,
		tx:              tx,
		storage:         storage,
		storageProvider: storageProvider,
		session:         session,
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
				var signaturePayload ttx.SignaturePayload
				require.NoError(t, json.Unmarshal(mustUnwrapBody(t, msg, ttx.TypeSignature), &signaturePayload))
				assert.Equal(t, []byte("a_token_sigma"), signaturePayload.Signature)
				_, msg = ctx.session.SendWithContextArgsForCall(1)
				require.NoError(t, json.Unmarshal(mustUnwrapBody(t, msg, ttx.TypeSignature), &signaturePayload))
				assert.Equal(t, []byte("an_ack_signature"), signaturePayload.Signature)
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
			expectErr:     ttx.ErrSignerIdentityMismatch,
			verify: func(ctx *TestEndorseViewContext, _ any) {
				assert.Equal(t, 0, ctx.session.SendWithContextCallCount())
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
