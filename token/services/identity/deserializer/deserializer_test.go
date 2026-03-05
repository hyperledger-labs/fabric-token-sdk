/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deserializer

import (
	"context"
	"errors"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type mockAuditInfo struct {
	enrollmentID     string
	revocationHandle string
}

func (m *mockAuditInfo) EnrollmentID() string {
	return m.enrollmentID
}

func (m *mockAuditInfo) RevocationHandle() string {
	return m.revocationHandle
}

type mockMatcher struct {
	matchFunc func(ctx context.Context, id []byte) error
}

func (m *mockMatcher) Match(ctx context.Context, id []byte) error {
	if m.matchFunc != nil {
		return m.matchFunc(ctx, id)
	}

	return nil
}

type mockVerifier struct {
	verifyFunc func(message, sigma []byte) error
}

func (m *mockVerifier) Verify(message, sigma []byte) error {
	if m.verifyFunc != nil {
		return m.verifyFunc(message, sigma)
	}

	return nil
}

type mockSigner struct {
	signFunc func(message []byte) ([]byte, error)
}

func (m *mockSigner) Sign(message []byte) ([]byte, error) {
	if m.signFunc != nil {
		return m.signFunc(message)
	}

	return []byte("signature"), nil
}

type mockTypedVerifierDeserializer struct {
	deserializeVerifierFunc func(ctx context.Context, typ identity.Type, raw []byte) (driver.Verifier, error)
	recipientsFunc          func(id driver.Identity, typ identity.Type, raw []byte) ([]driver.Identity, error)
	getAuditInfoFunc        func(ctx context.Context, id driver.Identity, typ identity.Type, raw []byte, p driver.AuditInfoProvider) ([]byte, error)
	getAuditInfoMatcherFunc func(ctx context.Context, owner driver.Identity, auditInfo []byte) (driver.Matcher, error)
}

func (m *mockTypedVerifierDeserializer) DeserializeVerifier(ctx context.Context, typ identity.Type, raw []byte) (driver.Verifier, error) {
	if m.deserializeVerifierFunc != nil {
		return m.deserializeVerifierFunc(ctx, typ, raw)
	}

	return &mockVerifier{}, nil
}

func (m *mockTypedVerifierDeserializer) Recipients(id driver.Identity, typ identity.Type, raw []byte) ([]driver.Identity, error) {
	if m.recipientsFunc != nil {
		return m.recipientsFunc(id, typ, raw)
	}

	return []driver.Identity{id}, nil
}

func (m *mockTypedVerifierDeserializer) GetAuditInfo(ctx context.Context, id driver.Identity, typ identity.Type, raw []byte, p driver.AuditInfoProvider) ([]byte, error) {
	if m.getAuditInfoFunc != nil {
		return m.getAuditInfoFunc(ctx, id, typ, raw, p)
	}

	return []byte("audit-info"), nil
}

func (m *mockTypedVerifierDeserializer) GetAuditInfoMatcher(ctx context.Context, owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	if m.getAuditInfoMatcherFunc != nil {
		return m.getAuditInfoMatcherFunc(ctx, owner, auditInfo)
	}

	return &mockMatcher{}, nil
}

type mockTypedSignerDeserializer struct {
	deserializeSignerFunc func(ctx context.Context, typ idriver.IdentityType, raw []byte) (driver.Signer, error)
}

func (m *mockTypedSignerDeserializer) DeserializeSigner(ctx context.Context, typ idriver.IdentityType, raw []byte) (driver.Signer, error) {
	if m.deserializeSignerFunc != nil {
		return m.deserializeSignerFunc(ctx, typ, raw)
	}

	return &mockSigner{}, nil
}

type mockAuditInfoProvider struct {
	getAuditInfoFunc func(ctx context.Context, id driver.Identity) ([]byte, error)
}

func (m *mockAuditInfoProvider) GetAuditInfo(ctx context.Context, id driver.Identity) ([]byte, error) {
	if m.getAuditInfoFunc != nil {
		return m.getAuditInfoFunc(ctx, id)
	}

	return []byte("audit-info"), nil
}

// Helper function to create a typed identity
func createTypedIdentity(t *testing.T, idType string, rawIdentity []byte) driver.Identity {
	t.Helper()

	ti := identity.TypedIdentity{
		Type:     idType,
		Identity: rawIdentity,
	}
	bytes, err := ti.Bytes()
	require.NoError(t, err)

	return bytes
}

// TestTypedAuditInfoMatcher tests the TypedAuditInfoMatcher
func TestTypedAuditInfoMatcher(t *testing.T) {
	// Tests the TypedAuditInfoMatcher running a valid match
	// i.e. the typed ID is correctly deserialized and the Match
	// succeeds on the deserialized id
	t.Run("Success", func(t *testing.T) {
		matcher := &mockMatcher{
			matchFunc: func(ctx context.Context, id []byte) error {
				assert.Equal(t, []byte("raw-identity"), id)

				return nil
			},
		}

		typedMatcher := &TypedAuditInfoMatcher{matcher: matcher}
		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))

		err := typedMatcher.Match(context.Background(), typedID)
		require.NoError(t, err)
	})

	// Tests the TypedAuditInfoMatcher running a failing match
	// due to mismatching typed identity
	t.Run("UnmarshalError", func(t *testing.T) {
		matcher := &mockMatcher{}
		typedMatcher := &TypedAuditInfoMatcher{matcher: matcher}

		err := typedMatcher.Match(context.Background(), []byte("invalid-identity"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal identity")
	})

	// Tests the TypedAuditInfoMatcher running a failing match
	// due to arbitrary matching issue
	t.Run("MatchError", func(t *testing.T) {
		expectedErr := errors.New("match failed")
		matcher := &mockMatcher{
			matchFunc: func(ctx context.Context, id []byte) error {
				return expectedErr
			},
		}

		typedMatcher := &TypedAuditInfoMatcher{matcher: matcher}
		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))

		err := typedMatcher.Match(context.Background(), typedID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to match identity")
	})
}

