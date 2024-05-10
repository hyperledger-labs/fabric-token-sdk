/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix/msp"

	idemix2 "github.com/IBM/idemix"
	"github.com/IBM/idemix/bccsp/types"
	bccsp "github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs/mock"
	registry2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	msp2 "github.com/hyperledger/fabric/msp"
	"github.com/stretchr/testify/assert"
)

func TestProvider(t *testing.T) {
	backend, err := kvs.NewWithConfig("memory", "", &mock.ConfigProvider{})
	assert.NoError(t, err)
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))

	config, err := msp2.GetLocalMspConfigWithType("./testdata/idemix", nil, "idemix", "idemix")
	assert.NoError(t, err)

	keyStore, err := msp.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err := msp.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p, err := idemix.NewProvider(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	p, err = idemix.NewProvider(config, sigService, bccsp.Standard, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	p, err = idemix.NewProvider(config, sigService, bccsp.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestIdentityWithEidRhNymPolicy(t *testing.T) {
	registry := registry2.New()

	backend, err := kvs.NewWithConfig("memory", "", &mock.ConfigProvider{})
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(backend))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))

	config, err := msp2.GetLocalMspConfigWithType("./testdata/idemix", nil, "idemix", "idemix")
	assert.NoError(t, err)

	keyStore, err := msp.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err := msp.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p, err := idemix.NewProvider(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	id, audit, err := p.Identity(nil)
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.NotNil(t, audit)
	info, err := p.Info(id, audit)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "MSP.Idemix: [alice]"))
	assert.True(t, strings.HasSuffix(info, "[idemix][idemixorg.example.com][ADMIN]"))

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

	keyStore, err = msp.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err = msp.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p, err = idemix.NewProvider(config, sigService, idemix.Any, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	id, audit, err = p.Identity(&common.IdentityOptions{EIDExtension: true})
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.NotNil(t, audit)
	info, err = p.Info(id, audit)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "MSP.Idemix: [alice]"))
	assert.True(t, strings.HasSuffix(info, "[idemix][idemixorg.example.com][ADMIN]"))

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

	backend, err := kvs.NewWithConfig("memory", "", &mock.ConfigProvider{})
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(backend))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))

	config, err := msp2.GetLocalMspConfigWithType("./testdata/idemix", nil, "idemix", "idemix")
	assert.NoError(t, err)

	keyStore, err := msp.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err := msp.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p, err := idemix.NewProvider(config, sigService, bccsp.Standard, cryptoProvider)
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

	keyStore, err = msp.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err = msp.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p, err = idemix.NewProvider(config, sigService, bccsp.Standard, cryptoProvider)
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

	keyStore, err = msp.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err = msp.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p, err = idemix.NewProvider(config, sigService, idemix.Any, cryptoProvider)
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

	backend, err := kvs.NewWithConfig("memory", "", &mock.ConfigProvider{})
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(backend))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))

	config, err := msp2.GetLocalMspConfigWithType("./testdata/idemix", nil, "idemix", "idemix")
	assert.NoError(t, err)
	keyStore, err := msp.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err := msp.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p, err := idemix.NewProvider(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	config, err = msp2.GetLocalMspConfigWithType("./testdata/idemix2", nil, "idemix", "idemix")
	assert.NoError(t, err)
	keyStore, err = msp.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err = msp.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p2, err := idemix.NewProvider(config, sigService, types.EidNymRhNym, cryptoProvider)
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

func TestProvider_DeserializeSigner(t *testing.T) {
	registry := registry2.New()

	backend, err := kvs.NewWithConfig("memory", "", &mock.ConfigProvider{})
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(backend))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))

	config, err := msp2.GetLocalMspConfigWithType("./testdata/sameissuer/idemix", nil, "idemix", "idemix")
	assert.NoError(t, err)
	keyStore, err := msp.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err := msp.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)
	p, err := idemix.NewProvider(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	config, err = msp2.GetLocalMspConfigWithType("./testdata/sameissuer/idemix2", nil, "idemix", "idemix")
	assert.NoError(t, err)
	p2, err := idemix.NewProvider(config, sigService, types.EidNymRhNym, cryptoProvider)
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

	backend, err := kvs.NewWithConfig("memory", "", &mock.ConfigProvider{})
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(backend))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))

	ipkBytes, err := msp.ReadFile(filepath.Join("./testdata/charlie.ExtraId2", idemix2.IdemixConfigFileIssuerPublicKey))
	assert.NoError(t, err)
	config, err := msp.NewMSPConfigFromURL(ipkBytes, "./testdata/charlie.ExtraId2", "charlie.ExtraId2", true)
	assert.NoError(t, err)

	keyStore, err := msp.NewKeyStore(math.BN254, backend)
	assert.NoError(t, err)
	cryptoProvider, err := msp.NewBCCSP(keyStore, math.BN254, false)
	assert.NoError(t, err)
	p, err := idemix.NewProvider(config, sigService, bccsp.Standard, cryptoProvider)
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

	keyStore, err = msp.NewKeyStore(math.BN254, backend)
	assert.NoError(t, err)
	cryptoProvider, err = msp.NewBCCSP(keyStore, math.BN254, false)
	assert.NoError(t, err)
	p, err = idemix.NewProvider(config, sigService, bccsp.Standard, cryptoProvider)
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

	keyStore, err = msp.NewKeyStore(math.BN254, backend)
	assert.NoError(t, err)
	cryptoProvider, err = msp.NewBCCSP(keyStore, math.BN254, false)
	assert.NoError(t, err)
	p, err = idemix.NewProvider(config, sigService, idemix.Any, cryptoProvider)
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

	backend, err := kvs.NewWithConfig("memory", "", &mock.ConfigProvider{})
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(backend))
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	assert.NoError(t, registry.RegisterService(sigService))

	ipkBytes, err := msp.ReadFile(filepath.Join("./testdata/charlie.ExtraId2", idemix2.IdemixConfigFileIssuerPublicKey))
	assert.NoError(t, err)
	config, err := msp.NewMSPConfigFromURL(ipkBytes, "./testdata/charlie.ExtraId2", "charlie.ExtraId2", true)
	assert.NoError(t, err)

	keyStore, err := msp.NewKeyStore(math.BN254, backend)
	assert.NoError(t, err)
	cryptoProvider, err := msp.NewBCCSP(keyStore, math.BN254, false)
	assert.NoError(t, err)
	p, err := idemix.NewProvider(config, sigService, bccsp.EidNymRhNym, cryptoProvider)
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
	assert.True(t, strings.HasPrefix(info, "MSP.Idemix: [charlie.ExtraId2]"))
	assert.True(t, strings.HasSuffix(info, "[charlie.ExtraId2][][MEMBER]"))

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

	keyStore, err = msp.NewKeyStore(math.BN254, backend)
	assert.NoError(t, err)
	cryptoProvider, err = msp.NewBCCSP(keyStore, math.BN254, false)
	assert.NoError(t, err)
	p, err = idemix.NewProvider(config, sigService, idemix.Any, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	id, audit, err = p.Identity(&common.IdentityOptions{EIDExtension: true})
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.NotNil(t, audit)
	info, err = p.Info(id, audit)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "MSP.Idemix: [charlie.ExtraId2]"))
	assert.True(t, strings.HasSuffix(info, "[charlie.ExtraId2][][MEMBER]"))

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
