/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/stretchr/testify/assert"
	"github.com/test-go/testify/require"
)

func TestTypedIdentity_Bytes(t *testing.T) {
	ti := identity.TypedIdentity{
		Type:     "testType",
		Identity: driver.Identity("testIdentity"),
	}

	bytes, err := ti.Bytes()
	require.NoError(t, err)
	assert.NotNil(t, bytes)
}

func TestTypedIdentity_Bytes_Error(t *testing.T) {
	ti := identity.TypedIdentity{
		Type:     identity.Type([]byte{0xff, 0xfe, 0xfd}),
		Identity: driver.Identity("testIdentity"),
	}

	_, err := ti.Bytes()
	require.Error(t, err)
}

func TestUnmarshalTypedIdentity(t *testing.T) {
	ti := identity.TypedIdentity{
		Type:     "testType",
		Identity: driver.Identity("testIdentity"),
	}

	bytes, err := ti.Bytes()
	require.NoError(t, err)

	unmarshaledTI, err := identity.UnmarshalTypedIdentity(bytes)
	require.NoError(t, err)
	assert.Equal(t, ti, *unmarshaledTI)
}

func TestUnmarshalTypedIdentity_Error(t *testing.T) {
	invalidBytes := []byte{0xff, 0xfe, 0xfd}

	_, err := identity.UnmarshalTypedIdentity(invalidBytes)
	require.Error(t, err)
}

func TestWrapWithType(t *testing.T) {
	idType := "testType"
	id := driver.Identity("testIdentity")

	wrappedID, err := identity.WrapWithType(idType, id)
	require.NoError(t, err)
	assert.NotNil(t, wrappedID)

	unmarshaledTI, err := identity.UnmarshalTypedIdentity(wrappedID)
	require.NoError(t, err)
	assert.Equal(t, idType, unmarshaledTI.Type)
	assert.Equal(t, id, unmarshaledTI.Identity)
}

func TestWrapWithType_Error(t *testing.T) {
	idType := string([]byte{0xff, 0xfe, 0xfd})
	id := driver.Identity("testIdentity")

	_, err := identity.WrapWithType(idType, id)
	require.Error(t, err)
}
