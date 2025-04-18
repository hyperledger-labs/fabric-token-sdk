/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	idemix2 "github.com/IBM/idemix"
	"github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	registry2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	crypto2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	"github.com/stretchr/testify/assert"
)

func TestNewKeyManager(t *testing.T) {
	testNewKeyManager(t, "./testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL, false)
	testNewKeyManager(t, "./testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS, true)
}

func testNewKeyManager(t *testing.T, configPath string, curveID math.CurveID, aries bool) {
	// prepare
	kvs, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityStore(kvs, token.TMSID{Network: "pineapple"}))
	config, err := crypto2.NewConfig(configPath)
	assert.NoError(t, err)
	tracker := kvs2.NewTrackedMemoryFrom(kvs)
	keyStore, err := crypto2.NewKeyStore(curveID, tracker)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, curveID, aries)
	assert.NoError(t, err)

	// check that version is enforced
	config.Version = 0
	_, err = NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.Error(t, err)
	assert.EqualError(t, err, "unsupported protocol version [0]")
	config.Version = crypto2.ProtobufProtocolVersionV1

	// new key manager loaded from file
	assert.Empty(t, config.Signer.Ski)
	keyManager, err := NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager)
	assert.False(t, keyManager.IsRemote())
	assert.True(t, keyManager.Anonymous())
	assert.Equal(t, "alice", keyManager.EnrollmentID())
	assert.Equal(t, IdentityType, keyManager.IdentityType())
	assert.Equal(t, fmt.Sprintf("Idemix KeyManager [%s]", hash.Hashable(keyManager.Ipk).String()), keyManager.String())
	assert.Equal(t, tracker.PutCounter, 1)
	assert.Equal(t, tracker.GetCounter, 0)

	// the config has been updated, load a new key manager
	assert.NotEmpty(t, config.Signer.Ski)
	keyManager, err = NewKeyManager(config, sigService, types.Standard, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager)
	assert.False(t, keyManager.IsRemote())
	assert.True(t, keyManager.Anonymous())
	assert.Equal(t, "alice", keyManager.EnrollmentID())
	assert.Equal(t, IdentityType, keyManager.IdentityType())
	assert.Equal(t, fmt.Sprintf("Idemix KeyManager [%s]", hash.Hashable(keyManager.Ipk).String()), keyManager.String())
	assert.Equal(t, tracker.PutCounter, 1) // this is still 1 because the key is loaded using the SKI
	assert.Equal(t, tracker.GetCounter, 1) // one get for the user key
	assert.Equal(t, tracker.GetHistory[0].Key, hex.EncodeToString(config.Signer.Ski))

	// load a new key manager again
	assert.NotEmpty(t, config.Signer.Ski)
	keyManager, err = NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager)
	assert.False(t, keyManager.IsRemote())
	assert.True(t, keyManager.Anonymous())
	assert.Equal(t, "alice", keyManager.EnrollmentID())
	assert.Equal(t, IdentityType, keyManager.IdentityType())
	assert.Equal(t, fmt.Sprintf("Idemix KeyManager [%s]", hash.Hashable(keyManager.Ipk).String()), keyManager.String())
	assert.Equal(t, tracker.PutCounter, 1) // this is still 1 because the key is loaded using the SKI
	assert.Equal(t, tracker.GetCounter, 2) // another get for the user key
	assert.Equal(t, tracker.GetHistory[1].Key, hex.EncodeToString(config.Signer.Ski))

	// invalid sig type
	_, err = NewKeyManager(config, sigService, -1, cryptoProvider)
	assert.Error(t, err)
	assert.EqualError(t, err, "unsupported signature type -1")

	assert.Equal(t, tracker.PutCounter, 1)
	assert.Equal(t, tracker.GetCounter, 3) // another get
	assert.Equal(t, tracker.GetHistory[2].Key, hex.EncodeToString(config.Signer.Ski))

	// no config
	_, err = NewKeyManager(nil, sigService, types.EidNymRhNym, cryptoProvider)
	assert.Error(t, err)
	assert.EqualError(t, err, "no idemix config provided")

	// no signer in config
	config.Signer = nil
	_, err = NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.Error(t, err)
	assert.EqualError(t, err, "no signer information found")

	// nothing changed
	assert.Equal(t, tracker.PutCounter, 1)
	assert.Equal(t, tracker.GetCounter, 3)
}

