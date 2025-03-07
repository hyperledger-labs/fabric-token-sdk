/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix_test

import (
	"path/filepath"
	"strings"
	"testing"

	idemix2 "github.com/IBM/idemix"
	"github.com/IBM/idemix/bccsp/types"
	bccsp "github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	registry2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix"
	crypto2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	"github.com/stretchr/testify/assert"
)

func TestNewKeyManager(t *testing.T) {
	backend, err := kvs2.NewInMemoryKVS()
	assert.NoError(t, err)
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))

	config, err := crypto2.NewConfig("./testdata/idemix")
	assert.NoError(t, err)

	keyStore, err := crypto2.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)

	keyManager, err := idemix.NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager)
	assert.False(t, keyManager.IsRemote())
	assert.True(t, keyManager.Anonymous())
	assert.Equal(t, "alice", keyManager.EnrollmentID())
	assert.Equal(t, idemix.IdentityType, keyManager.IdentityType())
	assert.Equal(t, "Idemix KeyManager [dJZK5i5D2i5B8S9DsVWDFzdHSJE/jcTLk9VaJzFB4fo=]", keyManager.String())

	keyManager, err = idemix.NewKeyManager(config, sigService, bccsp.Standard, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager)
	assert.False(t, keyManager.IsRemote())
	assert.True(t, keyManager.Anonymous())
	assert.Equal(t, "alice", keyManager.EnrollmentID())
	assert.Equal(t, idemix.IdentityType, keyManager.IdentityType())
	assert.Equal(t, "Idemix KeyManager [dJZK5i5D2i5B8S9DsVWDFzdHSJE/jcTLk9VaJzFB4fo=]", keyManager.String())

	keyManager, err = idemix.NewKeyManager(config, sigService, bccsp.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager)
	assert.False(t, keyManager.IsRemote())
	assert.True(t, keyManager.Anonymous())
	assert.Equal(t, "alice", keyManager.EnrollmentID())
	assert.Equal(t, idemix.IdentityType, keyManager.IdentityType())
	assert.Equal(t, "Idemix KeyManager [dJZK5i5D2i5B8S9DsVWDFzdHSJE/jcTLk9VaJzFB4fo=]", keyManager.String())

	// invalid sig type
	_, err = idemix.NewKeyManager(config, sigService, -1, cryptoProvider)
	assert.Error(t, err)
	assert.EqualError(t, err, "unsupported signature type -1")

	// no config
	_, err = idemix.NewKeyManager(nil, sigService, bccsp.EidNymRhNym, cryptoProvider)
	assert.Error(t, err)
	assert.EqualError(t, err, "nil conf reference")

	// no signer in config
	config.Signer = nil
	_, err = idemix.NewKeyManager(config, sigService, bccsp.EidNymRhNym, cryptoProvider)
	assert.Error(t, err)
	assert.EqualError(t, err, "no signer information found")
}

