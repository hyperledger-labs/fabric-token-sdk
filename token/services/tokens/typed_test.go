/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/driver"
	"github.com/stretchr/testify/assert"
	"github.com/test-go/testify/require"
)

func TestSerialization(t *testing.T) {
	raw := []byte("pineapple")
	wrappedToken, err := tokens.WrapWithType(0, raw)
	require.NoError(t, err)
	tok, err := tokens.UnmarshalTypedToken(wrappedToken)
	require.NoError(t, err)
	assert.Equal(t, driver.Type(0), tok.Type)
	assert.Equal(t, driver.Token(raw), tok.Token)
}

func TestTypedToken_Bytes(t *testing.T) {
	tt := tokens.TypedToken{Type: 42, Token: []byte("data")}
	b, err := tt.Bytes()
	require.NoError(t, err)
	assert.NotEmpty(t, b)
}

func TestUnmarshalTypedToken_Error(t *testing.T) {
	_, err := tokens.UnmarshalTypedToken([]byte("not-asn1-data-xxxxxxxxxxx"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal to TypedToken")
}

func TestTypedMetadata_RoundTrip(t *testing.T) {
	raw := []byte("meta-data")
	wrapped, err := tokens.WrapMetadataWithType(2, raw)
	require.NoError(t, err)
	assert.NotEmpty(t, wrapped)

	tm, err := tokens.UnmarshalTypedMetadata(wrapped)
	require.NoError(t, err)
	assert.Equal(t, driver.Type(2), tm.Type)
	assert.Equal(t, driver.Metadata(raw), tm.Metadata)
}

func TestTypedMetadata_Bytes(t *testing.T) {
	tm := tokens.TypedMetadata{Type: 1, Metadata: []byte("meta")}
	b, err := tm.Bytes()
	require.NoError(t, err)
	assert.NotEmpty(t, b)
}

func TestUnmarshalTypedMetadata_Error(t *testing.T) {
	_, err := tokens.UnmarshalTypedMetadata([]byte("not-valid-asn1-xxxxxxxxxxxx"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal to TypedMetadata")
}
