/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"context"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	idemix2 "github.com/IBM/idemix"
	"github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	crypto2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKeyManager(t *testing.T) {
	testNewKeyManager(t, "./testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
	testNewKeyManager(t, "./testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY)
}

func testNewKeyManager(t *testing.T, configPath string, curveID math.CurveID) {
	t.Helper()
	// prepare
	kvs, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	config, err := crypto2.NewConfig(configPath)
	assert.NoError(t, err)
	tracker := kvs2.NewTrackedMemoryFrom(kvs)
	keyStore, err := crypto2.NewKeyStore(curveID, tracker)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, curveID)
	assert.NoError(t, err)

	// check that version is enforced
	config.Version = 0
	_, err = NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	assert.Error(t, err)
	assert.EqualError(t, err, "unsupported protocol version [0]")
	config.Version = crypto2.ProtobufProtocolVersionV1

	// new key manager loaded from file
	assert.Empty(t, config.Signer.Ski)
	keyManager, err := NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager)
	assert.False(t, keyManager.IsRemote())
	assert.True(t, keyManager.Anonymous())
	assert.Equal(t, "alice", keyManager.EnrollmentID())
	assert.Equal(t, IdentityType, keyManager.IdentityType())
	assert.Equal(t, fmt.Sprintf("Idemix KeyManager [%s]", utils.Hashable(keyManager.Ipk).String()), keyManager.String())
	assert.Equal(t, 1, tracker.PutCounter)
	assert.Equal(t, 0, tracker.GetCounter)

	// the config has been updated, load a new key manager
	assert.NotEmpty(t, config.Signer.Ski)
	keyManager, err = NewKeyManager(config, types.Standard, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager)
	assert.False(t, keyManager.IsRemote())
	assert.True(t, keyManager.Anonymous())
	assert.Equal(t, "alice", keyManager.EnrollmentID())
	assert.Equal(t, IdentityType, keyManager.IdentityType())
	assert.Equal(t, fmt.Sprintf("Idemix KeyManager [%s]", utils.Hashable(keyManager.Ipk).String()), keyManager.String())
	assert.Equal(t, 1, tracker.PutCounter) // this is still 1 because the key is loaded using the SKI
	assert.Equal(t, 1, tracker.GetCounter) // one get for the user key
	assert.Equal(t, tracker.GetHistory[0].Key, hex.EncodeToString(config.Signer.Ski))

	// load a new key manager again
	assert.NotEmpty(t, config.Signer.Ski)
	keyManager, err = NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager)
	assert.False(t, keyManager.IsRemote())
	assert.True(t, keyManager.Anonymous())
	assert.Equal(t, "alice", keyManager.EnrollmentID())
	assert.Equal(t, IdentityType, keyManager.IdentityType())
	assert.Equal(t, fmt.Sprintf("Idemix KeyManager [%s]", utils.Hashable(keyManager.Ipk).String()), keyManager.String())
	assert.Equal(t, 1, tracker.PutCounter) // this is still 1 because the key is loaded using the SKI
	assert.Equal(t, 2, tracker.GetCounter) // another get for the user key
	assert.Equal(t, tracker.GetHistory[1].Key, hex.EncodeToString(config.Signer.Ski))

	// invalid sig type
	_, err = NewKeyManager(config, -1, cryptoProvider)
	assert.Error(t, err)
	assert.EqualError(t, err, "unsupported signature type -1")

	assert.Equal(t, 1, tracker.PutCounter)
	assert.Equal(t, 3, tracker.GetCounter) // another get
	assert.Equal(t, tracker.GetHistory[2].Key, hex.EncodeToString(config.Signer.Ski))

	// no config
	_, err = NewKeyManager(nil, types.EidNymRhNym, cryptoProvider)
	assert.Error(t, err)
	assert.EqualError(t, err, "no idemix config provided")

	// no signer in config
	config.Signer = nil
	_, err = NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	assert.Error(t, err)
	assert.EqualError(t, err, "no signer information found")

	// nothing changed
	assert.Equal(t, 1, tracker.PutCounter)
	assert.Equal(t, 3, tracker.GetCounter)
}

