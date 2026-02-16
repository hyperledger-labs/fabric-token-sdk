/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"context"
	"testing"

	idemix "github.com/IBM/idemix/bccsp/types"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/mock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestAuditInfo_Serialization(t *testing.T) {
	auditInfo := &AuditInfo{
		Attributes: [][]byte{
			[]byte("attr1"),
			[]byte("attr2"),
			[]byte("enrollment-id"),
			[]byte("revocation-handle"),
		},
		Schema: "test-schema",
	}

	// Test Bytes() serialization
	bytes, err := auditInfo.Bytes()
	assert.NoError(t, err)
	assert.NotNil(t, bytes)
	assert.Greater(t, len(bytes), 0)

	// Test FromBytes() round-trip
	newAuditInfo := &AuditInfo{}
	err = newAuditInfo.FromBytes(bytes)
	assert.NoError(t, err)
	assert.Equal(t, auditInfo.Schema, newAuditInfo.Schema)
	assert.Equal(t, len(auditInfo.Attributes), len(newAuditInfo.Attributes))

	// Test FromBytes() with invalid JSON
	err = newAuditInfo.FromBytes([]byte("invalid json"))
	assert.Error(t, err)

	// Test FromBytes() with empty JSON
	err = newAuditInfo.FromBytes([]byte("{}"))
	assert.NoError(t, err)
}

func TestAuditInfo_Accessors(t *testing.T) {
	auditInfo := &AuditInfo{
		Attributes: [][]byte{
			[]byte("attr0"),
			[]byte("attr1"),
			[]byte("test-enrollment-id"),
			[]byte("test-revocation-handle"),
		},
	}

	assert.Equal(t, "test-enrollment-id", auditInfo.EnrollmentID())
	assert.Equal(t, "test-revocation-handle", auditInfo.RevocationHandle())
}

// Test in/valid de/serialization cases for AuditInfo
func TestDeserializeAuditInfo(t *testing.T) {
	t.Run("Valid audit info", func(t *testing.T) {
		original := &AuditInfo{
			Attributes: [][]byte{[]byte("attr1"), []byte("attr2")},
			Schema:     "test-schema",
		}

		bytes, err := original.Bytes()
		require.NoError(t, err)

		result, err := DeserializeAuditInfo(bytes)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, original.Schema, result.Schema)
		assert.Equal(t, len(original.Attributes), len(result.Attributes))
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		_, err := DeserializeAuditInfo([]byte("invalid json"))
		assert.Error(t, err)
	})

	t.Run("Empty attributes", func(t *testing.T) {
		auditInfo := &AuditInfo{Attributes: [][]byte{}, Schema: "test-schema"}
		bytes, err := auditInfo.Bytes()
		require.NoError(t, err)

		_, err = DeserializeAuditInfo(bytes)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no attributes found")
	})
}

