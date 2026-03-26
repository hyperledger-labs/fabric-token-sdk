/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"context"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/multisig/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test that deserializing a serialized multi-id returns the original
func TestMultiIdentity_SerializeDeserialize(t *testing.T) {
	identities := identities(t, "id1", "id2")
	mi := &MultiIdentity{Identities: identities}

	serialized, err := mi.Serialize()
	require.NoError(t, err)

	deserialized := &MultiIdentity{}
	err = deserialized.Deserialize(serialized)
	require.NoError(t, err)

	assert.Equal(t, mi, deserialized)
}

// Test wrapping multi-identities together
// and then unwrapping them back to the original ids
func TestWrapIdentities(t *testing.T) {
	identities := identities(t, "id1", "id2")
	wrapped, err := WrapIdentities(identities...)
	require.NoError(t, err)

	unwrapped, isMultisig, err := Unwrap(wrapped)
	require.NoError(t, err)
	assert.True(t, isMultisig)
	assert.Equal(t, identities, unwrapped)
}

// GTest failure to unwrap invalid raw wrapped ids
func TestUnwrap_InvalidIdentity(t *testing.T) {
	invalidIdentity := []byte("invalid")
	unwrapped, isMultisig, err := Unwrap(invalidIdentity)
	require.Error(t, err)
	assert.False(t, isMultisig)
	assert.Nil(t, unwrapped)
}

// Test using a multi Info-matcher with two matchers against a raw multi-id with two ids
func TestInfoMatcher_Match(t *testing.T) {
	identities := identities(t, "id1", "id2")
	mi := &MultiIdentity{Identities: identities}
	serialized, err := mi.Serialize()
	require.NoError(t, err)

	matcher1 := &mock.Matcher{}
	matcher1.MatchReturns(nil)
	matcher2 := &mock.Matcher{}
	matcher2.MatchReturns(nil)

	matchers := []driver.Matcher{matcher1, matcher2}
	infoMatcher := &InfoMatcher{AuditInfoMatcher: matchers}

	err = infoMatcher.Match(context.Background(), serialized)
	require.NoError(t, err)

	// Verify each matcher was called once
	assert.Equal(t, 1, matcher1.MatchCallCount())
	assert.Equal(t, 1, matcher2.MatchCallCount())
}

// Test a multi-matcher with two matchers that fail to match a multi-id of two ids
// because one of the (mock) matchers fails to match
func TestInfoMatcher_Match_Invalid(t *testing.T) {
	identities := identities(t, "id1", "id2")
	mi := &MultiIdentity{Identities: identities}
	serialized, err := mi.Serialize()
	require.NoError(t, err)

	matcher1 := &mock.Matcher{}
	matcher1.MatchReturns(nil)
	matcher2 := &mock.Matcher{}
	matcher2.MatchReturns(errors.New("mismatch"))

	matchers := []driver.Matcher{matcher1, matcher2}
	infoMatcher := &InfoMatcher{AuditInfoMatcher: matchers}

	err = infoMatcher.Match(context.Background(), serialized)
	require.Error(t, err)

	// Verify first matcher was called, second matcher was called and returned error
	assert.Equal(t, 1, matcher1.MatchCallCount())
	assert.Equal(t, 1, matcher2.MatchCallCount())
}

// Test wrapping and unwrapping a multi-AuditInfo with two AuditInfos
func TestWrapAuditInfo(t *testing.T) {
	auditInfos := [][]byte{[]byte("audit1"), []byte("audit2")}
	wrapped, err := WrapAuditInfo(auditInfos)
	require.NoError(t, err)

	isMultisig, unwrapped, err := UnwrapAuditInfo(wrapped)
	require.NoError(t, err)
	assert.True(t, isMultisig)
	assert.Equal(t, auditInfos, unwrapped)
}

// Test failure to unwrap an invalid wrapped multi-AuditInfo
func TestUnwrapAuditInfo_Invalid(t *testing.T) {
	invalidInfo := []byte("invalid")
	isMultisig, unwrapped, err := UnwrapAuditInfo(invalidInfo)
	require.Error(t, err)
	assert.False(t, isMultisig)
	assert.Nil(t, unwrapped)
}

// Test failure to deserialize invalid raw MultiIdentity
func TestMultiIdentity_Deserialize_Error(t *testing.T) {
	mi := &MultiIdentity{}
	err := mi.Deserialize([]byte("invalid"))
	require.Error(t, err)
}

// Test failure to wrap an empty list of multi-ids
func TestWrapIdentities_Error(t *testing.T) {
	_, err := WrapIdentities()
	require.Error(t, err)
}

// Test failure to unwrap an invalid wrapped multi-identity
func TestUnwrap_Error(t *testing.T) {
	_, _, err := Unwrap([]byte("invalid"))
	require.Error(t, err)
}

