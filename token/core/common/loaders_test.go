/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	dmock "github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVaultLedgerTokenLoaderWithCounterfeiter(t *testing.T) {
	logger := &logging.MockLogger{}
	ids := []*token.ID{{TxId: "tx1", Index: 0}}
	ctx := context.Background()

	t.Run("GetTokenOutputs_Success", func(t *testing.T) {
		vault := &dmock.TokenVault{}
		deserializer := &dmock.TokenDeserializer[string]{}
		loader := NewLedgerTokenLoader[string](logger, nil, vault, deserializer)

		vault.GetTokenOutputsReturns(nil)
		vault.GetTokenOutputsCalls(func(ctx context.Context, ids []*token.ID, callback driver.QueryCallbackFunc) error {
			return callback(ids[0], []byte("token-raw"))
		})
		deserializer.DeserializeTokenReturns("token-deserialized", nil)

		res, err := loader.GetTokenOutputs(ctx, ids)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"tx1": "token-deserialized"}, res)
	})

	t.Run("GetTokenOutputs_EmptyBytes", func(t *testing.T) {
		vault := &dmock.TokenVault{}
		deserializer := &dmock.TokenDeserializer[string]{}
		loader := NewLedgerTokenLoader[string](logger, nil, vault, deserializer)
		loader.NumRetries = 1

		vault.GetTokenOutputsCalls(func(ctx context.Context, ids []*token.ID, callback driver.QueryCallbackFunc) error {
			return callback(ids[0], nil)
		})
		vault.IsPendingReturns(false, nil)

		res, err := loader.GetTokenOutputs(ctx, ids)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed getting serialized token output for id [[tx1:0]], nil value")
		assert.Nil(t, res)
	})

	t.Run("GetTokenOutputs_RetryOnPending", func(t *testing.T) {
		vault := &dmock.TokenVault{}
		deserializer := &dmock.TokenDeserializer[string]{}
		loader := NewLedgerTokenLoader[string](logger, nil, vault, deserializer)
		loader.NumRetries = 2
		loader.RetryDelay = 1 * time.Millisecond

		vault.GetTokenOutputsReturnsOnCall(0, errors.New("not found"))
		vault.IsPendingReturnsOnCall(0, true, nil)

		vault.GetTokenOutputsReturnsOnCall(1, nil)
		vault.GetTokenOutputsCalls(func(ctx context.Context, ids []*token.ID, callback driver.QueryCallbackFunc) error {
			if vault.GetTokenOutputsCallCount() == 2 {
				return callback(ids[0], []byte("token-raw"))
			}

			return errors.New("not found")
		})
		deserializer.DeserializeTokenReturns("token-deserialized", nil)

		res, err := loader.GetTokenOutputs(ctx, ids)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"tx1": "token-deserialized"}, res)
		assert.Equal(t, 2, vault.GetTokenOutputsCallCount())
	})

	t.Run("GetTokenOutputs_DeserializationError", func(t *testing.T) {
		vault := &dmock.TokenVault{}
		deserializer := &dmock.TokenDeserializer[string]{}
		loader := NewLedgerTokenLoader[string](logger, nil, vault, deserializer)
		loader.NumRetries = 1

		vault.GetTokenOutputsCalls(func(ctx context.Context, ids []*token.ID, callback driver.QueryCallbackFunc) error {
			return callback(ids[0], []byte("invalid"))
		})
		deserializer.DeserializeTokenReturns("", errors.New("bad token"))
		vault.IsPendingReturns(false, nil)

		res, err := loader.GetTokenOutputs(ctx, ids)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad token")
		assert.Nil(t, res)
	})

	t.Run("GetTokenOutputs_IsPendingError", func(t *testing.T) {
		vault := &dmock.TokenVault{}
		deserializer := &dmock.TokenDeserializer[string]{}
		loader := NewLedgerTokenLoader[string](logger, nil, vault, deserializer)
		loader.NumRetries = 1

		vault.GetTokenOutputsReturns(errors.New("not found"))
		vault.IsPendingReturns(false, errors.New("pending check failed"))

		res, err := loader.GetTokenOutputs(ctx, ids)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pending check failed")
		assert.Nil(t, res)
	})
}

