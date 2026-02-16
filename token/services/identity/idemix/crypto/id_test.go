/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"testing"

	"github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	idemixmock "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/mock"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/kvs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIdentity(t *testing.T) {
	testNewIdentity(t, "../testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
	testNewIdentity(t, "../testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY)
}

func testNewIdentity(t *testing.T, configPath string, curveID math.CurveID) {
	t.Helper()

	// Setup
	backend, err := kvs2.NewInMemory()
	require.NoError(t, err)

	config, err := NewConfig(configPath)
	require.NoError(t, err)

	keyStore, err := NewKeyStore(curveID, kvs2.Keystore(backend))
	require.NoError(t, err)

	cryptoProvider, err := NewBCCSP(keyStore, curveID)
	require.NoError(t, err)

	issuerPublicKey, err := cryptoProvider.KeyImport(
		config.Ipk,
		&types.IdemixIssuerPublicKeyImportOpts{
			Temporary: true,
		},
	)
	require.NoError(t, err)

	deserializer := &Deserializer{
		Name:            "test",
		Ipk:             config.Ipk,
		Csp:             cryptoProvider,
		IssuerPublicKey: issuerPublicKey,
		Epoch:           0,
		VerType:         types.ExpectEidNymRhNym,
		Schema:          "test-schema",
	}

	// Import user key and derive nym key
	userKey, err := cryptoProvider.KeyImport(
		config.Signer.Sk,
		&types.IdemixUserSecretKeyImportOpts{Temporary: true},
	)
	require.NoError(t, err)

	nymKey, err := cryptoProvider.KeyDeriv(
		userKey,
		&types.IdemixNymKeyDerivationOpts{
			Temporary: true,
			IssuerPK:  issuerPublicKey,
		},
	)
	require.NoError(t, err)

	// Test NewIdentity
	identity := NewIdentity(
		deserializer,
		nymKey,
		[]byte("fake-proof"),
		types.ExpectEidNymRhNym,
		nil, // SchemaManager not needed for construction
		"test-schema",
	)
	assert.NotNil(t, identity)
	assert.Equal(t, nymKey, identity.NymPublicKey)
	assert.Equal(t, deserializer, identity.Idemix)
	assert.Equal(t, []byte("fake-proof"), identity.AssociationProof)
	assert.Equal(t, types.ExpectEidNymRhNym, identity.VerificationType)
	assert.Equal(t, "test-schema", identity.Schema)
}

func TestIdentity_Serialize(t *testing.T) {
	testIdentitySerialize(t, "../testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
	testIdentitySerialize(t, "../testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY)
}

func testIdentitySerialize(t *testing.T, configPath string, curveID math.CurveID) {
	t.Helper()

	// Setup
	backend, err := kvs2.NewInMemory()
	require.NoError(t, err)

	config, err := NewConfig(configPath)
	require.NoError(t, err)

	keyStore, err := NewKeyStore(curveID, kvs2.Keystore(backend))
	require.NoError(t, err)

	cryptoProvider, err := NewBCCSP(keyStore, curveID)
	require.NoError(t, err)

	issuerPublicKey, err := cryptoProvider.KeyImport(
		config.Ipk,
		&types.IdemixIssuerPublicKeyImportOpts{
			Temporary: true,
		},
	)
	require.NoError(t, err)

	deserializer := &Deserializer{
		Name:            "test",
		Ipk:             config.Ipk,
		Csp:             cryptoProvider,
		IssuerPublicKey: issuerPublicKey,
		Epoch:           0,
		VerType:         types.ExpectEidNymRhNym,
		Schema:          "test-schema",
	}

	// Import user key and derive nym key
	userKey, err := cryptoProvider.KeyImport(
		config.Signer.Sk,
		&types.IdemixUserSecretKeyImportOpts{Temporary: true},
	)
	require.NoError(t, err)

	nymKey, err := cryptoProvider.KeyDeriv(
		userKey,
		&types.IdemixNymKeyDerivationOpts{
			Temporary: true,
			IssuerPK:  issuerPublicKey,
		},
	)
	require.NoError(t, err)

	identity := NewIdentity(
		deserializer,
		nymKey,
		[]byte("test-proof"),
		types.ExpectEidNymRhNym,
		nil,
		"test-schema",
	)

	// Test Serialize
	serialized, err := identity.Serialize()
	require.NoError(t, err)
	assert.NotNil(t, serialized)

	// Verify we can deserialize it back
	deserialized := &SerializedIdemixIdentity{}
	err = proto.Unmarshal(serialized, deserialized)
	require.NoError(t, err)
	assert.NotNil(t, deserialized.NymPublicKey)
	assert.Equal(t, []byte("test-proof"), deserialized.Proof)
}

func TestIdentity_Constants(t *testing.T) {
	// Test that constants are defined correctly
	assert.Equal(t, 2, EIDIndex)
	assert.Equal(t, 3, RHIndex)
	assert.Equal(t, "SignerConfigFull", SignerConfigFull)
}

func TestSigningIdentity_Sign(t *testing.T) {
	testSigningIdentitySign(t, "../testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
	testSigningIdentitySign(t, "../testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY)
}

func testSigningIdentitySign(t *testing.T, configPath string, curveID math.CurveID) {
	t.Helper()

	// Setup
	backend, err := kvs2.NewInMemory()
	require.NoError(t, err)

	config, err := NewConfig(configPath)
	require.NoError(t, err)

	keyStore, err := NewKeyStore(curveID, kvs2.Keystore(backend))
	require.NoError(t, err)

	cryptoProvider, err := NewBCCSP(keyStore, curveID)
	require.NoError(t, err)

	issuerPublicKey, err := cryptoProvider.KeyImport(
		config.Ipk,
		&types.IdemixIssuerPublicKeyImportOpts{
			Temporary: true,
		},
	)
	require.NoError(t, err)

	deserializer := &Deserializer{
		Name:            "test",
		Ipk:             config.Ipk,
		Csp:             cryptoProvider,
		IssuerPublicKey: issuerPublicKey,
		Epoch:           0,
		VerType:         types.ExpectEidNymRhNym,
		Schema:          "test-schema",
	}

	// Import user key and derive nym key (temporary for testing)
	userKey, err := cryptoProvider.KeyImport(
		config.Signer.Sk,
		&types.IdemixUserSecretKeyImportOpts{Temporary: true},
	)
	require.NoError(t, err)

	nymKey, err := cryptoProvider.KeyDeriv(
		userKey,
		&types.IdemixNymKeyDerivationOpts{
			Temporary: true,
			IssuerPK:  issuerPublicKey,
		},
	)
	require.NoError(t, err)

	identity := NewIdentity(
		deserializer,
		nymKey,
		[]byte("test-proof"),
		types.ExpectEidNymRhNym,
		nil,
		"test-schema",
	)

	// Test Sign with invalid nym key SKI - should fail when trying to get the key
	invalidSigning := &SigningIdentity{
		Identity:     identity,
		CSP:          cryptoProvider,
		EnrollmentId: "test-user",
		NymKeySKI:    []byte("invalid-ski-that-does-not-exist"),
		UserKeySKI:   userKey.SKI(),
	}
	_, err = invalidSigning.Sign([]byte("test message"))
	assert.Error(t, err, "Sign should fail with invalid nym key SKI")

	// Test Sign with invalid user key SKI - should fail when trying to get the key
	invalidSigning2 := &SigningIdentity{
		Identity:     identity,
		CSP:          cryptoProvider,
		EnrollmentId: "test-user",
		NymKeySKI:    nymKey.SKI(),
		UserKeySKI:   []byte("invalid-ski-that-does-not-exist"),
	}
	_, err = invalidSigning2.Sign([]byte("test message"))
	assert.Error(t, err, "Sign should fail with invalid user key SKI")
}

func TestIdentity_Verify(t *testing.T) {
	// Use mock BCCSP to test Verify function
	mockBCCSP := &idemixmock.BCCSP{}
	mockIPK := &idemixmock.Key{}
	mockNymPK := &idemixmock.Key{}

	deserializer := &Deserializer{
		Name:            "test",
		Ipk:             []byte("test-ipk"),
		Csp:             mockBCCSP,
		IssuerPublicKey: mockIPK,
	}

	identity := &Identity{
		Idemix:       deserializer,
		NymPublicKey: mockNymPK,
	}

	msg := []byte("test message")
	sig := []byte("test signature")

	t.Run("Verify success", func(t *testing.T) {
		mockBCCSP.VerifyReturns(true, nil)

		err := identity.Verify(msg, sig)
		assert.NoError(t, err)

		// Verify the mock was called with correct parameters
		assert.Equal(t, 1, mockBCCSP.VerifyCallCount())
		key, signature, message, opts := mockBCCSP.VerifyArgsForCall(0)
		assert.Equal(t, mockNymPK, key)
		assert.Equal(t, sig, signature)
		assert.Equal(t, msg, message)
		assert.NotNil(t, opts)
	})

	t.Run("Verify failure", func(t *testing.T) {
		mockBCCSP.VerifyReturns(false, errors.New("verification failed"))

		err := identity.Verify(msg, sig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "verification failed")
	})
}

func TestIdentity_Serialize_ErrorPaths(t *testing.T) {
	// Test error path when NymPublicKey.Bytes() fails
	mockBCCSP := &idemixmock.BCCSP{}
	mockIPK := &idemixmock.Key{}
	mockNymPK := &idemixmock.Key{}

	deserializer := &Deserializer{
		Name:            "test",
		Ipk:             []byte("test-ipk"),
		Csp:             mockBCCSP,
		IssuerPublicKey: mockIPK,
	}

	identity := &Identity{
		Idemix:           deserializer,
		NymPublicKey:     mockNymPK,
		AssociationProof: []byte("proof"),
		Schema:           "test-schema",
	}

	t.Run("NymPublicKey.Bytes() fails", func(t *testing.T) {
		mockNymPK.BytesReturns(nil, errors.New("bytes error"))

		_, err := identity.Serialize()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not serialize nym")
	})
}

func TestIdentity_verifyProof(t *testing.T) {
	mockBCCSP := &idemixmock.BCCSP{}
	mockIPK := &idemixmock.Key{}
	mockNymPK := &idemixmock.Key{}

	t.Run("verifyProof with NymEID", func(t *testing.T) {
		mockRevPK := &idemixmock.Key{}
		deserializer := &Deserializer{
			Name:            "test",
			Ipk:             []byte("test-ipk"),
			Csp:             mockBCCSP,
			IssuerPublicKey: mockIPK,
			NymEID:          []byte("nym-eid"),
			RhNym:           []byte("rh-nym"),
			RevocationPK:    mockRevPK,
			Epoch:           1,
		}

		identity := &Identity{
			Idemix:           deserializer,
			NymPublicKey:     mockNymPK,
			AssociationProof: []byte("proof"),
			VerificationType: types.ExpectEidNymRhNym,
		}

		mockBCCSP.VerifyReturns(true, nil)

		err := identity.verifyProof()
		assert.NoError(t, err)

		// Verify the mock was called
		assert.Equal(t, 1, mockBCCSP.VerifyCallCount())
	})

	t.Run("verifyProof without NymEID", func(t *testing.T) {
		mockRevPK := &idemixmock.Key{}
		deserializer := &Deserializer{
			Name:            "test",
			Ipk:             []byte("test-ipk"),
			Csp:             mockBCCSP,
			IssuerPublicKey: mockIPK,
			NymEID:          nil, // No NymEID
			RevocationPK:    mockRevPK,
			Epoch:           1,
		}

		identity := &Identity{
			Idemix:           deserializer,
			NymPublicKey:     mockNymPK,
			AssociationProof: []byte("proof"),
			VerificationType: types.ExpectStandard,
		}

		mockBCCSP.VerifyReturns(true, nil)

		err := identity.verifyProof()
		assert.NoError(t, err)
	})

	t.Run("verifyProof returns error", func(t *testing.T) {
		deserializer := &Deserializer{
			Name:            "test",
			Ipk:             []byte("test-ipk"),
			Csp:             mockBCCSP,
			IssuerPublicKey: mockIPK,
		}

		identity := &Identity{
			Idemix:           deserializer,
			NymPublicKey:     mockNymPK,
			AssociationProof: []byte("proof"),
		}

		mockBCCSP.VerifyReturns(false, errors.New("verify error"))

		err := identity.verifyProof()
		assert.Error(t, err)
	})

	t.Run("verifyProof returns false without error", func(t *testing.T) {
		deserializer := &Deserializer{
			Name:            "test",
			Ipk:             []byte("test-ipk"),
			Csp:             mockBCCSP,
			IssuerPublicKey: mockIPK,
		}

		identity := &Identity{
			Idemix:           deserializer,
			NymPublicKey:     mockNymPK,
			AssociationProof: []byte("proof"),
		}

		// This should trigger the error path at line 113-114
		mockBCCSP.VerifyReturns(false, nil)

		err := identity.verifyProof()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected condition")
	})
}

func TestSigningIdentity_Sign_ErrorPaths(t *testing.T) {
	t.Run("Sign fails at GetKey for nym key", func(t *testing.T) {
		// Create fresh mocks for this test
		mockCSP := &idemixmock.BCCSP{}
		mockIPK := &idemixmock.Key{}
		mockNymPK := &idemixmock.Key{}

		deserializer := &Deserializer{
			Name:            "test",
			Ipk:             []byte("test-ipk"),
			Csp:             &idemixmock.BCCSP{},
			IssuerPublicKey: mockIPK,
		}

		identity := &Identity{
			Idemix:       deserializer,
			NymPublicKey: mockNymPK,
		}

		signingIdentity := &SigningIdentity{
			Identity:     identity,
			CSP:          mockCSP,
			EnrollmentId: "test-user",
			NymKeySKI:    []byte("nym-ski"),
			UserKeySKI:   []byte("user-ski"),
		}

		// Mock GetKey to fail for nym key
		mockCSP.GetKeyReturns(nil, errors.New("nym key not found"))

		_, err := signingIdentity.Sign([]byte("test message"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot find nym secret key")
	})

	t.Run("Sign fails at GetKey for user key", func(t *testing.T) {
		// Create fresh mocks for this test
		mockCSP := &idemixmock.BCCSP{}
		mockIPK := &idemixmock.Key{}
		mockNymPK := &idemixmock.Key{}
		mockNymSK := &idemixmock.Key{}

		deserializer := &Deserializer{
			Name:            "test",
			Ipk:             []byte("test-ipk"),
			Csp:             &idemixmock.BCCSP{},
			IssuerPublicKey: mockIPK,
		}

		identity := &Identity{
			Idemix:       deserializer,
			NymPublicKey: mockNymPK,
		}

		signingIdentity := &SigningIdentity{
			Identity:     identity,
			CSP:          mockCSP,
			EnrollmentId: "test-user",
			NymKeySKI:    []byte("nym-ski"),
			UserKeySKI:   []byte("user-ski"),
		}

		// Mock GetKey to succeed for nym key, fail for user key
		mockCSP.GetKeyReturnsOnCall(0, mockNymSK, nil)
		mockCSP.GetKeyReturnsOnCall(1, nil, errors.New("user key not found"))

		_, err := signingIdentity.Sign([]byte("test message"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve user key")
	})

	t.Run("Sign fails at CSP.Sign", func(t *testing.T) {
		// Create fresh mocks for this test
		mockBCCSP := &idemixmock.BCCSP{}
		mockIPK := &idemixmock.Key{}
		mockNymPK := &idemixmock.Key{}
		mockNymSK := &idemixmock.Key{}
		mockUserSK := &idemixmock.Key{}

		deserializer := &Deserializer{
			Name:            "test",
			Ipk:             []byte("test-ipk"),
			Csp:             mockBCCSP,
			IssuerPublicKey: mockIPK,
		}

		identity := &Identity{
			Idemix:       deserializer,
			NymPublicKey: mockNymPK,
		}

		signingIdentity := &SigningIdentity{
			Identity:     identity,
			CSP:          mockBCCSP,
			EnrollmentId: "test-user",
			NymKeySKI:    []byte("nym-ski"),
			UserKeySKI:   []byte("user-ski"),
		}

		// Mock GetKey to return keys
		mockBCCSP.GetKeyReturnsOnCall(0, mockNymSK, nil)  // First call for nym key
		mockBCCSP.GetKeyReturnsOnCall(1, mockUserSK, nil) // Second call for user key

		// Mock Sign to fail
		mockBCCSP.SignReturns(nil, errors.New("sign error"))

		_, err := signingIdentity.Sign([]byte("test message"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sign error")
	})

	t.Run("Sign succeeds", func(t *testing.T) {
		// Create fresh mocks for this test
		mockCSP := &idemixmock.BCCSP{}        // For GetKey calls
		mockIdemixCSP := &idemixmock.BCCSP{}  // For Sign call
		mockIPK := &idemixmock.Key{}
		mockNymPK := &idemixmock.Key{}
		mockNymSK := &idemixmock.Key{}
		mockUserSK := &idemixmock.Key{}

		deserializer := &Deserializer{
			Name:            "test",
			Ipk:             []byte("test-ipk"),
			Csp:             mockIdemixCSP,  // This is used for Sign
			IssuerPublicKey: mockIPK,
		}

		identity := &Identity{
			Idemix:       deserializer,
			NymPublicKey: mockNymPK,
		}

		signingIdentity := &SigningIdentity{
			Identity:     identity,
			CSP:          mockCSP,  // This is used for GetKey
			EnrollmentId: "test-user",
			NymKeySKI:    []byte("nym-ski"),
			UserKeySKI:   []byte("user-ski"),
		}

		// Mock GetKey to return keys (on mockCSP)
		mockCSP.GetKeyReturnsOnCall(0, mockNymSK, nil)
		mockCSP.GetKeyReturnsOnCall(1, mockUserSK, nil)

		// Mock Sign to succeed (on mockIdemixCSP)
		expectedSig := []byte("signature")
		mockIdemixCSP.SignReturns(expectedSig, nil)

		sig, err := signingIdentity.Sign([]byte("test message"))
		assert.NoError(t, err)
		assert.Equal(t, expectedSig, sig)
	})
}

func TestNymSignatureVerifier_Verify(t *testing.T) {
	testNymSignatureVerifierVerify(t, "../testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
	testNymSignatureVerifierVerify(t, "../testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY)
}

func testNymSignatureVerifierVerify(t *testing.T, configPath string, curveID math.CurveID) {
	t.Helper()

	// Setup
	backend, err := kvs2.NewInMemory()
	require.NoError(t, err)

	config, err := NewConfig(configPath)
	require.NoError(t, err)

	keyStore, err := NewKeyStore(curveID, kvs2.Keystore(backend))
	require.NoError(t, err)

	cryptoProvider, err := NewBCCSP(keyStore, curveID)
	require.NoError(t, err)

	issuerPublicKey, err := cryptoProvider.KeyImport(
		config.Ipk,
		&types.IdemixIssuerPublicKeyImportOpts{
			Temporary: true,
		},
	)
	require.NoError(t, err)

	// Import user key and derive nym key
	userKey, err := cryptoProvider.KeyImport(
		config.Signer.Sk,
		&types.IdemixUserSecretKeyImportOpts{Temporary: false},
	)
	require.NoError(t, err)

	nymKey, err := cryptoProvider.KeyDeriv(
		userKey,
		&types.IdemixNymKeyDerivationOpts{
			Temporary: false,
			IssuerPK:  issuerPublicKey,
		},
	)
	require.NoError(t, err)

	// Get the public part
	nymPublicKey, err := nymKey.PublicKey()
	require.NoError(t, err)

	verifier := &NymSignatureVerifier{
		CSP:    cryptoProvider,
		IPK:    issuerPublicKey,
		NymPK:  nymPublicKey,
		Schema: "test-schema",
	}

	// Test with invalid signature
	err = verifier.Verify([]byte("message"), []byte("invalid-signature"))
	assert.Error(t, err)
}