func TestIdentityWithEidRhNymPolicy(t *testing.T) {
	testIdentityWithEidRhNymPolicy(t, "./testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL, false)
	testIdentityWithEidRhNymPolicy(t, "./testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY, true)
}

func testIdentityWithEidRhNymPolicy(t *testing.T, configPath string, curveID math.CurveID, aries bool) {
	t.Helper()
	// prepare
	registry := view.NewServiceProvider()
	kvs, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(kvs))
	storage := kvs2.NewIdentityStore(kvs, token.TMSID{Network: "pineapple"})
	identityProvider := identity.NewProvider(logging.MustGetLogger(), storage, deserializer.NewTypedSignerDeserializerMultiplex(), nil, nil)
	config, err := crypto2.NewConfig(configPath)
	assert.NoError(t, err)
	tracker := kvs2.NewTrackedMemoryFrom(kvs)
	keyStore, err := crypto2.NewKeyStore(curveID, tracker)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, curveID)
	assert.NoError(t, err)

	// init key manager
	// with invalid sig type
	_, err = NewKeyManager(config, -1, cryptoProvider)
	assert.Error(t, err)
	assert.EqualError(t, err, "unsupported signature type -1")
	// correctly
	keyManager, err := NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager)

	// get an identity and check it
	identityDescriptor, err := keyManager.Identity(t.Context(), nil)
	assert.NoError(t, err)
	id := identityDescriptor.Identity
	audit := identityDescriptor.AuditInfo
	require.NoError(t, identityProvider.RegisterSigner(t.Context(), id, identityDescriptor.Signer, identityDescriptor.Verifier, identityDescriptor.SignerInfo, false))
	assert.NotNil(t, id)
	assert.NotNil(t, audit)
	info, err := keyManager.Info(t.Context(), id, audit)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: [alice]"))

	// get another identity and compare the info
	identityDescriptor2, err := keyManager.Identity(t.Context(), audit)
	assert.NoError(t, err)
	id2 := identityDescriptor2.Identity
	audit2 := identityDescriptor2.AuditInfo
	assert.NotNil(t, id2)
	assert.NotNil(t, audit2)
	info2, err := keyManager.Info(t.Context(), id2, audit2)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info2, "Idemix: [alice]"))
	assert.Equal(t, audit, audit2)

	// deserialize the audit information
	auditInfo, err := keyManager.DeserializeAuditInfo(t.Context(), audit)
	assert.NoError(t, err)
	assert.NoError(t, auditInfo.Match(t.Context(), id))
	assert.NoError(t, auditInfo.Match(t.Context(), id2))
	auditInfo2, err := keyManager.DeserializeAuditInfo(t.Context(), audit2)
	assert.NoError(t, err)
	assert.NoError(t, auditInfo2.Match(t.Context(), id))
	assert.NoError(t, auditInfo2.Match(t.Context(), id2))

	assert.Equal(t, 3, tracker.GetCounter)

	// deserialize an invalid signer
	_, err = keyManager.DeserializeSigner(t.Context(), nil)
	assert.Error(t, err)
	_, err = keyManager.DeserializeSigner(t.Context(), []byte{})
	assert.Error(t, err)
	_, err = keyManager.DeserializeSigner(t.Context(), []byte{0, 1, 2})
	assert.Error(t, err)
	assert.Equal(t, 3, tracker.GetCounter)
	// deserialize a valid signer
	signer, err := keyManager.DeserializeSigner(t.Context(), id)
	assert.NoError(t, err)
	assert.Equal(t, 5, tracker.GetCounter) // this is due the call to Sign used to test if the signer belong to this key manager
	assert.Equal(t, hex.EncodeToString(keyManager.userKeySKI), tracker.GetHistory[4].Key)

	// deserialize an invalid verifier
	_, err = keyManager.DeserializeVerifier(t.Context(), nil)
	assert.Error(t, err)
	_, err = keyManager.DeserializeVerifier(t.Context(), []byte{})
	assert.Error(t, err)
	_, err = keyManager.DeserializeVerifier(t.Context(), []byte{0, 1, 2})
	assert.Error(t, err)
	// deserialize a valid verifier
	verifier, err := keyManager.DeserializeVerifier(t.Context(), id)
	assert.NoError(t, err)

	// sign and verify
	sigma, err := signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))
	assert.Equal(t, 7, tracker.GetCounter)
	assert.Equal(t, tracker.GetHistory[3].Key, tracker.GetHistory[5].Key)
	assert.Equal(t, tracker.GetHistory[3].Value, tracker.GetHistory[5].Value)
	assert.Equal(t, hex.EncodeToString(keyManager.userKeySKI), tracker.GetHistory[6].Key)
	assert.Equal(t, tracker.GetHistory[4].Value, tracker.GetHistory[6].Value)
}