func TestIdentityWithEidRhNymPolicy(t *testing.T) {
	testIdentityWithEidRhNymPolicy(t, "./testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL, false)
	testIdentityWithEidRhNymPolicy(t, "./testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS, true)
}

func testIdentityWithEidRhNymPolicy(t *testing.T, configPath string, curveID math.CurveID, aries bool) {
	// prepare
	registry := registry2.New()
	kvs, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(kvs))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityStore(kvs, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))
	config, err := crypto2.NewConfig(configPath)
	assert.NoError(t, err)
	tracker := kvs2.NewTrackedMemoryFrom(kvs)
	keyStore, err := crypto2.NewKeyStore(curveID, tracker)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, curveID, aries)
	assert.NoError(t, err)

	// init key manager
	// with invalid sig type
	_, err = NewKeyManager(config, sigService, -1, cryptoProvider)
	assert.Error(t, err)
	assert.EqualError(t, err, "unsupported signature type -1")
	// correctly
	keyManager, err := NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager)

	// get an identity and check it
	id, audit, err := keyManager.Identity(nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.NotNil(t, audit)
	info, err := keyManager.Info(id, audit)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: [alice]"))

	// get another identity and compare the info
	id2, audit2, err := keyManager.Identity(audit)
	assert.NoError(t, err)
	assert.NotNil(t, id2)
	assert.NotNil(t, audit2)
	info2, err := keyManager.Info(id2, audit2)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info2, "Idemix: [alice]"))
	assert.Equal(t, audit, audit2)

	// deserialize the audit information
	auditInfo, err := keyManager.DeserializeAuditInfo(audit)
	assert.NoError(t, err)
	assert.NoError(t, auditInfo.Match(id))
	assert.NoError(t, auditInfo.Match(id2))
	auditInfo2, err := keyManager.DeserializeAuditInfo(audit2)
	assert.NoError(t, err)
	assert.NoError(t, auditInfo2.Match(id))
	assert.NoError(t, auditInfo2.Match(id2))

	assert.Equal(t, tracker.GetCounter, 3)

	// deserialize an invalid signer
	_, err = keyManager.DeserializeSigner(nil)
	assert.Error(t, err)
	_, err = keyManager.DeserializeSigner([]byte{})
	assert.Error(t, err)
	_, err = keyManager.DeserializeSigner([]byte{0, 1, 2})
	assert.Error(t, err)
	assert.Equal(t, tracker.GetCounter, 3)
	// deserialize a valid signer
	signer, err := keyManager.DeserializeSigner(id)
	assert.NoError(t, err)
	assert.Equal(t, tracker.GetCounter, 5) // this is due the call to Sign used to test if the signer belong to this key manager
	assert.Equal(t, hex.EncodeToString(keyManager.userKeySKI), tracker.GetHistory[4].Key)

	// deserialize an invalid verifier
	_, err = keyManager.DeserializeVerifier(nil)
	assert.Error(t, err)
	_, err = keyManager.DeserializeVerifier([]byte{})
	assert.Error(t, err)
	_, err = keyManager.DeserializeVerifier([]byte{0, 1, 2})
	assert.Error(t, err)
	// deserialize a valid verifier
	verifier, err := keyManager.DeserializeVerifier(id)
	assert.NoError(t, err)

	// get the signer from the sigService as well
	signer2, err := sigService.GetSigner(id)
	assert.NoError(t, err)
	assert.NotNil(t, signer2)

	// sign and verify
	sigma, err := signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))
	assert.Equal(t, tracker.GetCounter, 7)
	assert.Equal(t, tracker.GetHistory[3].Key, tracker.GetHistory[5].Key)
	assert.Equal(t, tracker.GetHistory[3].Value, tracker.GetHistory[5].Value)
	assert.Equal(t, hex.EncodeToString(keyManager.userKeySKI), tracker.GetHistory[6].Key)
	assert.Equal(t, tracker.GetHistory[4].Value, tracker.GetHistory[6].Value)

	sigma, err = signer2.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))
	assert.Equal(t, tracker.GetCounter, 9)
	assert.Equal(t, tracker.GetHistory[3].Key, tracker.GetHistory[7].Key)
	assert.Equal(t, tracker.GetHistory[3].Value, tracker.GetHistory[7].Value)
	assert.Equal(t, hex.EncodeToString(keyManager.userKeySKI), tracker.GetHistory[8].Key)
	assert.Equal(t, tracker.GetHistory[4].Value, tracker.GetHistory[8].Value)
}