func TestVaultLedgerTokenAndMetadataLoaderWithCounterfeiter(t *testing.T) {
	ids := []*token.ID{{TxId: "tx1", Index: 0}}
	ctx := context.Background()

	t.Run("LoadTokens_Success", func(t *testing.T) {
		vault := &dmock.TokenVault{}
		deserializer := &dmock.TokenAndMetadataDeserializer[string, string]{}
		loader := NewVaultLedgerTokenAndMetadataLoader[string, string](vault, deserializer)
		outputs := [][]byte{[]byte("output1")}
		metadata := [][]byte{[]byte("meta1")}
		formats := []token.Format{token.Format("f1")}
		vault.GetTokenOutputsAndMetaReturns(outputs, metadata, formats, nil)
		deserializer.DeserializeTokenReturns("tok1", nil)
		deserializer.DeserializeMetadataReturns("meta1-des", nil)

		res, err := loader.LoadTokens(ctx, ids)
		require.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, "tok1", res[0].Token)
		assert.Equal(t, "meta1-des", res[0].Metadata)
	})

	t.Run("LoadTokens_VaultError", func(t *testing.T) {
		vault := &dmock.TokenVault{}
		deserializer := &dmock.TokenAndMetadataDeserializer[string, string]{}
		loader := NewVaultLedgerTokenAndMetadataLoader[string, string](vault, deserializer)
		vault.GetTokenOutputsAndMetaReturns(nil, nil, nil, errors.New("vault error"))
		res, err := loader.LoadTokens(ctx, ids)
		require.Error(t, err)
		assert.Equal(t, "vault error", err.Error())
		assert.Nil(t, res)
	})

	t.Run("LoadTokens_EmptyOutput", func(t *testing.T) {
		vault := &dmock.TokenVault{}
		deserializer := &dmock.TokenAndMetadataDeserializer[string, string]{}
		loader := NewVaultLedgerTokenAndMetadataLoader[string, string](vault, deserializer)
		outputs := [][]byte{nil}
		metadata := [][]byte{[]byte("meta1")}
		formats := []token.Format{token.Format("f1")}
		vault.GetTokenOutputsAndMetaReturns(outputs, metadata, formats, nil)
		_, err := loader.LoadTokens(ctx, ids)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil comm value")
	})

	t.Run("LoadTokens_EmptyMetadata", func(t *testing.T) {
		vault := &dmock.TokenVault{}
		deserializer := &dmock.TokenAndMetadataDeserializer[string, string]{}
		loader := NewVaultLedgerTokenAndMetadataLoader[string, string](vault, deserializer)
		outputs := [][]byte{[]byte("output1")}
		metadata := [][]byte{nil}
		formats := []token.Format{token.Format("f1")}
		vault.GetTokenOutputsAndMetaReturns(outputs, metadata, formats, nil)
		_, err := loader.LoadTokens(ctx, ids)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil info value")
	})

	t.Run("LoadTokens_DeserializeTokenError", func(t *testing.T) {
		vault := &dmock.TokenVault{}
		deserializer := &dmock.TokenAndMetadataDeserializer[string, string]{}
		loader := NewVaultLedgerTokenAndMetadataLoader[string, string](vault, deserializer)
		outputs := [][]byte{[]byte("output1")}
		metadata := [][]byte{[]byte("meta1")}
		formats := []token.Format{token.Format("f1")}
		vault.GetTokenOutputsAndMetaReturns(outputs, metadata, formats, nil)
		deserializer.DeserializeTokenReturns("", errors.New("bad token"))

		_, err := loader.LoadTokens(ctx, ids)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad token")
	})

	t.Run("LoadTokens_DeserializeMetadataError", func(t *testing.T) {
		vault := &dmock.TokenVault{}
		deserializer := &dmock.TokenAndMetadataDeserializer[string, string]{}
		loader := NewVaultLedgerTokenAndMetadataLoader[string, string](vault, deserializer)
		outputs := [][]byte{[]byte("output1")}
		metadata := [][]byte{[]byte("meta1")}
		formats := []token.Format{token.Format("f1")}
		vault.GetTokenOutputsAndMetaReturns(outputs, metadata, formats, nil)
		deserializer.DeserializeTokenReturns("tok1", nil)
		deserializer.DeserializeMetadataReturns("", errors.New("bad meta"))

		_, err := loader.LoadTokens(ctx, ids)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad meta")
	})
}