func TestAuditInfo_Match(t *testing.T) {
	// Helper to create valid identity bytes
	createIdentity := func() []byte {
		serialized := &SerializedIdemixIdentity{
			NymPublicKey: []byte("fake-nym-key"),
			Proof:        []byte("fake-proof"),
			Schema:       "test-schema",
		}
		identityBytes, _ := proto.Marshal(serialized)
		return identityBytes
	}

	// Helper to create base audit info
	createAuditInfo := func() *AuditInfo {
		return &AuditInfo{
			Attributes: [][]byte{
				[]byte("attr0"),
				[]byte("attr1"),
				[]byte("enrollment-id"),
				[]byte("revocation-handle"),
			},
			Schema: "test-schema",
		}
	}

	t.Run("Invalid protobuf identity", func(t *testing.T) {
		auditInfo := createAuditInfo()
		err := auditInfo.Match(context.Background(), []byte("invalid protobuf"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not deserialize a SerializedIdemixIdentity")
	})

	t.Run("SchemaManager errors", func(t *testing.T) {
		tests := []struct {
			name          string
			setupMock     func(*mock.SchemaManager, *mock.BCCSP)
			setupAuditInfo func(*AuditInfo)
			expectedError string
		}{
			{
				name: "EidNymAuditOpts error",
				setupMock: func(sm *mock.SchemaManager, _ *mock.BCCSP) {
					sm.EidNymAuditOptsReturns(nil, errors.New("schema error"))
				},
				setupAuditInfo: func(ai *AuditInfo) {},
				expectedError:  "error while getting a RhNymAuditOpts",
			},
			{
				name: "RhNymAuditOpts error",
				setupMock: func(sm *mock.SchemaManager, csp *mock.BCCSP) {
					sm.EidNymAuditOptsReturns(&idemix.EidNymAuditOpts{EidIndex: 2, EnrollmentID: "enrollment-id"}, nil)
					sm.RhNymAuditOptsReturns(nil, errors.New("rh schema error"))
					csp.VerifyReturns(true, nil)
				},
				setupAuditInfo: func(ai *AuditInfo) {
					ai.EidNymAuditData = &idemix.AttrNymAuditData{}
					ai.RhNymAuditData = &idemix.AttrNymAuditData{}
				},
				expectedError: "error while getting a RhNymAuditOpts",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockCsp := &mock.BCCSP{}
				mockSchemaManager := &mock.SchemaManager{}
				tt.setupMock(mockSchemaManager, mockCsp)

				auditInfo := createAuditInfo()
				auditInfo.Csp = mockCsp
				auditInfo.IssuerPublicKey = &mock.Key{}
				auditInfo.SchemaManager = mockSchemaManager
				tt.setupAuditInfo(auditInfo)

				err := auditInfo.Match(context.Background(), createIdentity())
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			})
		}
	})

	t.Run("EidNymAuditData is nil - panic", func(t *testing.T) {
		mockSchemaManager := &mock.SchemaManager{}
		mockSchemaManager.EidNymAuditOptsReturns(&idemix.EidNymAuditOpts{EidIndex: 2, EnrollmentID: "enrollment-id"}, nil)

		auditInfo := createAuditInfo()
		auditInfo.SchemaManager = mockSchemaManager
		auditInfo.EidNymAuditData = nil

		assert.Panics(t, func() {
			_ = auditInfo.Match(context.Background(), createIdentity())
		})
	})

	t.Run("BCCSP Verify errors and failures", func(t *testing.T) {
		tests := []struct {
			name           string
			setupMock      func(*mock.BCCSP, *mock.SchemaManager)
			setupAuditInfo func(*AuditInfo)
			expectedError  string
		}{
			{
				name: "EID verify error",
				setupMock: func(csp *mock.BCCSP, sm *mock.SchemaManager) {
					sm.EidNymAuditOptsReturns(&idemix.EidNymAuditOpts{EidIndex: 2, EnrollmentID: "enrollment-id"}, nil)
					csp.VerifyReturns(false, errors.New("verify failed"))
				},
				setupAuditInfo: func(ai *AuditInfo) {
					ai.EidNymAuditData = &idemix.AttrNymAuditData{}
				},
				expectedError: "error while verifying the nym eid",
			},
			{
				name: "EID verify returns false",
				setupMock: func(csp *mock.BCCSP, sm *mock.SchemaManager) {
					sm.EidNymAuditOptsReturns(&idemix.EidNymAuditOpts{EidIndex: 2, EnrollmentID: "enrollment-id"}, nil)
					csp.VerifyReturns(false, nil)
				},
				setupAuditInfo: func(ai *AuditInfo) {
					ai.EidNymAuditData = &idemix.AttrNymAuditData{}
				},
				expectedError: "invalid nym rh",
			},
			{
				name: "RH verify error",
				setupMock: func(csp *mock.BCCSP, sm *mock.SchemaManager) {
					sm.EidNymAuditOptsReturns(&idemix.EidNymAuditOpts{EidIndex: 2, EnrollmentID: "enrollment-id"}, nil)
					sm.RhNymAuditOptsReturns(&idemix.RhNymAuditOpts{RhIndex: 3, RevocationHandle: "revocation-handle"}, nil)
					csp.VerifyReturnsOnCall(0, true, nil)
					csp.VerifyReturnsOnCall(1, false, errors.New("rh verify failed"))
				},
				setupAuditInfo: func(ai *AuditInfo) {
					ai.EidNymAuditData = &idemix.AttrNymAuditData{}
					ai.RhNymAuditData = &idemix.AttrNymAuditData{}
				},
				expectedError: "error while verifying the nym rh",
			},
			{
				name: "RH verify returns false",
				setupMock: func(csp *mock.BCCSP, sm *mock.SchemaManager) {
					sm.EidNymAuditOptsReturns(&idemix.EidNymAuditOpts{EidIndex: 2, EnrollmentID: "enrollment-id"}, nil)
					sm.RhNymAuditOptsReturns(&idemix.RhNymAuditOpts{RhIndex: 3, RevocationHandle: "revocation-handle"}, nil)
					csp.VerifyReturnsOnCall(0, true, nil)
					csp.VerifyReturnsOnCall(1, false, nil)
				},
				setupAuditInfo: func(ai *AuditInfo) {
					ai.EidNymAuditData = &idemix.AttrNymAuditData{}
					ai.RhNymAuditData = &idemix.AttrNymAuditData{}
				},
				expectedError: "invalid nym eid",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockCsp := &mock.BCCSP{}
				mockSchemaManager := &mock.SchemaManager{}
				tt.setupMock(mockCsp, mockSchemaManager)

				auditInfo := createAuditInfo()
				auditInfo.Csp = mockCsp
				auditInfo.IssuerPublicKey = &mock.Key{}
				auditInfo.SchemaManager = mockSchemaManager
				tt.setupAuditInfo(auditInfo)

				err := auditInfo.Match(context.Background(), createIdentity())
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			})
		}
	})

	t.Run("Successful match", func(t *testing.T) {
		mockCsp := &mock.BCCSP{}
		mockSchemaManager := &mock.SchemaManager{}

		mockSchemaManager.EidNymAuditOptsReturns(&idemix.EidNymAuditOpts{EidIndex: 2, EnrollmentID: "enrollment-id"}, nil)
		mockSchemaManager.RhNymAuditOptsReturns(&idemix.RhNymAuditOpts{RhIndex: 3, RevocationHandle: "revocation-handle"}, nil)
		mockCsp.VerifyReturns(true, nil)

		auditInfo := createAuditInfo()
		auditInfo.Csp = mockCsp
		auditInfo.IssuerPublicKey = &mock.Key{}
		auditInfo.SchemaManager = mockSchemaManager
		auditInfo.EidNymAuditData = &idemix.AttrNymAuditData{}
		auditInfo.RhNymAuditData = &idemix.AttrNymAuditData{}

		err := auditInfo.Match(context.Background(), createIdentity())
		assert.NoError(t, err)
		assert.Equal(t, 2, mockCsp.VerifyCallCount())
	})
}