func TestIdentityStandard(t *testing.T) {
	testIdentityStandard(t, "./testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL, false)
	testIdentityStandard(t, "./testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY, true)
}

func testIdentityStandard(t *testing.T, configPath string, curveID math.CurveID, aries bool) {
	t.Helper()
	registry := view.NewServiceProvider()

	kvs, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(kvs))

	config, err := crypto2.NewConfig(configPath)
	assert.NoError(t, err)

	keyStore, err := crypto2.NewKeyStore(curveID, kvs2.Keystore(kvs))
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, curveID)
	assert.NoError(t, err)
	p, err := NewKeyManager(config, types.Standard, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	identityDescriptor, err := p.Identity(t.Context(), nil)
	assert.NoError(t, err)
	id := identityDescriptor.Identity
	audit := identityDescriptor.AuditInfo
	assert.NotNil(t, id)
	assert.Nil(t, audit)

	signer, err := p.DeserializeSigner(t.Context(), id)
	assert.NoError(t, err)
	verifier, err := p.DeserializeVerifier(t.Context(), id)
	assert.NoError(t, err)

	sigma, err := signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))

	keyStore, err = crypto2.NewKeyStore(curveID, kvs2.Keystore(kvs))
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, curveID)
	assert.NoError(t, err)
	p, err = NewKeyManager(config, types.Standard, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	_, err = p.Identity(t.Context(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.Nil(t, audit)

	signer, err = p.DeserializeSigner(t.Context(), id)
	assert.NoError(t, err)
	verifier, err = p.DeserializeVerifier(t.Context(), id)
	assert.NoError(t, err)

	sigma, err = signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))

	keyStore, err = crypto2.NewKeyStore(curveID, kvs2.Keystore(kvs))
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, curveID)
	assert.NoError(t, err)
	p, err = NewKeyManager(config, Any, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	_, err = p.Identity(t.Context(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.Nil(t, audit)

	signer, err = p.DeserializeSigner(t.Context(), id)
	assert.NoError(t, err)
	verifier, err = p.DeserializeVerifier(t.Context(), id)
	assert.NoError(t, err)

	sigma, err = signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))
}

func TestAuditWithEidRhNymPolicy(t *testing.T) {
	testAuditWithEidRhNymPolicy(t, "./testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
	testAuditWithEidRhNymPolicy(t, "./testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY)
}

func testAuditWithEidRhNymPolicy(t *testing.T, configPath string, curveID math.CurveID) {
	t.Helper()
	registry := view.NewServiceProvider()

	kvs, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(kvs))

	config, err := crypto2.NewConfig(configPath)
	assert.NoError(t, err)
	keyStore, err := crypto2.NewKeyStore(curveID, kvs2.Keystore(kvs))
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, curveID)
	assert.NoError(t, err)
	p, err := NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	config, err = crypto2.NewConfig(configPath + "2")
	assert.NoError(t, err)
	keyStore, err = crypto2.NewKeyStore(curveID, kvs2.Keystore(kvs))
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, curveID)
	assert.NoError(t, err)
	p2, err := NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p2)

	identityDescriptor, err := p.Identity(t.Context(), nil)
	assert.NoError(t, err)
	id := identityDescriptor.Identity
	audit := identityDescriptor.AuditInfo
	assert.NotNil(t, id)
	assert.NotNil(t, audit)
	identityDescriptor2, err := p2.Identity(t.Context(), nil)
	assert.NoError(t, err)
	id2 := identityDescriptor2.Identity
	audit2 := identityDescriptor2.AuditInfo
	assert.NotNil(t, id2)
	assert.NotNil(t, audit2)

	auditInfo, err := p.DeserializeAuditInfo(t.Context(), audit)
	assert.NoError(t, err)
	assert.NoError(t, auditInfo.Match(t.Context(), id))
	assert.Error(t, auditInfo.Match(t.Context(), id2))

	auditInfo, err = p2.DeserializeAuditInfo(t.Context(), audit)
	assert.NoError(t, err)
	assert.NoError(t, auditInfo.FromBytes(audit2))
	assert.NoError(t, auditInfo.Match(t.Context(), id2))
	assert.Error(t, auditInfo.Match(t.Context(), id))
}