// TestEIDRHDeserializer tests the EIDRHDeserializer
// that deserializes audit information including
// the Enrollment ID and the Revocation Handle
func TestEIDRHDeserializer(t *testing.T) {
	// test success in getting the EID from the EIDRHDeserializer
	t.Run("GetEnrollmentID_Success", func(t *testing.T) {
		deserializer := NewEIDRHDeserializer()

		mockDeserializer := &mock.AuditInfoDeserializer{}
		mockDeserializer.DeserializeAuditInfoReturns(&mockAuditInfo{
			enrollmentID:     "test-eid",
			revocationHandle: "test-rh",
		}, nil)

		deserializer.AddDeserializer("test-type", mockDeserializer)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		eid, err := deserializer.GetEnrollmentID(context.Background(), typedID, []byte("audit-info"))

		require.NoError(t, err)
		assert.Equal(t, "test-eid", eid)
	})

	// test success in getting the RevocationHandler from the EIDRHDeserializer
	t.Run("GetRevocationHandler_Success", func(t *testing.T) {
		deserializer := NewEIDRHDeserializer()

		mockDeserializer := &mock.AuditInfoDeserializer{}
		mockDeserializer.DeserializeAuditInfoReturns(&mockAuditInfo{
			enrollmentID:     "test-eid",
			revocationHandle: "test-rh",
		}, nil)

		deserializer.AddDeserializer("test-type", mockDeserializer)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		rh, err := deserializer.GetRevocationHandler(context.Background(), typedID, []byte("audit-info"))

		require.NoError(t, err)
		assert.Equal(t, "test-rh", rh)
	})

	// test success in getting the both the EID and the RevocationHandler from the
	// EIDRHDeserializer using GetEIDAndRH
	t.Run("GetEIDAndRH_Success", func(t *testing.T) {
		deserializer := NewEIDRHDeserializer()

		mockDeserializer := &mock.AuditInfoDeserializer{}
		mockDeserializer.DeserializeAuditInfoReturns(&mockAuditInfo{
			enrollmentID:     "test-eid",
			revocationHandle: "test-rh",
		}, nil)

		deserializer.AddDeserializer("test-type", mockDeserializer)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		eid, rh, err := deserializer.GetEIDAndRH(context.Background(), typedID, []byte("audit-info"))

		require.NoError(t, err)
		assert.Equal(t, "test-eid", eid)
		assert.Equal(t, "test-rh", rh)
	})

	// test error when trying to deserialize and extract the EID
	// when the audit info is nil
	t.Run("NilAuditInfo", func(t *testing.T) {
		deserializer := NewEIDRHDeserializer()
		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))

		_, err := deserializer.GetEnrollmentID(context.Background(), typedID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil audit info")
	})

	// test error when trying to deserialize and extract the EID
	// when the audit info is empty
	t.Run("EmptyAuditInfo", func(t *testing.T) {
		deserializer := NewEIDRHDeserializer()
		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))

		_, err := deserializer.GetEnrollmentID(context.Background(), typedID, []byte{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil audit info")
	})

	// test error when trying to deserialize and extract the EID
	// when the typed identity is invalid
	t.Run("UnmarshalError", func(t *testing.T) {
		deserializer := NewEIDRHDeserializer()

		_, err := deserializer.GetEnrollmentID(context.Background(), []byte("invalid"), []byte("audit-info"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal to TypedIdentity")
	})

	// test error when trying to deserialize and extract the EID
	// when the typed identity is of an unrecognized type
	t.Run("NoDeserializerFound", func(t *testing.T) {
		deserializer := NewEIDRHDeserializer()
		typedID := createTypedIdentity(t, "unknown-type", []byte("raw-identity"))

		_, err := deserializer.GetEnrollmentID(context.Background(), typedID, []byte("audit-info"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no deserializer found")
	})

	// test error when trying to deserialize and extract the EID
	// when the NewEIDRHDeserializer returns with an arbitrary error
	t.Run("DeserializeAuditInfoError", func(t *testing.T) {
		deserializer := NewEIDRHDeserializer()

		mockDeserializer := &mock.AuditInfoDeserializer{}
		mockDeserializer.DeserializeAuditInfoReturns(nil, errors.New("deserialize error"))

		deserializer.AddDeserializer("test-type", mockDeserializer)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		_, err := deserializer.GetEnrollmentID(context.Background(), typedID, []byte("audit-info"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to deserialize audit info")
	})
}

// TestTypedSignerDeserializerMultiplex tests the TypedSignerDeserializerMultiplex
// that deserializes a transaction signer
func TestTypedSignerDeserializerMultiplex(t *testing.T) {

	// test success in getting the signer from the TypedSignerDeserializerMultiplex
	t.Run("DeserializeSigner_Success", func(t *testing.T) {
		multiplex := NewTypedSignerDeserializerMultiplex()

		mockDeserializer := &mockTypedSignerDeserializer{
			deserializeSignerFunc: func(ctx context.Context, typ idriver.IdentityType, raw []byte) (driver.Signer, error) {
				return &mockSigner{}, nil
			},
		}

		multiplex.AddTypedSignerDeserializer("test-type", mockDeserializer)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		signer, err := multiplex.DeserializeSigner(context.Background(), typedID)

		require.NoError(t, err)
		assert.NotNil(t, signer)
	})

	// test error when trying to deserialize and extract the signer
	// when the typed identity is invalid
	t.Run("DeserializeSigner_UnmarshalError", func(t *testing.T) {
		multiplex := NewTypedSignerDeserializerMultiplex()

		_, err := multiplex.DeserializeSigner(context.Background(), []byte("invalid"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal to TypedIdentity")
	})

	// test error when trying to deserialize and extract the signer
	// when the TypedSignerDeserializerMultiplex wasn't set
	// with any TypedSignerDeserializer
	t.Run("DeserializeSigner_NoDeserializerFound", func(t *testing.T) {
		multiplex := NewTypedSignerDeserializerMultiplex()
		typedID := createTypedIdentity(t, "unknown-type", []byte("raw-identity"))

		_, err := multiplex.DeserializeSigner(context.Background(), typedID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no deserializer found")
	})

	// Test that when the TypedSignerDeserializerMultiplex is set with two deserializers
	// for the same type, then if the first one succeeds then the 2nd one isn't used
	t.Run("DeserializeSigner_MultipleDeserializers_FirstSucceeds", func(t *testing.T) {
		multiplex := NewTypedSignerDeserializerMultiplex()

		mockDeserializer1 := &mockTypedSignerDeserializer{
			deserializeSignerFunc: func(ctx context.Context, typ idriver.IdentityType, raw []byte) (driver.Signer, error) {
				return &mockSigner{}, nil
			},
		}
		mockDeserializer2 := &mockTypedSignerDeserializer{
			deserializeSignerFunc: func(ctx context.Context, typ idriver.IdentityType, raw []byte) (driver.Signer, error) {
				return nil, errors.New("should not be called")
			},
		}

		multiplex.AddTypedSignerDeserializer("test-type", mockDeserializer1)
		multiplex.AddTypedSignerDeserializer("test-type", mockDeserializer2)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		signer, err := multiplex.DeserializeSigner(context.Background(), typedID)

		require.NoError(t, err)
		assert.NotNil(t, signer)
	})

	// Test that when the TypedSignerDeserializerMultiplex is set with two deserializers
	// for the same type, then if the first one fails then the 2nd one is given a chance
	t.Run("DeserializeSigner_MultipleDeserializers_SecondSucceeds", func(t *testing.T) {
		multiplex := NewTypedSignerDeserializerMultiplex()

		mockDeserializer1 := &mockTypedSignerDeserializer{
			deserializeSignerFunc: func(ctx context.Context, typ idriver.IdentityType, raw []byte) (driver.Signer, error) {
				return nil, errors.New("first fails")
			},
		}
		mockDeserializer2 := &mockTypedSignerDeserializer{
			deserializeSignerFunc: func(ctx context.Context, typ idriver.IdentityType, raw []byte) (driver.Signer, error) {
				return &mockSigner{}, nil
			},
		}

		multiplex.AddTypedSignerDeserializer("test-type", mockDeserializer1)
		multiplex.AddTypedSignerDeserializer("test-type", mockDeserializer2)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		signer, err := multiplex.DeserializeSigner(context.Background(), typedID)

		require.NoError(t, err)
		assert.NotNil(t, signer)
	})

	// Test that when the TypedSignerDeserializerMultiplex is set with two deserializers
	// for the same type, then if both of them fail then the deserialization fails
	t.Run("DeserializeSigner_AllDeserializersFail", func(t *testing.T) {
		multiplex := NewTypedSignerDeserializerMultiplex()

		mockDeserializer1 := &mockTypedSignerDeserializer{
			deserializeSignerFunc: func(ctx context.Context, typ idriver.IdentityType, raw []byte) (driver.Signer, error) {
				return nil, errors.New("first fails")
			},
		}
		mockDeserializer2 := &mockTypedSignerDeserializer{
			deserializeSignerFunc: func(ctx context.Context, typ idriver.IdentityType, raw []byte) (driver.Signer, error) {
				return nil, errors.New("second fails")
			},
		}

		multiplex.AddTypedSignerDeserializer("test-type", mockDeserializer1)
		multiplex.AddTypedSignerDeserializer("test-type", mockDeserializer2)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		_, err := multiplex.DeserializeSigner(context.Background(), typedID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to deserialize verifier")
	})
}

// TestTypedVerifierDeserializerMultiplex tests the TypedVerifierDeserializerMultiplex
// that deserializes a signature verifier
func TestTypedVerifierDeserializerMultiplex(t *testing.T) {
	// test success in getting the verifier from the TypedVerifierDeserializerMultiplex
	t.Run("DeserializeVerifier_Success", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		mockDeserializer := &mockTypedVerifierDeserializer{}
		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		verifier, err := multiplex.DeserializeVerifier(context.Background(), typedID)

		require.NoError(t, err)
		assert.NotNil(t, verifier)
	})

	// test error when trying to deserialize and extract the verifier
	// when the typed identity is invalid
	t.Run("DeserializeVerifier_UnmarshalError", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		_, err := multiplex.DeserializeVerifier(context.Background(), []byte("invalid"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal to TypedIdentity")
	})

	// test error when trying to deserialize and extract the verifier
	// when the TypedVerifierDeserializerMultiplex wasn't set
	// with any TypedVerifierDeserializer
	t.Run("DeserializeVerifier_NoDeserializerFound", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()
		typedID := createTypedIdentity(t, "unknown-type", []byte("raw-identity"))

		_, err := multiplex.DeserializeVerifier(context.Background(), typedID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no deserializer found")
	})

	// Test that when the TypedVerifierDeserializerMultiplex is set with a deserializer
	// that fails then the deserialization fails when using an id of the corresponding type
	t.Run("DeserializeVerifier_AllDeserializersFail", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		mockDeserializer := &mockTypedVerifierDeserializer{
			deserializeVerifierFunc: func(ctx context.Context, typ identity.Type, raw []byte) (driver.Verifier, error) {
				return nil, errors.New("deserialize failed")
			},
		}
		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		_, err := multiplex.DeserializeVerifier(context.Background(), typedID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to deserialize verifier")
	})

	// Test success in getting the recipient identities from a typed identity
	t.Run("Recipients_Success", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		mockDeserializer := &mockTypedVerifierDeserializer{}
		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		recipients, err := multiplex.Recipients(typedID)

		require.NoError(t, err)
		assert.NotNil(t, recipients)
		assert.Len(t, recipients, 1)
	})

	// Test that getting the recipient identities from a nil identity
	// returns a nil recepients list
	t.Run("Recipients_NoneIdentity", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		recipients, err := multiplex.Recipients(driver.Identity(nil))
		require.NoError(t, err)
		assert.Nil(t, recipients)
	})

	// Test that getting the recipient identities from an invalid identity
	// returns error
	t.Run("Recipients_UnmarshalError", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		_, err := multiplex.Recipients([]byte("invalid"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal to TypedIdentity")
	})

	// ?????test error when trying to deserialize and extract the TypedVerifier
	// when the TypedVerifierDeserializerMultiplex wasn't set
	// with any TypedSignerDeserializer
	t.Run("Recipients_NoDeserializerFound", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()
		typedID := createTypedIdentity(t, "unknown-type", []byte("raw-identity"))

		_, err := multiplex.Recipients(typedID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no deserializer found")
	})

	t.Run("Recipients_AllDeserializersFail", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		mockDeserializer := &mockTypedVerifierDeserializer{
			recipientsFunc: func(id driver.Identity, typ identity.Type, raw []byte) ([]driver.Identity, error) {
				return nil, errors.New("recipients failed")
			},
		}
		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		_, err := multiplex.Recipients(typedID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to deserializer recipients")
	})

	t.Run("GetAuditInfoMatcher_Success", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		mockDeserializer := &mockTypedVerifierDeserializer{}
		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		matcher, err := multiplex.GetAuditInfoMatcher(context.Background(), typedID, []byte("audit-info"))

		require.NoError(t, err)
		assert.NotNil(t, matcher)
	})

	t.Run("GetAuditInfoMatcher_NoneIdentity", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		matcher, err := multiplex.GetAuditInfoMatcher(context.Background(), driver.Identity(nil), []byte("audit-info"))
		require.NoError(t, err)
		assert.Nil(t, matcher)
	})

	t.Run("GetAuditInfoMatcher_UnmarshalError", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		_, err := multiplex.GetAuditInfoMatcher(context.Background(), []byte("invalid"), []byte("audit-info"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal to TypedIdentity")
	})

	t.Run("GetAuditInfoMatcher_NoDeserializerFound", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()
		typedID := createTypedIdentity(t, "unknown-type", []byte("raw-identity"))

		_, err := multiplex.GetAuditInfoMatcher(context.Background(), typedID, []byte("audit-info"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no deserializer found")
	})

	t.Run("MatchIdentity_Success", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		mockDeserializer := &mockTypedVerifierDeserializer{}
		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		err := multiplex.MatchIdentity(context.Background(), typedID, []byte("audit-info"))

		require.NoError(t, err)
	})

	t.Run("MatchIdentity_UnmarshalError", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		err := multiplex.MatchIdentity(context.Background(), []byte("invalid"), []byte("audit-info"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal identity")
	})

	t.Run("MatchIdentity_GetMatcherError", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()
		typedID := createTypedIdentity(t, "unknown-type", []byte("raw-identity"))

		err := multiplex.MatchIdentity(context.Background(), typedID, []byte("audit-info"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed getting audit info matcher")
	})

	t.Run("MatchIdentity_MatchError", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		mockDeserializer := &mockTypedVerifierDeserializer{
			getAuditInfoMatcherFunc: func(ctx context.Context, owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
				return &mockMatcher{
					matchFunc: func(ctx context.Context, id []byte) error {
						return errors.New("match failed")
					},
				}, nil
			},
		}
		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		err := multiplex.MatchIdentity(context.Background(), typedID, []byte("audit-info"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to match identity to audit infor")
	})

	t.Run("GetAuditInfo_Success", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		mockDeserializer := &mockTypedVerifierDeserializer{}
		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		auditInfo, err := multiplex.GetAuditInfo(context.Background(), typedID, &mockAuditInfoProvider{})

		require.NoError(t, err)
		assert.NotNil(t, auditInfo)
	})

	t.Run("GetAuditInfo_UnmarshalError", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		_, err := multiplex.GetAuditInfo(context.Background(), []byte("invalid"), &mockAuditInfoProvider{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal to TypedIdentity")
	})

	t.Run("GetAuditInfo_NoDeserializerFound", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()
		typedID := createTypedIdentity(t, "unknown-type", []byte("raw-identity"))

		_, err := multiplex.GetAuditInfo(context.Background(), typedID, &mockAuditInfoProvider{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no deserializer found")
	})

	t.Run("GetAuditInfo_AllDeserializersFail", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		mockDeserializer := &mockTypedVerifierDeserializer{
			getAuditInfoFunc: func(ctx context.Context, id driver.Identity, typ identity.Type, raw []byte, p driver.AuditInfoProvider) ([]byte, error) {
				return nil, errors.New("get audit info failed")
			},
		}
		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		_, err := multiplex.GetAuditInfo(context.Background(), typedID, &mockAuditInfoProvider{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find a valid deserializer for audit info")
	})

	t.Run("AddTypedVerifierDeserializer_MultipleTypes", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		mockDeserializer1 := &mockTypedVerifierDeserializer{}
		mockDeserializer2 := &mockTypedVerifierDeserializer{}

		// Add first deserializer for type1
		multiplex.AddTypedVerifierDeserializer("type1", mockDeserializer1)

		// Add second deserializer for type2
		multiplex.AddTypedVerifierDeserializer("type2", mockDeserializer2)

		// Verify both types work
		typedID1 := createTypedIdentity(t, "type1", []byte("raw-identity"))
		verifier1, err := multiplex.DeserializeVerifier(context.Background(), typedID1)
		require.NoError(t, err)
		assert.NotNil(t, verifier1)

		typedID2 := createTypedIdentity(t, "type2", []byte("raw-identity"))
		verifier2, err := multiplex.DeserializeVerifier(context.Background(), typedID2)
		require.NoError(t, err)
		assert.NotNil(t, verifier2)
	})

	t.Run("GetMatcher_MultipleDeserializers_SecondSucceeds", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		mockDeserializer1 := &mockTypedVerifierDeserializer{
			getAuditInfoMatcherFunc: func(ctx context.Context, owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
				return nil, errors.New("first matcher fails")
			},
		}
		mockDeserializer2 := &mockTypedVerifierDeserializer{
			getAuditInfoMatcherFunc: func(ctx context.Context, owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
				return &mockMatcher{}, nil
			},
		}

		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer1)
		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer2)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		matcher, err := multiplex.GetAuditInfoMatcher(context.Background(), typedID, []byte("audit-info"))

		require.NoError(t, err)
		assert.NotNil(t, matcher)
	})

	t.Run("DeserializeVerifier_MultipleDeserializers_SecondSucceeds", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		mockDeserializer1 := &mockTypedVerifierDeserializer{
			deserializeVerifierFunc: func(ctx context.Context, typ identity.Type, raw []byte) (driver.Verifier, error) {
				return nil, errors.New("first fails")
			},
		}
		mockDeserializer2 := &mockTypedVerifierDeserializer{
			deserializeVerifierFunc: func(ctx context.Context, typ identity.Type, raw []byte) (driver.Verifier, error) {
				return &mockVerifier{}, nil
			},
		}

		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer1)
		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer2)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		verifier, err := multiplex.DeserializeVerifier(context.Background(), typedID)

		require.NoError(t, err)
		assert.NotNil(t, verifier)
	})

	t.Run("Recipients_MultipleDeserializers_SecondSucceeds", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		mockDeserializer1 := &mockTypedVerifierDeserializer{
			recipientsFunc: func(id driver.Identity, typ identity.Type, raw []byte) ([]driver.Identity, error) {
				return nil, errors.New("first fails")
			},
		}
		mockDeserializer2 := &mockTypedVerifierDeserializer{
			recipientsFunc: func(id driver.Identity, typ identity.Type, raw []byte) ([]driver.Identity, error) {
				return []driver.Identity{id}, nil
			},
		}

		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer1)
		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer2)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		recipients, err := multiplex.Recipients(typedID)

		require.NoError(t, err)
		assert.NotNil(t, recipients)
		assert.Len(t, recipients, 1)
	})

	t.Run("GetAuditInfo_MultipleDeserializers_SecondSucceeds", func(t *testing.T) {
		multiplex := NewTypedVerifierDeserializerMultiplex()

		mockDeserializer1 := &mockTypedVerifierDeserializer{
			getAuditInfoFunc: func(ctx context.Context, id driver.Identity, typ identity.Type, raw []byte, p driver.AuditInfoProvider) ([]byte, error) {
				return nil, errors.New("first fails")
			},
		}
		mockDeserializer2 := &mockTypedVerifierDeserializer{
			getAuditInfoFunc: func(ctx context.Context, id driver.Identity, typ identity.Type, raw []byte, p driver.AuditInfoProvider) ([]byte, error) {
				return []byte("audit-info"), nil
			},
		}

		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer1)
		multiplex.AddTypedVerifierDeserializer("test-type", mockDeserializer2)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		auditInfo, err := multiplex.GetAuditInfo(context.Background(), typedID, &mockAuditInfoProvider{})

		require.NoError(t, err)
		assert.NotNil(t, auditInfo)
	})
}

// TestTypedIdentityVerifierDeserializer tests the TypedIdentityVerifierDeserializer
func TestTypedIdentityVerifierDeserializer(t *testing.T) {
	t.Run("DeserializeVerifier_Success", func(t *testing.T) {
		mockVerifierDeserializer := &mockVerifierDeserializer{
			deserializeVerifierFunc: func(ctx context.Context, id driver.Identity) (driver.Verifier, error) {
				return &mockVerifier{}, nil
			},
		}
		mockMatcherDeserializer := &mockMatcherDeserializer{}

		deserializer := NewTypedIdentityVerifierDeserializer(mockVerifierDeserializer, mockMatcherDeserializer)

		verifier, err := deserializer.DeserializeVerifier(context.Background(), "test-type", []byte("raw-identity"))
		require.NoError(t, err)
		assert.NotNil(t, verifier)
	})

	t.Run("Recipients_Success", func(t *testing.T) {
		mockVerifierDeserializer := &mockVerifierDeserializer{}
		mockMatcherDeserializer := &mockMatcherDeserializer{}

		deserializer := NewTypedIdentityVerifierDeserializer(mockVerifierDeserializer, mockMatcherDeserializer)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		recipients, err := deserializer.Recipients(typedID, "test-type", []byte("raw-identity"))

		require.NoError(t, err)
		assert.Len(t, recipients, 1)
		assert.Equal(t, typedID, recipients[0])
	})

	t.Run("GetAuditInfo_Success", func(t *testing.T) {
		mockVerifierDeserializer := &mockVerifierDeserializer{}
		mockMatcherDeserializer := &mockMatcherDeserializer{}

		deserializer := NewTypedIdentityVerifierDeserializer(mockVerifierDeserializer, mockMatcherDeserializer)

		provider := &mockAuditInfoProvider{
			getAuditInfoFunc: func(ctx context.Context, id driver.Identity) ([]byte, error) {
				return []byte("audit-info"), nil
			},
		}

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		auditInfo, err := deserializer.GetAuditInfo(context.Background(), typedID, "test-type", []byte("raw-identity"), provider)

		require.NoError(t, err)
		assert.Equal(t, []byte("audit-info"), auditInfo)
	})

	t.Run("GetAuditInfo_ProviderError", func(t *testing.T) {
		mockVerifierDeserializer := &mockVerifierDeserializer{}
		mockMatcherDeserializer := &mockMatcherDeserializer{}

		deserializer := NewTypedIdentityVerifierDeserializer(mockVerifierDeserializer, mockMatcherDeserializer)

		provider := &mockAuditInfoProvider{
			getAuditInfoFunc: func(ctx context.Context, id driver.Identity) ([]byte, error) {
				return nil, errors.New("provider error")
			},
		}

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		_, err := deserializer.GetAuditInfo(context.Background(), typedID, "test-type", []byte("raw-identity"), provider)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed getting audit info for recipient identity")
	})

	t.Run("GetAuditInfoMatcher_Success", func(t *testing.T) {
		mockVerifierDeserializer := &mockVerifierDeserializer{}
		mockMatcherDeserializer := &mockMatcherDeserializer{
			getAuditInfoMatcherFunc: func(ctx context.Context, owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
				return &mockMatcher{}, nil
			},
		}

		deserializer := NewTypedIdentityVerifierDeserializer(mockVerifierDeserializer, mockMatcherDeserializer)

		typedID := createTypedIdentity(t, "test-type", []byte("raw-identity"))
		matcher, err := deserializer.GetAuditInfoMatcher(context.Background(), typedID, []byte("audit-info"))

		require.NoError(t, err)
		assert.NotNil(t, matcher)
	})
}

// Mock implementations for TypedIdentityVerifierDeserializer tests

type mockVerifierDeserializer struct {
	deserializeVerifierFunc func(ctx context.Context, id driver.Identity) (driver.Verifier, error)
}

func (m *mockVerifierDeserializer) DeserializeVerifier(ctx context.Context, id driver.Identity) (driver.Verifier, error) {
	if m.deserializeVerifierFunc != nil {
		return m.deserializeVerifierFunc(ctx, id)
	}

	return &mockVerifier{}, nil
}

type mockMatcherDeserializer struct {
	getAuditInfoMatcherFunc func(ctx context.Context, owner driver.Identity, auditInfo []byte) (driver.Matcher, error)
}

func (m *mockMatcherDeserializer) GetAuditInfoMatcher(ctx context.Context, owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	if m.getAuditInfoMatcherFunc != nil {
		return m.getAuditInfoMatcherFunc(ctx, owner, auditInfo)
	}

	return &mockMatcher{}, nil
}

// Made with Bob