func TestIdentityStandard(t *testing.T) {
	testIdentityStandard(t, "./testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL, false)
	testIdentityStandard(t, "./testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS, true)
}

func testIdentityStandard(t *testing.T, configPath string, curveID math.CurveID, aries bool) {
	registry := registry2.New()

	kvs, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(kvs))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityStore(kvs, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))

	config, err := crypto2.NewConfig(configPath)
	assert.NoError(t, err)

	keyStore, err := crypto2.NewKeyStore(curveID, kvs)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, curveID, aries)
	assert.NoError(t, err)
	p, err := NewKeyManager(config, sigService, types.Standard, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	id, audit, err := p.Identity(nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.Nil(t, audit)

	signer, err := p.DeserializeSigner(id)
	assert.NoError(t, err)
	verifier, err := p.DeserializeVerifier(id)
	assert.NoError(t, err)

	sigma, err := signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))

	keyStore, err = crypto2.NewKeyStore(curveID, kvs)
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, curveID, aries)
	assert.NoError(t, err)
	p, err = NewKeyManager(config, sigService, types.Standard, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	id, audit, err = p.Identity(nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.Nil(t, audit)

	signer, err = p.DeserializeSigner(id)
	assert.NoError(t, err)
	verifier, err = p.DeserializeVerifier(id)
	assert.NoError(t, err)

	sigma, err = signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))

	keyStore, err = crypto2.NewKeyStore(curveID, kvs)
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, curveID, aries)
	assert.NoError(t, err)
	p, err = NewKeyManager(config, sigService, Any, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	id, audit, err = p.Identity(nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.Nil(t, audit)

	signer, err = p.DeserializeSigner(id)
	assert.NoError(t, err)
	verifier, err = p.DeserializeVerifier(id)
	assert.NoError(t, err)

	sigma, err = signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))
}

func TestAuditWithEidRhNymPolicy(t *testing.T) {
	testAuditWithEidRhNymPolicy(t, "./testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL, false)
	testAuditWithEidRhNymPolicy(t, "./testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS, true)
}

func testAuditWithEidRhNymPolicy(t *testing.T, configPath string, curveID math.CurveID, aries bool) {
	registry := registry2.New()

	kvs, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(kvs))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityStore(kvs, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))

	config, err := crypto2.NewConfig(configPath)
	assert.NoError(t, err)
	keyStore, err := crypto2.NewKeyStore(curveID, kvs)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, curveID, aries)
	assert.NoError(t, err)
	p, err := NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	config, err = crypto2.NewConfig(configPath + "2")
	assert.NoError(t, err)
	keyStore, err = crypto2.NewKeyStore(curveID, kvs)
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, curveID, aries)
	assert.NoError(t, err)
	p2, err := NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p2)

	id, audit, err := p.Identity(nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.NotNil(t, audit)
	id2, audit2, err := p2.Identity(nil)
	assert.NoError(t, err)
	assert.NotNil(t, id2)
	assert.NotNil(t, audit2)

	auditInfo, err := p.DeserializeAuditInfo(audit)
	assert.NoError(t, err)
	assert.NoError(t, auditInfo.Match(id))
	assert.Error(t, auditInfo.Match(id2))

	auditInfo, err = p2.DeserializeAuditInfo(audit)
	assert.NoError(t, err)
	assert.NoError(t, auditInfo.FromBytes(audit2))
	assert.NoError(t, auditInfo.Match(id2))
	assert.Error(t, auditInfo.Match(id))
}

func TestKeyManager_DeserializeSigner(t *testing.T) {
	testKeyManager_DeserializeSigner(t, "./testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL, false)
	testKeyManager_DeserializeSigner(t, "./testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS, true)
}

func testKeyManager_DeserializeSigner(t *testing.T, configPath string, curveID math.CurveID, aries bool) {
	// prepare
	registry := registry2.New()
	kvs, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(kvs))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityStore(kvs, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))
	keyStore, err := crypto2.NewKeyStore(curveID, kvs)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, curveID, aries)
	assert.NoError(t, err)

	// first key manager
	config, err := crypto2.NewConfig(configPath)
	assert.NoError(t, err)
	keyManager, err := NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager)

	// second key manager
	config, err = crypto2.NewConfig(configPath + "2")
	assert.NoError(t, err)
	keyManager2, err := NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager2)

	// keyManager and keyManager2 use the same key store

	id, _, err := keyManager.Identity(nil)
	assert.NoError(t, err)

	id2, _, err := keyManager2.Identity(nil)
	assert.NoError(t, err)

	// This must work
	signer, err := keyManager.DeserializeSigner(id)
	assert.NoError(t, err)
	verifier, err := keyManager.DeserializeVerifier(id)
	assert.NoError(t, err)
	msg := []byte("Hello World!!!")
	sigma, err := signer.Sign(msg)
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify(msg, sigma))

	// Try to deserialize id2 with provider for id, it should fail
	_, err = keyManager.DeserializeSigner(id2)
	assert.Error(t, err)
	_, err = keyManager.DeserializeVerifier(id2)
	assert.NoError(t, err)

	// this must work
	des := sig.NewMultiplexDeserializer()
	des.AddDeserializer(keyManager)
	des.AddDeserializer(keyManager2)
	signer, err = des.DeserializeSigner(id)
	assert.NoError(t, err)
	verifier, err = des.DeserializeVerifier(id)
	assert.NoError(t, err)
	sigma, err = signer.Sign(msg)
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify(msg, sigma))
}

