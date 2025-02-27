/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/stretchr/testify/assert"
)

func TestMultiIdentity_SerializeDeserialize(t *testing.T) {
	identities := identities(t, "id1", "id2")
	mi := &MultiIdentity{Identities: identities}

	serialized, err := mi.Serialize()
	assert.NoError(t, err)

	deserialized := &MultiIdentity{}
	err = deserialized.Deserialize(serialized)
	assert.NoError(t, err)

	assert.Equal(t, mi, deserialized)
}

func TestWrapIdentities(t *testing.T) {
	identities := identities(t, "id1", "id2")
	wrapped, err := WrapIdentities(identities...)
	assert.NoError(t, err)

	isMultisig, unwrapped, err := Unwrap(wrapped)
	assert.NoError(t, err)
	assert.True(t, isMultisig)
	assert.Equal(t, identities, unwrapped)
}

func TestUnwrap_InvalidIdentity(t *testing.T) {
	invalidIdentity := []byte("invalid")
	isMultisig, unwrapped, err := Unwrap(invalidIdentity)
	assert.Error(t, err)
	assert.False(t, isMultisig)
	assert.Nil(t, unwrapped)
}

func TestInfoMatcher_Match(t *testing.T) {
	identities := identities(t, "id1", "id2")
	mi := &MultiIdentity{Identities: identities}
	serialized, err := mi.Serialize()
	assert.NoError(t, err)

	matchers := []driver.Matcher{
		&mockMatcher{expected: []byte("id1")},
		&mockMatcher{expected: []byte("id2")},
	}
	infoMatcher := &InfoMatcher{AuditInfoMatcher: matchers}

	err = infoMatcher.Match(serialized)
	assert.NoError(t, err)
}

func TestInfoMatcher_Match_Invalid(t *testing.T) {
	identities := identities(t, "id1", "id2")
	mi := &MultiIdentity{Identities: identities}
	serialized, err := mi.Serialize()
	assert.NoError(t, err)

	matchers := []driver.Matcher{
		&mockMatcher{expected: []byte("id1")},
		&mockMatcher{expected: []byte("id3")},
	}
	infoMatcher := &InfoMatcher{AuditInfoMatcher: matchers}

	err = infoMatcher.Match(serialized)
	assert.Error(t, err)
}

func TestWrapAuditInfo(t *testing.T) {
	auditInfos := [][]byte{[]byte("audit1"), []byte("audit2")}
	wrapped, err := WrapAuditInfo(auditInfos)
	assert.NoError(t, err)

	isMultisig, unwrapped, err := UnwrapAuditInfo(wrapped)
	assert.NoError(t, err)
	assert.True(t, isMultisig)
	assert.Equal(t, auditInfos, unwrapped)
}

func TestUnwrapAuditInfo_Invalid(t *testing.T) {
	invalidInfo := []byte("invalid")
	isMultisig, unwrapped, err := UnwrapAuditInfo(invalidInfo)
	assert.Error(t, err)
	assert.False(t, isMultisig)
	assert.Nil(t, unwrapped)
}

func TestMultiIdentity_Deserialize_Error(t *testing.T) {
	mi := &MultiIdentity{}
	err := mi.Deserialize([]byte("invalid"))
	assert.Error(t, err)
}

func TestWrapIdentities_Error(t *testing.T) {
	_, err := WrapIdentities()
	assert.Error(t, err)
}

func TestUnwrap_Error(t *testing.T) {
	_, _, err := Unwrap([]byte("invalid"))
	assert.Error(t, err)
}

func TestInfoMatcher_Match_Error(t *testing.T) {
	invalidSerialized := []byte("invalid")
	matchers := []driver.Matcher{
		&mockMatcher{expected: []byte("id1")},
		&mockMatcher{expected: []byte("id2")},
	}
	infoMatcher := &InfoMatcher{AuditInfoMatcher: matchers}

	err := infoMatcher.Match(invalidSerialized)
	assert.Error(t, err)
}

func TestWrapAuditInfo_Error(t *testing.T) {
	_, err := WrapAuditInfo(nil)
	assert.Error(t, err)
}

func TestUnwrapAuditInfo_Error(t *testing.T) {
	_, _, err := UnwrapAuditInfo([]byte("invalid"))
	assert.Error(t, err)
}

func mulsigIdentities(t *testing.T, names ...string) token.Identity {
	id, err := WrapIdentities(identities(t, names...)...)
	assert.NoError(t, err)
	return id
}

func identities(t *testing.T, names ...string) []token.Identity {
	identities := make([]token.Identity, len(names))
	var err error
	for i, name := range names {
		identities[i], err = identity.WrapWithType("name", []byte(name))
		assert.NoError(t, err)
	}
	return identities
}

type mockMatcher struct {
	expected []byte
}

func (m *mockMatcher) Match(raw []byte) error {
	if string(raw) != string(m.expected) {
		return errors.New("mismatch")
	}
	return nil
}