func TestVaultTokenInfoLoaderWithCounterfeiter(t *testing.T) {
	ids := []*token.ID{{TxId: "tx1", Index: 0}}
	ctx := context.Background()

	t.Run("GetTokenInfos_Success", func(t *testing.T) {
		qe := &dmock.QueryEngine{}
		deserializer := &dmock.MetadataDeserializer[string]{}
		loader := NewVaultTokenInfoLoader[string](qe, deserializer)
		metadata := [][]byte{[]byte("meta1")}
		qe.GetTokenMetadataReturns(metadata, nil)
		deserializer.DeserializeMetadataReturns("meta1-des", nil)

		res, err := loader.GetTokenInfos(ctx, ids)
		require.NoError(t, err)
		assert.Equal(t, []string{"meta1-des"}, res)
	})

	t.Run("GetTokenInfos_QueryError", func(t *testing.T) {
		qe := &dmock.QueryEngine{}
		deserializer := &dmock.MetadataDeserializer[string]{}
		loader := NewVaultTokenInfoLoader[string](qe, deserializer)
		qe.GetTokenMetadataReturns(nil, errors.New("query error"))
		res, err := loader.GetTokenInfos(ctx, ids)
		require.Error(t, err)
		assert.Equal(t, "query error", err.Error())
		assert.Nil(t, res)
	})

	t.Run("GetTokenInfos_DeserializeError", func(t *testing.T) {
		qe := &dmock.QueryEngine{}
		deserializer := &dmock.MetadataDeserializer[string]{}
		loader := NewVaultTokenInfoLoader[string](qe, deserializer)
		metadata := [][]byte{[]byte("meta1")}
		qe.GetTokenMetadataReturns(metadata, nil)
		deserializer.DeserializeMetadataReturns("", errors.New("bad meta"))

		_, err := loader.GetTokenInfos(ctx, ids)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad meta")
	})
}

func TestVaultTokenLoaderWithCounterfeiter(t *testing.T) {
	ids := []*token.ID{{TxId: "tx1", Index: 0}}
	ctx := context.Background()

	t.Run("GetTokens_Success", func(t *testing.T) {
		qe := &dmock.QueryEngine{}
		loader := NewVaultTokenLoader(qe)
		tokens := []*token.Token{{Owner: []byte("owner1")}}
		qe.GetTokensReturns(tokens, nil)

		res, err := loader.GetTokens(ctx, ids)
		require.NoError(t, err)
		assert.Equal(t, tokens, res)
	})
}

func TestVaultTokenCertificationLoaderWithCounterfeiter(t *testing.T) {
	ids := []*token.ID{{TxId: "tx1", Index: 0}}
	ctx := context.Background()

	t.Run("GetCertifications_Success", func(t *testing.T) {
		storage := &dmock.TokenCertificationStorage{}
		loader := &VaultTokenCertificationLoader{TokenCertificationStorage: storage}
		certs := [][]byte{[]byte("cert1")}
		storage.GetCertificationsReturns(certs, nil)

		res, err := loader.GetCertifications(ctx, ids)
		require.NoError(t, err)
		assert.Equal(t, certs, res)

		storage.GetCertificationsReturns(nil, errors.New("cert error"))
		res, err = loader.GetCertifications(ctx, ids)
		require.Error(t, err)
		assert.Equal(t, "cert error", err.Error())
		assert.Nil(t, res)
	})
}

func TestIdentityTokenAndMetadataDeserializerWithCounterfeiter(t *testing.T) {
	d := IdentityTokenAndMetadataDeserializer{}

	b := []byte("hello")
	res, err := d.DeserializeToken(b)
	require.NoError(t, err)
	assert.Equal(t, b, res)

	res, err = d.DeserializeMetadata(b)
	require.NoError(t, err)
	assert.Equal(t, b, res)
}