func TestKeyManager_DeserializeSigner(t *testing.T) {
	testKeyManager_DeserializeSigner(t, "./testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
	testKeyManager_DeserializeSigner(t, "./testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY)
}

func testKeyManager_DeserializeSigner(t *testing.T, configPath string, curveID math.CurveID) {
	t.Helper()
	// prepare
	registry := view.NewServiceProvider()
	kvs, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(kvs))
	keyStore, err := crypto2.NewKeyStore(curveID, kvs2.Keystore(kvs))
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, curveID)
	assert.NoError(t, err)

	// first key manager
	config, err := crypto2.NewConfig(configPath)
	assert.NoError(t, err)
	keyManager, err := NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager)

	// second key manager
	config, err = crypto2.NewConfig(configPath + "2")
	assert.NoError(t, err)
	keyManager2, err := NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager2)

	// keyManager and keyManager2 use the same key store

	identityDescriptor, err := keyManager.Identity(t.Context(), nil)
	assert.NoError(t, err)
	id := identityDescriptor.Identity

	identityDescriptor2, err := keyManager2.Identity(t.Context(), nil)
	assert.NoError(t, err)
	id2 := identityDescriptor2.Identity

	// This must work
	signer, err := keyManager.DeserializeSigner(t.Context(), id)
	assert.NoError(t, err)
	verifier, err := keyManager.DeserializeVerifier(t.Context(), id)
	assert.NoError(t, err)
	msg := []byte("Hello World!!!")
	sigma, err := signer.Sign(msg)
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify(msg, sigma))

	// Try to deserialize id2 with provider for id, it should fail
	_, err = keyManager.DeserializeSigner(t.Context(), id2)
	assert.Error(t, err)
	_, err = keyManager.DeserializeVerifier(t.Context(), id2)
	assert.NoError(t, err)

	// this must work
	signer, err = keyManager.DeserializeSigner(t.Context(), id)
	assert.NoError(t, err)
	verifier, err = keyManager.DeserializeVerifier(t.Context(), id)
	assert.NoError(t, err)
	sigma, err = signer.Sign(msg)
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify(msg, sigma))
}

func TestIdentityFromFabricCA(t *testing.T) {
	registry := view.NewServiceProvider()

	kvs, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(kvs))
	ipkBytes, err := crypto2.ReadFile(filepath.Join("./testdata/fp256bn_amcl/charlie.ExtraId2", idemix2.IdemixConfigFileIssuerPublicKey))
	assert.NoError(t, err)
	config, err := crypto2.NewConfigWithIPK(ipkBytes, "./testdata/fp256bn_amcl/charlie.ExtraId2", true)
	assert.NoError(t, err)

	keyStore, err := crypto2.NewKeyStore(math.BN254, kvs2.Keystore(kvs))
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, math.BN254)
	assert.NoError(t, err)
	p, err := NewKeyManager(config, types.Standard, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	identityDescriptor, err := p.Identity(t.Context(), nil)
	assert.NoError(t, err)
	id := identityDescriptor.Identity
	audit := identityDescriptor.AuditInfo
	assert.NotNil(t, id)
	assert.Nil(t, audit)

	signer, err := p.DeserializeSigner(t.Context(), id)
	assert.NoError(t, err)
	verifier, err := p.DeserializeVerifier(t.Context(), id)
	assert.NoError(t, err)

	sigma, err := signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))

	keyStore, err = crypto2.NewKeyStore(math.BN254, kvs2.Keystore(kvs))
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, math.BN254)
	assert.NoError(t, err)
	p, err = NewKeyManager(config, types.Standard, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	_, err = p.Identity(t.Context(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.Nil(t, audit)

	signer, err = p.DeserializeSigner(t.Context(), id)
	assert.NoError(t, err)
	verifier, err = p.DeserializeVerifier(t.Context(), id)
	assert.NoError(t, err)

	sigma, err = signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))

	keyStore, err = crypto2.NewKeyStore(math.BN254, kvs2.Keystore(kvs))
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, math.BN254)
	assert.NoError(t, err)
	p, err = NewKeyManager(config, Any, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	_, err = p.Identity(t.Context(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.Nil(t, audit)

	signer, err = p.DeserializeSigner(t.Context(), id)
	assert.NoError(t, err)
	verifier, err = p.DeserializeVerifier(t.Context(), id)
	assert.NoError(t, err)

	sigma, err = signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))
}

