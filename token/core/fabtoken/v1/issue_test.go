/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"context"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIssueService(t *testing.T) {
	ppm := &mock.PublicParamsManager{}
	ws := &mock.WalletService{}
	des := &mock.Deserializer{}
	service := NewIssueService(ppm, ws, des)
	assert.NotNil(t, service)
	assert.Equal(t, ppm, service.PublicParamsManager)
	assert.Equal(t, ws, service.WalletService)
	assert.Equal(t, des, service.Deserializer)
}

func TestIssue(t *testing.T) {
	ctx := context.Background()
	issuer := driver.Identity("issuer")
	tokenType := token.Type("ABC")
	values := []uint64{100, 200}
	owners := [][]byte{[]byte("owner1"), []byte("owner2")}

	t.Run("Success", func(t *testing.T) {
		ppm := &mock.PublicParamsManager{}
		ws := &mock.WalletService{}
		des := &mock.Deserializer{}
		service := NewIssueService(ppm, ws, des)

		pp := &mock.PublicParameters{}
		pp.PrecisionReturns(64)
		ppm.PublicParametersReturns(pp)

		des.GetAuditInfoReturns([]byte("audit-info"), nil)

		action, metadata, err := service.Issue(ctx, issuer, tokenType, values, owners, nil)
		require.NoError(t, err)
		assert.NotNil(t, action)
		assert.NotNil(t, metadata)
		assert.Equal(t, issuer, metadata.Issuer.Identity)
		assert.Equal(t, []byte("audit-info"), metadata.Issuer.AuditInfo)
		assert.Len(t, metadata.Outputs, 2)
	})

	t.Run("EmptyOwnerError", func(t *testing.T) {
		service := NewIssueService(nil, nil, nil)
		_, _, err := service.Issue(ctx, issuer, tokenType, values, [][]byte{[]byte("owner1"), nil}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "all recipients should be defined")
	})

	t.Run("InvalidArgumentsError", func(t *testing.T) {
		service := NewIssueService(nil, nil, nil)
		_, _, err := service.Issue(ctx, nil, "", nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "issuer identity, token type and values should be defined")
	})

	t.Run("RedeemNotSupported", func(t *testing.T) {
		service := NewIssueService(nil, nil, nil)
		opts := &driver.IssueOptions{
			TokensUpgradeRequest: &driver.TokenUpgradeRequest{},
		}
		_, _, err := service.Issue(ctx, issuer, tokenType, values, owners, opts)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "redeem during issue is not supported")
	})

	t.Run("GetAuditInfoError", func(t *testing.T) {
		ppm := &mock.PublicParamsManager{}
		ws := &mock.WalletService{}
		des := &mock.Deserializer{}
		service := NewIssueService(ppm, ws, des)

		pp := &mock.PublicParameters{}
		pp.PrecisionReturns(64)
		ppm.PublicParametersReturns(pp)

		des.GetAuditInfoReturns(nil, assert.AnError)

		_, _, err := service.Issue(ctx, issuer, tokenType, values, owners, nil)
		require.Error(t, err)
	})

	t.Run("IssuerAuditInfoError", func(t *testing.T) {
		ppm := &mock.PublicParamsManager{}
		ws := &mock.WalletService{}
		des := &mock.Deserializer{}
		service := NewIssueService(ppm, ws, des)

		pp := &mock.PublicParameters{}
		pp.PrecisionReturns(64)
		ppm.PublicParametersReturns(pp)

		// Success for owners in loop (called twice)
		des.GetAuditInfoReturnsOnCall(0, []byte("audit1"), nil)
		des.GetAuditInfoReturnsOnCall(1, []byte("audit2"), nil)
		// Failure for issuer
		des.GetAuditInfoReturnsOnCall(2, nil, assert.AnError)

		_, _, err := service.Issue(ctx, issuer, tokenType, values, owners, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get audit info for issuer identity")
	})

	t.Run("PrecisionError", func(t *testing.T) {
		ppm := &mock.PublicParamsManager{}
		ws := &mock.WalletService{}
		des := &mock.Deserializer{}
		service := NewIssueService(ppm, ws, des)

		pp := &mock.PublicParameters{}
		pp.PrecisionReturns(0) // Invalid precision
		ppm.PublicParametersReturns(pp)

		_, _, err := service.Issue(ctx, issuer, tokenType, values, owners, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to convert")
	})

	t.Run("IssueWithAttributes", func(t *testing.T) {
		ppm := &mock.PublicParamsManager{}
		ws := &mock.WalletService{}
		des := &mock.Deserializer{}
		service := NewIssueService(ppm, ws, des)

		pp := &mock.PublicParameters{}
		pp.PrecisionReturns(64)
		ppm.PublicParametersReturns(pp)
		des.GetAuditInfoReturns([]byte("audit"), nil)

		opts := &driver.IssueOptions{
			Attributes: map[interface{}]interface{}{"key": "value"},
		}
		action, _, err := service.Issue(ctx, issuer, tokenType, values, owners, opts)
		require.NoError(t, err)
		assert.NotNil(t, action)
	})
}

func TestVerifyIssue(t *testing.T) {
	service := NewIssueService(nil, nil, nil)
	err := service.VerifyIssue(context.Background(), nil, nil)
	require.NoError(t, err)
}

func TestDeserializeIssueAction(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ppm := &mock.PublicParamsManager{}
		pp := &mock.PublicParameters{}
		pp.PrecisionReturns(64)
		ppm.PublicParametersReturns(pp)
		des := &mock.Deserializer{}
		des.GetAuditInfoReturns([]byte("audit"), nil)
		s := NewIssueService(ppm, nil, des)

		issuer := driver.Identity("issuer")
		action, _, err := s.Issue(context.Background(), issuer, "ABC", []uint64{100}, [][]byte{[]byte("owner")}, nil)
		require.NoError(t, err)

		raw, err := action.Serialize()
		require.NoError(t, err)

		service := NewIssueService(nil, nil, nil)
		deserialized, err := service.DeserializeIssueAction(raw)
		require.NoError(t, err)
		assert.NotNil(t, deserialized)
	})

	t.Run("Error", func(t *testing.T) {
		service := NewIssueService(nil, nil, nil)
		_, err := service.DeserializeIssueAction([]byte("invalid"))
		require.Error(t, err)
	})
}
