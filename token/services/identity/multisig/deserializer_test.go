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
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeserializeVerifier_Success(t *testing.T) {
	verifierDES := &mockVerifierDES{}
	deserializer := NewTypedIdentityDeserializer(verifierDES, nil)
	id := &MultiIdentity{Identities: []token.Identity{[]byte("valid_multisig_identity")}}
	idRaw, err := id.Bytes()
	require.NoError(t, err)
	verifier, err := deserializer.DeserializeVerifier(Multisig, idRaw)

	require.NoError(t, err)
	assert.NotNil(t, verifier)
}

func TestDeserializeVerifier_InvalidMultisigIdentity(t *testing.T) {
	verifierDES := &mockVerifierDES{}
	deserializer := NewTypedIdentityDeserializer(verifierDES, nil)
	id := []byte("invalid")

	verifier, err := deserializer.DeserializeVerifier(Multisig, id)
	require.Error(t, err)
	assert.Nil(t, verifier)
}

func TestDeserializeVerifier_InvalidIdentity(t *testing.T) {
	verifierDES := &mockVerifierDES{}
	deserializer := NewTypedIdentityDeserializer(verifierDES, nil)
	id := &MultiIdentity{Identities: []token.Identity{[]byte("invalid")}}
	idRaw, err := id.Bytes()
	require.NoError(t, err)

	verifier, err := deserializer.DeserializeVerifier(Multisig, idRaw)
	require.Error(t, err)
	assert.Nil(t, verifier)
}

func TestGetOwnerAuditInfo_Success(t *testing.T) {
	verifierDES := &mockVerifierDES{}
	auditInfoMatcher := &mockAuditInfoMatcher{}
	deserializer := NewTypedIdentityDeserializer(verifierDES, auditInfoMatcher)
	id := []byte("valid_multisig_identity")
	rawIdentity := []byte("valid_raw_identity")
	provider := &mockAuditInfoProvider{}

	auditInfo, err := deserializer.GetAuditInfo(context.Background(), id, Multisig, rawIdentity, provider)
	require.NoError(t, err)
	assert.NotNil(t, auditInfo)
}

func TestGetAuditInfo_InvalidType(t *testing.T) {
	verifierDES := &mockVerifierDES{}
	auditInfoMatcher := &mockAuditInfoMatcher{}
	deserializer := NewTypedIdentityDeserializer(verifierDES, auditInfoMatcher)
	id := []byte("valid_multisig_identity")
	rawIdentity := []byte("valid_raw_identity")
	provider := &mockAuditInfoProvider{}

	auditInfo, err := deserializer.GetAuditInfo(context.Background(), id, identity.Type("InvalidType"), rawIdentity, provider)
	require.Error(t, err)
	assert.Nil(t, auditInfo)
}

func TestGetOwnerMatcher_InvalidAuditInfo(t *testing.T) {
	verifierDES := &mockVerifierDES{}
	auditInfoMatcher := &mockAuditInfoMatcher{}
	deserializer := NewTypedIdentityDeserializer(verifierDES, auditInfoMatcher)
	owner := []byte("valid_owner")
	auditInfo := []byte("invalid")

	matcher, err := deserializer.GetAuditInfoMatcher(owner, auditInfo)
	require.Error(t, err)
	assert.Nil(t, matcher)
}

func TestRecipients_Success(t *testing.T) {
	deserializer := NewTypedIdentityDeserializer(nil, nil)
	id := &MultiIdentity{Identities: []token.Identity{[]byte("valid_multisig_identity")}}
	idRaw, err := id.Bytes()
	require.NoError(t, err)

	recipients, err := deserializer.Recipients(nil, Multisig, idRaw)
	require.NoError(t, err)
	assert.NotNil(t, recipients)
}

func TestRecipients_InvalidRaw(t *testing.T) {
	verifierDES := &mockVerifierDES{}
	deserializer := NewTypedIdentityDeserializer(verifierDES, nil)
	id := []byte("valid_multisig_identity")
	raw := []byte("invalid")

	recipients, err := deserializer.Recipients(id, Multisig, raw)
	require.Error(t, err)
	assert.Nil(t, recipients)
}

type mockVerifierDES struct{}

func (m *mockVerifierDES) DeserializeVerifier(id token.Identity) (driver.Verifier, error) {
	if string(id) == "valid_multisig_identity" {
		return &mockVerifier{}, nil
	}
	return nil, errors.New("invalid identity")
}

type mockAuditInfoMatcher struct{}

func (m *mockAuditInfoMatcher) GetAuditInfoMatcher(owner token.Identity, auditInfo []byte) (driver.Matcher, error) {
	if string(auditInfo) == "valid_audit_info" {
		return &mockMatcher{}, nil
	}
	return nil, errors.New("invalid audit info")
}

type mockAuditInfoProvider struct{}

func (m *mockAuditInfoProvider) GetAuditInfo(ctx context.Context, id token.Identity) ([]byte, error) {
	if string(id) == "valid_multisig_identity" {
		return []byte("valid_audit_info"), nil
	}
	return nil, errors.New("invalid identity")
}

type mockVerifier struct{}

func (m *mockVerifier) Verify(message, signature []byte) error {
	return nil
}
