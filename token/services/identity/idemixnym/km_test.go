/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemixnym

import (
	"encoding/json"
	"testing"

	"github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemixnym/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemixnym/nym"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/kvs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKeyManager(t *testing.T) {
	testNewKeyManager(t, "../idemix/testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
	testNewKeyManager(t, "../idemix/testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY)
}

func testNewKeyManager(t *testing.T, configPath string, curveID math.CurveID) {
	t.Helper()
	// prepare
	kvs, err := kvs2.NewInMemory()
	require.NoError(t, err)
	config, err := crypto.NewConfig(configPath)
	require.NoError(t, err)
	keyStore, err := crypto.NewKeyStore(curveID, kvs2.Keystore(kvs))
	require.NoError(t, err)
	cryptoProvider, err := crypto.NewBCCSP(keyStore, curveID)
	require.NoError(t, err)

	// create backend key manager
	backendKM, err := idemix.NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	require.NoError(t, err)
	assert.NotNil(t, backendKM)

	// create mock identity store service
	identityStoreService := &mock.IdentityStoreService{}

	// create idemixnym key manager
	keyManager := NewKeyManager(backendKM, identityStoreService)
	require.NotNil(t, keyManager)
	assert.False(t, keyManager.IsRemote())
	assert.True(t, keyManager.Anonymous())
	assert.Equal(t, "alice", keyManager.EnrollmentID())
	assert.Equal(t, IdentityType, keyManager.IdentityType())
}

func TestKeyManagerIdentity(t *testing.T) {
	testKeyManagerIdentity(t, "../idemix/testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
	testKeyManagerIdentity(t, "../idemix/testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY)
}

func testKeyManagerIdentity(t *testing.T, configPath string, curveID math.CurveID) {
	t.Helper()
	// prepare
	kvs, err := kvs2.NewInMemory()
	require.NoError(t, err)
	config, err := crypto.NewConfig(configPath)
	require.NoError(t, err)
	keyStore, err := crypto.NewKeyStore(curveID, kvs2.Keystore(kvs))
	require.NoError(t, err)
	cryptoProvider, err := crypto.NewBCCSP(keyStore, curveID)
	require.NoError(t, err)

	// create backend key manager
	backendKM, err := idemix.NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	require.NoError(t, err)

	// create mock identity store service
	identityStoreService := &mock.IdentityStoreService{}

	// create idemixnym key manager
	keyManager := NewKeyManager(backendKM, identityStoreService)
	require.NotNil(t, keyManager)

	// get an identity
	identityDescriptor, err := keyManager.Identity(t.Context(), nil)
	require.NoError(t, err)
	assert.NotNil(t, identityDescriptor)
	assert.NotNil(t, identityDescriptor.Identity)
	assert.NotNil(t, identityDescriptor.AuditInfo)
	assert.NotNil(t, identityDescriptor.Signer)
	assert.False(t, identityDescriptor.Ephemeral)

	// verify the audit info contains the nym
	auditInfo := &nym.AuditInfo{}
	err = json.Unmarshal(identityDescriptor.AuditInfo, auditInfo)
	require.NoError(t, err)
	assert.NotNil(t, auditInfo.AuditInfo)
	assert.NotNil(t, auditInfo.IdemixSignature)

	// verify the identity is the nym (commitment to EID)
	assert.NotNil(t, auditInfo.EidNymAuditData)
	assert.NotNil(t, auditInfo.EidNymAuditData.Nym)
	assert.Equal(t, auditInfo.EidNymAuditData.Nym.Bytes(), []byte(identityDescriptor.Identity))
}

func TestDeserializeVerifier(t *testing.T) {
	testDeserializeVerifier(t, "../idemix/testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
	testDeserializeVerifier(t, "../idemix/testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY)
}

func testDeserializeVerifier(t *testing.T, configPath string, curveID math.CurveID) {
	t.Helper()
	// prepare
	kvs, err := kvs2.NewInMemory()
	require.NoError(t, err)
	config, err := crypto.NewConfig(configPath)
	require.NoError(t, err)
	keyStore, err := crypto.NewKeyStore(curveID, kvs2.Keystore(kvs))
	require.NoError(t, err)
	cryptoProvider, err := crypto.NewBCCSP(keyStore, curveID)
	require.NoError(t, err)

	// create backend key manager
	backendKM, err := idemix.NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	require.NoError(t, err)

	// create mock identity store service
	identityStoreService := &mock.IdentityStoreService{}

	// create idemixnym key manager
	keyManager := NewKeyManager(backendKM, identityStoreService)
	require.NotNil(t, keyManager)

	// get an identity
	identityDescriptor, err := keyManager.Identity(t.Context(), nil)
	require.NoError(t, err)

	// deserialize verifier
	verifier, err := keyManager.DeserializeVerifier(t.Context(), identityDescriptor.Identity)
	require.NoError(t, err)
	assert.NotNil(t, verifier)

	// sign and verify
	msg := []byte("test message")
	sigma, err := identityDescriptor.Signer.Sign(msg)
	require.NoError(t, err)
	err = identityDescriptor.Verifier.Verify(msg, sigma)
	require.NoError(t, err)
}

func TestDeserializeSigner(t *testing.T) {
	testDeserializeSigner(t, "../idemix/testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
	testDeserializeSigner(t, "../idemix/testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY)
}

func testDeserializeSigner(t *testing.T, configPath string, curveID math.CurveID) {
	t.Helper()
	// prepare
	kvs, err := kvs2.NewInMemory()
	require.NoError(t, err)
	config, err := crypto.NewConfig(configPath)
	require.NoError(t, err)
	keyStore, err := crypto.NewKeyStore(curveID, kvs2.Keystore(kvs))
	require.NoError(t, err)
	cryptoProvider, err := crypto.NewBCCSP(keyStore, curveID)
	require.NoError(t, err)

	// create backend key manager
	backendKM, err := idemix.NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	require.NoError(t, err)

	// create mock identity store service
	identityStoreService := &mock.IdentityStoreService{}

	// create idemixnym key manager
	keyManager := NewKeyManager(backendKM, identityStoreService)
	require.NotNil(t, keyManager)

	// get an identity
	identityDescriptor, err := keyManager.Identity(t.Context(), nil)
	require.NoError(t, err)

	// setup mock to return the signer info
	identityStoreService.GetSignerInfoReturns(identityDescriptor.SignerInfo, nil)

	// deserialize signer
	signer, err := keyManager.DeserializeSigner(t.Context(), identityDescriptor.Identity)
	require.NoError(t, err)
	assert.NotNil(t, signer)

	// verify the mock was called
	assert.Equal(t, 1, identityStoreService.GetSignerInfoCallCount())

	// sign a message
	msg := []byte("test message")
	sigma, err := signer.Sign(msg)
	require.NoError(t, err)
	assert.NotNil(t, sigma)

	// deserialize verifier and verify
	verifier, err := keyManager.DeserializeVerifier(t.Context(), identityDescriptor.Identity)
	require.NoError(t, err)
	err = verifier.Verify(msg, sigma)
	require.NoError(t, err)
}

func TestKeyManagerErrorPaths(t *testing.T) {
	testKeyManagerErrorPaths(t, "../idemix/testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
	testKeyManagerErrorPaths(t, "../idemix/testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY)
}

func testKeyManagerErrorPaths(t *testing.T, configPath string, curveID math.CurveID) {
	t.Helper()
	kvs, err := kvs2.NewInMemory()
	require.NoError(t, err)
	config, err := crypto.NewConfig(configPath)
	require.NoError(t, err)
	keyStore, err := crypto.NewKeyStore(curveID, kvs2.Keystore(kvs))
	require.NoError(t, err)
	cryptoProvider, err := crypto.NewBCCSP(keyStore, curveID)
	require.NoError(t, err)

	// create backend key manager
	backendKM, err := idemix.NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	require.NoError(t, err)

	// create mock identity store service
	identityStoreService := &mock.IdentityStoreService{}

	// create idemixnym key manager
	keyManager := NewKeyManager(backendKM, identityStoreService)
	require.NotNil(t, keyManager)

	// test DeserializeSigner with error from identity store
	identityStoreService.GetSignerInfoReturns(nil, errors.New("identity store error"))
	_, err = keyManager.DeserializeSigner(t.Context(), []byte("test-id"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve signer info")

	// test DeserializeSigner with invalid signer info
	identityStoreService.GetSignerInfoReturns([]byte("invalid json"), nil)
	_, err = keyManager.DeserializeSigner(t.Context(), []byte("test-id"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to deserialize audit info")

	// test DeserializeSigner with valid JSON but invalid audit info structure
	invalidAuditInfo := &nym.AuditInfo{}
	invalidAuditInfoRaw, err := json.Marshal(invalidAuditInfo)
	require.NoError(t, err)
	identityStoreService.GetSignerInfoReturns(invalidAuditInfoRaw, nil)
	_, err = keyManager.DeserializeSigner(t.Context(), []byte("test-id"))
	require.Error(t, err)
}

func TestKeyManagerMethods(t *testing.T) {
	testKeyManagerMethods(t, "../idemix/testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
	testKeyManagerMethods(t, "../idemix/testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY)
}

func testKeyManagerMethods(t *testing.T, configPath string, curveID math.CurveID) {
	t.Helper()
	kvs, err := kvs2.NewInMemory()
	require.NoError(t, err)
	config, err := crypto.NewConfig(configPath)
	require.NoError(t, err)
	keyStore, err := crypto.NewKeyStore(curveID, kvs2.Keystore(kvs))
	require.NoError(t, err)
	cryptoProvider, err := crypto.NewBCCSP(keyStore, curveID)
	require.NoError(t, err)

	// create backend key manager
	backendKM, err := idemix.NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	require.NoError(t, err)

	// create mock identity store service
	identityStoreService := &mock.IdentityStoreService{}

	// create idemixnym key manager
	keyManager := NewKeyManager(backendKM, identityStoreService)
	require.NotNil(t, keyManager)

	// test EnrollmentID
	assert.Equal(t, "alice", keyManager.EnrollmentID())

	// test IsRemote
	assert.False(t, keyManager.IsRemote())

	// test Anonymous
	assert.True(t, keyManager.Anonymous())

	// test IdentityType
	assert.Equal(t, IdentityType, keyManager.IdentityType())
}

func TestKeyManagerIdentityWithAuditInfo(t *testing.T) {
	testKeyManagerIdentityWithAuditInfo(t, "../idemix/testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
	testKeyManagerIdentityWithAuditInfo(t, "../idemix/testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY)
}

func testKeyManagerIdentityWithAuditInfo(t *testing.T, configPath string, curveID math.CurveID) {
	t.Helper()
	kvs, err := kvs2.NewInMemory()
	require.NoError(t, err)
	config, err := crypto.NewConfig(configPath)
	require.NoError(t, err)
	keyStore, err := crypto.NewKeyStore(curveID, kvs2.Keystore(kvs))
	require.NoError(t, err)
	cryptoProvider, err := crypto.NewBCCSP(keyStore, curveID)
	require.NoError(t, err)

	// create backend key manager
	backendKM, err := idemix.NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	require.NoError(t, err)

	// create mock identity store service
	identityStoreService := &mock.IdentityStoreService{}

	// create idemixnym key manager
	keyManager := NewKeyManager(backendKM, identityStoreService)
	require.NotNil(t, keyManager)

	// get first identity
	id1, err := keyManager.Identity(t.Context(), nil)
	require.NoError(t, err)

	// get second identity with same audit info (should reuse the nym)
	id2, err := keyManager.Identity(t.Context(), id1.AuditInfo)
	require.NoError(t, err)

	// verify both identities are the same (same nym)
	assert.Equal(t, id1.Identity, id2.Identity)

	// verify both have the same audit info
	ai1, err := keyManager.DeserializeAuditInfo(t.Context(), id1.AuditInfo)
	require.NoError(t, err)
	ai2, err := keyManager.DeserializeAuditInfo(t.Context(), id2.AuditInfo)
	require.NoError(t, err)
	assert.Equal(t, ai1.AuditInfo, ai2.AuditInfo)
	assert.NotEqual(t, ai1.IdemixSignature, ai2.IdemixSignature)
}

// Made with Bob