func TestIdentityFromFabricCAWithEidRhNymPolicy(t *testing.T) {
	registry := view.NewServiceProvider()

	kvs, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(kvs))
	ipkBytes, err := crypto2.ReadFile(filepath.Join("./testdata/fp256bn_amcl/charlie.ExtraId2", idemix2.IdemixConfigFileIssuerPublicKey))
	assert.NoError(t, err)
	config, err := crypto2.NewConfigWithIPK(ipkBytes, "./testdata/fp256bn_amcl/charlie.ExtraId2", true)
	assert.NoError(t, err)

	keyStore, err := crypto2.NewKeyStore(math.BN254, kvs2.Keystore(kvs))
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, math.BN254)
	assert.NoError(t, err)
	p, err := NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	// get an identity with its own audit info from the provider
	// id is in its serialized form
	identityDescriptor, err := p.Identity(t.Context(), nil)
	assert.NoError(t, err)
	id := identityDescriptor.Identity
	audit := identityDescriptor.AuditInfo
	assert.NotNil(t, id)
	assert.NotNil(t, audit)
	info, err := p.Info(t.Context(), id, audit)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: [charlie.ExtraId2]"))

	auditInfo, err := p.DeserializeAuditInfo(t.Context(), audit)
	assert.NoError(t, err)
	assert.NoError(t, auditInfo.Match(t.Context(), id))

	signer, err := p.DeserializeSigner(t.Context(), id)
	assert.NoError(t, err)
	verifier, err := p.DeserializeVerifier(t.Context(), id)
	assert.NoError(t, err)

	sigma, err := signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))

	keyStore, err = crypto2.NewKeyStore(math.BN254, kvs2.Keystore(kvs))
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, math.BN254)
	assert.NoError(t, err)
	p, err = NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	_, err = p.Identity(t.Context(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.NotNil(t, audit)
	info, err = p.Info(t.Context(), id, audit)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: [charlie.ExtraId2]"))

	auditInfo, err = p.DeserializeAuditInfo(t.Context(), audit)
	assert.NoError(t, err)
	assert.NoError(t, auditInfo.Match(t.Context(), id))

	signer, err = p.DeserializeSigner(t.Context(), id)
	assert.NoError(t, err)
	verifier, err = p.DeserializeVerifier(t.Context(), id)
	assert.NoError(t, err)

	sigma, err = signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))
}

func TestKeyManagerForRace(t *testing.T) {
	t.Run("FP256BN_AMCL", func(t *testing.T) {
		keyManager, cleanup := setupKeyManager(t, "./testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
		defer cleanup()
		runIdentityConcurrently(t, t.Context(), keyManager)
	})

	t.Run("BLS12_381_BBS", func(t *testing.T) {
		keyManager, cleanup := setupKeyManager(t, "./testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS)
		defer cleanup()
		runIdentityConcurrently(t, t.Context(), keyManager)
	})

	t.Run("BLS12_381_BBS_GURVY", func(t *testing.T) {
		keyManager, cleanup := setupKeyManager(t, "./testdata/bls12_381_bbs_gurvy/idemix", math.BLS12_381_BBS_GURVY)
		defer cleanup()
		runIdentityConcurrently(t, t.Context(), keyManager)
	})
}

func setupKeyManager(t assert.TestingT, configPath string, curveID math.CurveID) (*KeyManager, func()) {
	kvs, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	config, err := crypto2.NewConfig(configPath)
	assert.NoError(t, err)
	tracker := kvs2.NewTrackedMemoryFrom(kvs)
	keyStore, err := crypto2.NewKeyStore(curveID, tracker)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, curveID)
	assert.NoError(t, err)

	// check that version is enforced
	config.Version = 0
	_, err = NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	assert.Error(t, err)
	assert.EqualError(t, err, "unsupported protocol version [0]")
	config.Version = crypto2.ProtobufProtocolVersionV1

	// new key manager loaded from file
	assert.Empty(t, config.Signer.Ski)
	keyManager, err := NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager)
	assert.False(t, keyManager.IsRemote())
	assert.True(t, keyManager.Anonymous())
	assert.Equal(t, "alice", keyManager.EnrollmentID())
	assert.Equal(t, IdentityType, keyManager.IdentityType())
	assert.Equal(t, fmt.Sprintf("Idemix KeyManager [%s]", utils.Hashable(keyManager.Ipk).String()), keyManager.String())
	assert.Equal(t, tracker.PutCounter, 1)
	assert.Equal(t, tracker.GetCounter, 0)

	return keyManager, func() {
		// cleanup
	}
}

func runIdentityConcurrently(t assert.TestingT, ctx context.Context, keyManager *KeyManager) {
	numRoutines := 4
	var wg sync.WaitGroup
	wg.Add(numRoutines)
	for range numRoutines {
		go func() {
			defer wg.Done()

			for range 10 {
				id, err2 := keyManager.Identity(ctx, nil)
				assert.NoError(t, err2)
				assert.NotNil(t, id)
				assert.NotEmpty(t, id.Identity)
				assert.NotNil(t, id.Signer)
			}
		}()
	}
	wg.Wait()
}