func TestIdentityWithEidRhNymPolicy(t *testing.T) {
	registry := registry2.New()

	backend, err := kvs2.NewInMemoryKVS()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(backend))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))

	config, err := crypto2.NewConfig("./testdata/idemix")
	assert.NoError(t, err)

	keyStore, err := crypto2.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p, err := idemix.NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	id, audit, err := p.Identity(nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.NotNil(t, audit)
	info, err := p.Info(id, audit)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: [alice]"))

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

	keyStore, err = crypto2.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p, err = idemix.NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	id, audit, err = p.Identity(nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.NotNil(t, audit)
	info, err = p.Info(id, audit)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: [alice]"))

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

func TestIdentityStandard(t *testing.T) {
	registry := registry2.New()

	backend, err := kvs2.NewInMemoryKVS()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(backend))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))

	config, err := crypto2.NewConfig("./testdata/idemix")
	assert.NoError(t, err)

	keyStore, err := crypto2.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p, err := idemix.NewKeyManager(config, sigService, bccsp.Standard, cryptoProvider)
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

	keyStore, err = crypto2.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p, err = idemix.NewKeyManager(config, sigService, bccsp.Standard, cryptoProvider)
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

	keyStore, err = crypto2.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p, err = idemix.NewKeyManager(config, sigService, idemix.Any, cryptoProvider)
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
	registry := registry2.New()

	backend, err := kvs2.NewInMemoryKVS()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(backend))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))

	config, err := crypto2.NewConfig("./testdata/idemix")
	assert.NoError(t, err)
	keyStore, err := crypto2.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p, err := idemix.NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	config, err = crypto2.NewConfig("./testdata/idemix2")
	assert.NoError(t, err)
	keyStore, err = crypto2.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p2, err := idemix.NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
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
	registry := registry2.New()

	backend, err := kvs2.NewInMemoryKVS()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(backend))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))

	config, err := crypto2.NewConfig("./testdata/sameissuer/idemix")
	assert.NoError(t, err)
	keyStore, err := crypto2.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p, err := idemix.NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	config, err = crypto2.NewConfig("./testdata/sameissuer/idemix2")
	assert.NoError(t, err)
	p2, err := idemix.NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p2)

	id, _, err := p.Identity(nil)
	assert.NoError(t, err)

	id2, _, err := p2.Identity(nil)
	assert.NoError(t, err)

	// This must work
	signer, err := p.DeserializeSigner(id)
	assert.NoError(t, err)
	verifier, err := p.DeserializeVerifier(id)
	assert.NoError(t, err)
	msg := []byte("Hello World!!!")
	sigma, err := signer.Sign(msg)
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify(msg, sigma))

	// Try to deserialize id2 with provider for id, must fail
	_, err = p.DeserializeSigner(id2)
	assert.Error(t, err)
	_, err = p.DeserializeVerifier(id2)
	assert.NoError(t, err)

	// this must work
	des := sig.NewMultiplexDeserializer()
	des.AddDeserializer(p)
	des.AddDeserializer(p2)
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

	backend, err := kvs2.NewInMemoryKVS()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(backend))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))

	ipkBytes, err := crypto2.ReadFile(filepath.Join("./testdata/charlie.ExtraId2", idemix2.IdemixConfigFileIssuerPublicKey))
	assert.NoError(t, err)
	config, err := crypto2.NewConfigWithIPK(ipkBytes, "./testdata/charlie.ExtraId2", true)
	assert.NoError(t, err)

	keyStore, err := crypto2.NewKeyStore(math.BN254, backend)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, math.BN254, false)
	assert.NoError(t, err)
	p, err := idemix.NewKeyManager(config, sigService, bccsp.Standard, cryptoProvider)
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

	keyStore, err = crypto2.NewKeyStore(math.BN254, backend)
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, math.BN254, false)
	assert.NoError(t, err)
	p, err = idemix.NewKeyManager(config, sigService, bccsp.Standard, cryptoProvider)
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

	keyStore, err = crypto2.NewKeyStore(math.BN254, backend)
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, math.BN254, false)
	assert.NoError(t, err)
	p, err = idemix.NewKeyManager(config, sigService, idemix.Any, cryptoProvider)
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

	backend, err := kvs2.NewInMemoryKVS()
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(backend))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))

	ipkBytes, err := crypto2.ReadFile(filepath.Join("./testdata/charlie.ExtraId2", idemix2.IdemixConfigFileIssuerPublicKey))
	assert.NoError(t, err)
	config, err := crypto2.NewConfigWithIPK(ipkBytes, "./testdata/charlie.ExtraId2", true)
	assert.NoError(t, err)

	keyStore, err := crypto2.NewKeyStore(math.BN254, backend)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, math.BN254, false)
	assert.NoError(t, err)
	p, err := idemix.NewKeyManager(config, sigService, bccsp.EidNymRhNym, cryptoProvider)
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

	keyStore, err = crypto2.NewKeyStore(math.BN254, backend)
	assert.NoError(t, err)
	cryptoProvider, err = crypto2.NewBCCSP(keyStore, math.BN254, false)
	assert.NoError(t, err)
	p, err = idemix.NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
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
