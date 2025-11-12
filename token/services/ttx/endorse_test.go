/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx_test

import (
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

type TestEndorseViewContext struct {
	ctx             *mock2.Context
	tx              *ttx.Transaction
	options         []ttx.TxOption
	storageProvider *mock2.StorageProvider
	storage         *mock2.Storage
}

func newTestEndorseViewContext(t *testing.T) *TestEndorseViewContext {
	t.Helper()
	session := &mock2.Session{}
	ch := make(chan *view.Message, 2)
	session.ReceiveReturns(ch)
	ctx := &mock2.Context{}
	ctx.SessionReturns(session)
	ctx.ContextReturns(t.Context())
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
	req := token.NewRequest(nil, "an_anchor")
	req.Metadata.Issues = []*driver.IssueMetadata{
		{
			Issuer: driver.AuditableIdentity{
				Identity: []byte("an_issuer"),
			},
		},
	}
	tms.NewRequestReturns(req, nil)
	tokenAPITMS := tokenapi.NewMockedManagementService(t, tmsID)
	tms.SetTokenManagementServiceStub = func(arg1 *token.Request) error {
		arg1.SetTokenService(tokenAPITMS)
		return nil
	}
	tmsp := &mock2.TokenManagementServiceProvider{}
	tmsp.TokenManagementServiceReturns(tms, nil)
	ctx.GetServiceReturnsOnCall(0, tmsp, nil)

	network := &mock2.Network{}
	network.ComputeTxIDReturns("an_anchor")
	np := &mock2.NetworkProvider{}
	np.GetNetworkReturns(network, nil)
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
	nis.SignReturns([]byte("a_signature"), nil)
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
		Signer: []byte("an_issuer"),
	}
	signatureRequestRaw, err := signatureRequest.Bytes()
	require.NoError(t, err)
	ch <- &view.Message{
		Payload: signatureRequestRaw,
	}
	// then the transaction
	ch <- &view.Message{
		Payload: txRaw,
	}

	ctx.RunViewStub = func(v view.View, option ...view.RunViewOption) (interface{}, error) {
		return v.Call(ctx)
	}

	c := &TestEndorseViewContext{
		ctx:             ctx,
		tx:              tx,
		storage:         storage,
		storageProvider: storageProvider,
	}
	return c
}

func TestEndorseView(t *testing.T) {
	testCases := []struct {
		name          string
		prepare       func() (view.Context, *ttx.Transaction, []ttx.TxOption)
		expectError   bool
		errorContains string
		expectErr     error
		verify        func(tx *ttx.Transaction)
	}{
		{
			name: "transaction is nil",
			prepare: func() (view.Context, *ttx.Transaction, []ttx.TxOption) {
				c := newTestEndorseViewContext(t)
				return c.ctx, nil, c.options
			},
			expectError:   true,
			errorContains: "transaction is nil",
			expectErr:     ttx.ErrInvalidInput,
		},
		{
			name: "success",
			prepare: func() (view.Context, *ttx.Transaction, []ttx.TxOption) {
				c := newTestEndorseViewContext(t)
				return c.ctx, c.tx, c.options
			},
			expectError: false,
		},
		{
			name: "failed storage append",
			prepare: func() (view.Context, *ttx.Transaction, []ttx.TxOption) {
				c := newTestEndorseViewContext(t)
				c.storage.AppendReturns(errors.Errorf("pineapple"))
				return c.ctx, c.tx, c.options
			},
			expectError:   true,
			errorContains: "pineapple",
			expectErr:     ttx.ErrStorage,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, tx, _ := tc.prepare()
			v := ttx.NewEndorseView(tx)
			txBoxed, err := v.Call(ctx)
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
					tc.verify(txBoxed.(*ttx.Transaction))
				}
			}
		})
	}
}