// Test failure to use a multi-matcher with two matchers
// to match against an invalid raw data
func TestInfoMatcher_Match_Error(t *testing.T) {
	invalidSerialized := []byte("invalid")

	matcher1 := &mock.Matcher{}
	matcher2 := &mock.Matcher{}

	matchers := []driver.Matcher{matcher1, matcher2}
	infoMatcher := &InfoMatcher{AuditInfoMatcher: matchers}

	err := infoMatcher.Match(context.Background(), invalidSerialized)
	require.Error(t, err)

	// Verify no matchers were called (failed during deserialization)
	assert.Equal(t, 0, matcher1.MatchCallCount())
	assert.Equal(t, 0, matcher2.MatchCallCount())
}

// Test failure to wrap AuditInfos from a nil list
func TestWrapAuditInfo_Error(t *testing.T) {
	_, err := WrapAuditInfo(nil)
	require.Error(t, err)
}

// Test failure to unwrap invalid raw wrapped AuditInfos
func TestUnwrapAuditInfo_Error(t *testing.T) {
	_, _, err := UnwrapAuditInfo([]byte("invalid"))
	require.Error(t, err)
}

func identities(t *testing.T, names ...string) []token.Identity {
	t.Helper()
	ids := make([]token.Identity, len(names))
	for i, name := range names {
		ids[i] = wrapIdentity(t, name)
	}

	return ids
}

func wrapIdentity(t *testing.T, name string) token.Identity {
	t.Helper()
	id, err := identity.WrapWithType("name", []byte(name))
	require.NoError(t, err)

	return id
}

// Test extracting the EID (EnrollmentID) from a multi-IdentityAuditInfos
func TestAuditInfo_EnrollmentID(t *testing.T) {
	ai := &AuditInfo{
		IdentityAuditInfos: []IdentityAuditInfo{
			{AuditInfo: []byte("audit1")},
		},
	}

	eid := ai.EnrollmentID()
	assert.Empty(t, eid)
}

// Test extracting the RH (RevocationHandle) from a multi-IdentityAuditInfos
func TestAuditInfo_RevocationHandle(t *testing.T) {
	ai := &AuditInfo{
		IdentityAuditInfos: []IdentityAuditInfo{
			{AuditInfo: []byte("audit1")},
		},
	}

	rh := ai.RevocationHandle()
	assert.Empty(t, rh)
}

// Test serializing multi-IdentityAuditInfo and then deserializing this
// and comparing the result with the original
func TestAuditInfo_Bytes(t *testing.T) {
	ai := &AuditInfo{
		IdentityAuditInfos: []IdentityAuditInfo{
			{AuditInfo: []byte("audit1")},
			{AuditInfo: []byte("audit2")},
		},
	}

	bytes, err := ai.Bytes()
	require.NoError(t, err)
	assert.NotEmpty(t, bytes)

	// Verify it can be unmarshaled back
	ai2 := &AuditInfo{}
	err = json.Unmarshal(bytes, ai2)
	require.NoError(t, err)
	assert.Equal(t, ai, ai2)
}

// Test comparing the serializations of multi-ids with Bytes() and with Serialize()
func TestMultiIdentity_Bytes(t *testing.T) {
	identities := identities(t, "id1", "id2")
	mi := &MultiIdentity{Identities: identities}

	bytes, err := mi.Bytes()
	require.NoError(t, err)
	assert.NotEmpty(t, bytes)

	// Verify it matches Serialize
	serialized, err := mi.Serialize()
	require.NoError(t, err)
	assert.Equal(t, serialized, bytes)
}

// Test wrapping a multi-sig of a given type and then unwrapping the result
func TestUnwrap_NotMultisig(t *testing.T) {
	// Create a non-multisig typed identity
	nonMultisigID, err := identity.WrapWithType("other", []byte("data"))
	require.NoError(t, err)

	unwrapped, isMultisig, err := Unwrap(nonMultisigID)
	require.NoError(t, err)
	assert.False(t, isMultisig)
	assert.Nil(t, unwrapped)
}

// Test failure to use a multi-matcher with just 2 matchers
// to match the AuditInfo of a multi-id with 3 ids
func TestInfoMatcher_Match_CountMismatch(t *testing.T) {
	identities := identities(t, "id1", "id2", "id3")
	mi := &MultiIdentity{Identities: identities}
	serialized, err := mi.Serialize()
	require.NoError(t, err)

	// Only 2 matchers for 3 identities
	matcher1 := &mock.Matcher{}
	matcher2 := &mock.Matcher{}

	matchers := []driver.Matcher{matcher1, matcher2}
	infoMatcher := &InfoMatcher{AuditInfoMatcher: matchers}

	err = infoMatcher.Match(context.Background(), serialized)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected")
	assert.Contains(t, err.Error(), "identities, received")

	// Verify no matchers were called (count mismatch detected before matching)
	assert.Equal(t, 0, matcher1.MatchCallCount())
	assert.Equal(t, 0, matcher2.MatchCallCount())
}