func TestIdentityFromFabricCA(t *testing.T) {
	registry := registry2.New()

	kvs, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(kvs))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityStore(kvs, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))

	ipkBytes, err := crypto2.ReadFile(filepath.Join("./testdata/fp256bn_amcl/charlie.ExtraId2", idemix2.IdemixConfigFileIssuerPublicKey))
	assert.NoError(t, err)
	config, err := crypto2.NewConfigWithIPK(ipkBytes, "./testdata/fp256bn_amcl/charlie.ExtraId2", true)
	assert.NoError(t, err)

	keyStore, err := crypto2.NewKeyStore(math.BN254, kvs)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, math.BN254, false)
	assert.NoError(t, err)
	p, err := NewKeyManager(config, sigService, types.Standard, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	id, audit, err := p.Identity(nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.Nil(t, audit)

	signer, err := p.DeserializeSigner(id)
	assert.NoError(t, err)
	verifier, err := p.DeserializeVerifier(id)
	assert.NoError(t, err)

	sigma, err := signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))

	keyStore, err = crypto2.NewKeyStore(math.BN254, kvs)
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, math.BN254, false)
	assert.NoError(t, err)
	p, err = NewKeyManager(config, sigService, types.Standard, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	id, audit, err = p.Identity(nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.Nil(t, audit)

	signer, err = p.DeserializeSigner(id)
	assert.NoError(t, err)
	verifier, err = p.DeserializeVerifier(id)
	assert.NoError(t, err)

	sigma, err = signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))

	keyStore, err = crypto2.NewKeyStore(math.BN254, kvs)
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, math.BN254, false)
	assert.NoError(t, err)
	p, err = NewKeyManager(config, sigService, Any, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	id, audit, err = p.Identity(nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.Nil(t, audit)

	signer, err = p.DeserializeSigner(id)
	assert.NoError(t, err)
	verifier, err = p.DeserializeVerifier(id)
	assert.NoError(t, err)

	sigma, err = signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))
}

func TestIdentityFromFabricCAWithEidRhNymPolicy(t *testing.T) {
	registry := registry2.New()

	kvs, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(kvs))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityStore(kvs, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))

	ipkBytes, err := crypto2.ReadFile(filepath.Join("./testdata/fp256bn_amcl/charlie.ExtraId2", idemix2.IdemixConfigFileIssuerPublicKey))
	assert.NoError(t, err)
	config, err := crypto2.NewConfigWithIPK(ipkBytes, "./testdata/fp256bn_amcl/charlie.ExtraId2", true)
	assert.NoError(t, err)

	keyStore, err := crypto2.NewKeyStore(math.BN254, kvs)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, math.BN254, false)
	assert.NoError(t, err)
	p, err := NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	// get an identity with its own audit info from the provider
	// id is in its serialized form
	id, audit, err := p.Identity(nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.NotNil(t, audit)
	info, err := p.Info(id, audit)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: [charlie.ExtraId2]"))

	auditInfo, err := p.DeserializeAuditInfo(audit)
	assert.NoError(t, err)
	assert.NoError(t, auditInfo.Match(id))

	signer, err := p.DeserializeSigner(id)
	assert.NoError(t, err)
	verifier, err := p.DeserializeVerifier(id)
	assert.NoError(t, err)

	sigma, err := signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))

	keyStore, err = crypto2.NewKeyStore(math.BN254, kvs)
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, math.BN254, false)
	assert.NoError(t, err)
	p, err = NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	id, audit, err = p.Identity(nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.NotNil(t, audit)
	info, err = p.Info(id, audit)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: [charlie.ExtraId2]"))

	auditInfo, err = p.DeserializeAuditInfo(audit)
	assert.NoError(t, err)
	assert.NoError(t, auditInfo.Match(id))

	signer, err = p.DeserializeSigner(id)
	assert.NoError(t, err)
	verifier, err = p.DeserializeVerifier(id)
	assert.NoError(t, err)

	sigma, err = signer.Sign([]byte("hello world!!!"))
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify([]byte("hello world!!!"), sigma))
}
