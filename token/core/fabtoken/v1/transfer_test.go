/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1_test

import (
	"context"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTokenLoader is a mock implementation of the TokenLoader interface
type MockTokenLoader struct {
	GetTokensStub func(ctx context.Context, ids []*token.ID) ([]*token.Token, error)
}

func (m *MockTokenLoader) GetTokens(ctx context.Context, ids []*token.ID) ([]*token.Token, error) {
	return m.GetTokensStub(ctx, ids)
}

type mockPublicParamsManager struct {
	*mock.PublicParamsManager
}

func (m *mockPublicParamsManager) PublicParams() *setup.PublicParams {
	return m.PublicParameters().(*setup.PublicParams)
}

func TestTransferService(t *testing.T) {
	logger := logging.MustGetLogger("test")
	ctx := context.Background()

	t.Run("NewTransferService", func(t *testing.T) {
		ppm := &mockPublicParamsManager{PublicParamsManager: &mock.PublicParamsManager{}}
		ws := &mock.WalletService{}
		tl := &MockTokenLoader{}
		des := &mock.Deserializer{}
		s := v1.NewTransferService(logger, ppm, ws, tl, des)
		assert.NotNil(t, s)
	})

	t.Run("Transfer Success", func(t *testing.T) {
		ppm := &mockPublicParamsManager{PublicParamsManager: &mock.PublicParamsManager{}}
		ws := &mock.WalletService{}
		tl := &MockTokenLoader{}
		des := &mock.Deserializer{}
		s := v1.NewTransferService(logger, ppm, ws, tl, des)

		ids := []*token.ID{{TxId: "tx1", Index: 0}}
		inputTokens := []*token.Token{
			{Owner: []byte("owner1"), Type: "type1", Quantity: "10"},
		}
		tl.GetTokensStub = func(ctx context.Context, ids []*token.ID) ([]*token.Token, error) {
			return inputTokens, nil
		}

		outputs := []*token.Token{
			{Owner: []byte("owner2"), Type: "type1", Quantity: "10"},
		}

		des.GetAuditInfoReturns([]byte("audit1"), nil)
		des.RecipientsReturns([]driver.Identity{[]byte("owner2")}, nil)

		action, metadata, err := s.Transfer(ctx, "", nil, ids, outputs, nil)
		require.NoError(t, err)
		assert.NotNil(t, action)
		assert.NotNil(t, metadata)

		// 1 input, 1 output.
		// GetAuditInfo called for:
		// 1. input owner ("owner1")
		// 2. output owner ("owner2")
		// 3. recipient ("owner2")
		assert.Equal(t, 3, des.GetAuditInfoCallCount())
	})

	t.Run("Transfer Redeem Success", func(t *testing.T) {
		ppm := &mockPublicParamsManager{PublicParamsManager: &mock.PublicParamsManager{}}
		ws := &mock.WalletService{}
		tl := &MockTokenLoader{}
		des := &mock.Deserializer{}
		s := v1.NewTransferService(logger, ppm, ws, tl, des)

		ids := []*token.ID{{TxId: "tx1", Index: 0}}
		inputTokens := []*token.Token{
			{Owner: []byte("owner1"), Type: "type1", Quantity: "10"},
		}
		tl.GetTokensStub = func(ctx context.Context, ids []*token.ID) ([]*token.Token, error) {
			return inputTokens, nil
		}

		outputs := []*token.Token{
			{Owner: nil, Type: "type1", Quantity: "10"},
		}

		des.GetAuditInfoReturns([]byte("audit1"), nil)

		pp := &setup.PublicParams{
			IssuerIDs: []driver.Identity{[]byte("issuer1")},
		}
		ppm.PublicParametersReturns(pp)

		action, metadata, err := s.Transfer(ctx, "", nil, ids, outputs, &driver.TransferOptions{Attributes: make(map[interface{}]interface{})})
		require.NoError(t, err)
		assert.NotNil(t, action)
		assert.NotNil(t, metadata)

		assert.Equal(t, driver.Identity([]byte("issuer1")), action.(interface{ GetIssuer() driver.Identity }).GetIssuer())
	})

	t.Run("Transfer Error TokenLoader", func(t *testing.T) {
		ppm := &mockPublicParamsManager{PublicParamsManager: &mock.PublicParamsManager{}}
		ws := &mock.WalletService{}
		tl := &MockTokenLoader{}
		des := &mock.Deserializer{}
		s := v1.NewTransferService(logger, ppm, ws, tl, des)

		tl.GetTokensStub = func(ctx context.Context, ids []*token.ID) ([]*token.Token, error) {
			return nil, errors.New("loading error")
		}

		_, _, err := s.Transfer(ctx, "", nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load tokens")
	})

	t.Run("Transfer Error GetAuditInfoSender", func(t *testing.T) {
		ppm := &mockPublicParamsManager{PublicParamsManager: &mock.PublicParamsManager{}}
		ws := &mock.WalletService{}
		tl := &MockTokenLoader{}
		des := &mock.Deserializer{}
		s := v1.NewTransferService(logger, ppm, ws, tl, des)

		tl.GetTokensStub = func(ctx context.Context, ids []*token.ID) ([]*token.Token, error) {
			return []*token.Token{{Owner: []byte("owner1"), Type: "type1", Quantity: "10"}}, nil
		}

		des.GetAuditInfoReturns(nil, errors.New("audit error"))

		_, _, err := s.Transfer(ctx, "", nil, []*token.ID{{TxId: "tx1", Index: 0}}, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed getting audit info for sender identity")
	})

	t.Run("Transfer Error GetAuditInfoOutputOwner", func(t *testing.T) {
		ppm := &mockPublicParamsManager{PublicParamsManager: &mock.PublicParamsManager{}}
		ws := &mock.WalletService{}
		tl := &MockTokenLoader{}
		des := &mock.Deserializer{}
		s := v1.NewTransferService(logger, ppm, ws, tl, des)

		tl.GetTokensStub = func(ctx context.Context, ids []*token.ID) ([]*token.Token, error) {
			return []*token.Token{{Owner: []byte("owner1"), Type: "type1", Quantity: "10"}}, nil
		}

		des.GetAuditInfoReturnsOnCall(0, []byte("audit1"), nil)
		des.GetAuditInfoReturnsOnCall(1, nil, errors.New("audit error output"))

		outputs := []*token.Token{
			{Owner: []byte("owner2"), Type: "type1", Quantity: "10"},
		}

		_, _, err := s.Transfer(ctx, "", nil, []*token.ID{{TxId: "tx1", Index: 0}}, outputs, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed getting audit info for sender identity")
	})

	t.Run("Transfer Error Recipients", func(t *testing.T) {
		ppm := &mockPublicParamsManager{PublicParamsManager: &mock.PublicParamsManager{}}
		ws := &mock.WalletService{}
		tl := &MockTokenLoader{}
		des := &mock.Deserializer{}
		s := v1.NewTransferService(logger, ppm, ws, tl, des)

		tl.GetTokensStub = func(ctx context.Context, ids []*token.ID) ([]*token.Token, error) {
			return []*token.Token{{Owner: []byte("owner1"), Type: "type1", Quantity: "10"}}, nil
		}

		des.GetAuditInfoReturnsOnCall(0, []byte("audit1"), nil)
		des.GetAuditInfoReturnsOnCall(1, []byte("audit2"), nil)
		des.RecipientsReturns(nil, errors.New("recipients error"))

		outputs := []*token.Token{
			{Owner: []byte("owner2"), Type: "type1", Quantity: "10"},
		}

		_, _, err := s.Transfer(ctx, "", nil, []*token.ID{{TxId: "tx1", Index: 0}}, outputs, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed getting recipients")
	})

	t.Run("Transfer Error GetAuditInfoRecipient", func(t *testing.T) {
		ppm := &mockPublicParamsManager{PublicParamsManager: &mock.PublicParamsManager{}}
		ws := &mock.WalletService{}
		tl := &MockTokenLoader{}
		des := &mock.Deserializer{}
		s := v1.NewTransferService(logger, ppm, ws, tl, des)

		tl.GetTokensStub = func(ctx context.Context, ids []*token.ID) ([]*token.Token, error) {
			return []*token.Token{{Owner: []byte("owner1"), Type: "type1", Quantity: "10"}}, nil
		}

		des.GetAuditInfoReturnsOnCall(0, []byte("audit1"), nil)
		des.GetAuditInfoReturnsOnCall(1, []byte("audit2"), nil)
		des.RecipientsReturns([]driver.Identity{[]byte("owner2")}, nil)
		des.GetAuditInfoReturnsOnCall(2, nil, errors.New("audit error recipient"))

		outputs := []*token.Token{
			{Owner: []byte("owner2"), Type: "type1", Quantity: "10"},
		}

		_, _, err := s.Transfer(ctx, "", nil, []*token.ID{{TxId: "tx1", Index: 0}}, outputs, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed getting audit info for receiver identity")
	})

	t.Run("Transfer Error SelectIssuerForRedeem", func(t *testing.T) {
		ppm := &mockPublicParamsManager{PublicParamsManager: &mock.PublicParamsManager{}}
		ws := &mock.WalletService{}
		tl := &MockTokenLoader{}
		des := &mock.Deserializer{}
		s := v1.NewTransferService(logger, ppm, ws, tl, des)

		tl.GetTokensStub = func(ctx context.Context, ids []*token.ID) ([]*token.Token, error) {
			return []*token.Token{{Owner: []byte("owner1"), Type: "type1", Quantity: "10"}}, nil
		}

		des.GetAuditInfoReturns([]byte("audit1"), nil)

		pp := &setup.PublicParams{
			IssuerIDs: []driver.Identity{[]byte("issuer1")},
		}
		ppm.PublicParametersReturns(pp)

		outputs := []*token.Token{
			{Owner: nil, Type: "type1", Quantity: "10"},
		}

		opts := &driver.TransferOptions{
			Attributes: map[interface{}]interface{}{
				ttx.IssuerFSCIdentityKey: "invalid identity type",
			},
		}

		_, _, err := s.Transfer(ctx, "", nil, []*token.ID{{TxId: "tx1", Index: 0}}, outputs, opts)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to select issuer for redeem")
	})

	t.Run("VerifyTransfer", func(t *testing.T) {
		ppm := &mockPublicParamsManager{PublicParamsManager: &mock.PublicParamsManager{}}
		ws := &mock.WalletService{}
		tl := &MockTokenLoader{}
		des := &mock.Deserializer{}
		s := v1.NewTransferService(logger, ppm, ws, tl, des)
		err := s.VerifyTransfer(ctx, nil, nil)
		require.NoError(t, err)
	})

	t.Run("DeserializeTransferAction", func(t *testing.T) {
		ppm := &mockPublicParamsManager{PublicParamsManager: &mock.PublicParamsManager{}}
		ws := &mock.WalletService{}
		tl := &MockTokenLoader{}
		des := &mock.Deserializer{}
		s := v1.NewTransferService(logger, ppm, ws, tl, des)

		action := &actions.TransferAction{
			Inputs: []*actions.TransferActionInput{
				{
					ID: &token.ID{TxId: "tx1", Index: 0},
					Input: &actions.Output{
						Owner:    []byte("owner1"),
						Type:     "type1",
						Quantity: "10",
					},
				},
			},
			Outputs: []*actions.Output{
				{
					Owner:    []byte("owner2"),
					Type:     "type1",
					Quantity: "10",
				},
			},
		}
		raw, err := action.Serialize()
		require.NoError(t, err)

		desAction, err := s.DeserializeTransferAction(raw)
		require.NoError(t, err)
		assert.NotNil(t, desAction)
		assert.Equal(t, action.Inputs[0].ID, desAction.GetInputs()[0])

		_, err = s.DeserializeTransferAction([]byte("invalid"))
		require.Error(t, err)
	})
}
