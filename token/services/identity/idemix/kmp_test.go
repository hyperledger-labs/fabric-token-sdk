/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/membership"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	"github.com/stretchr/testify/assert"
)

type mockConfig struct {
}

func (m mockConfig) CacheSizeForOwnerID(id string) int {
	return 0
}

func (m mockConfig) TranslatePath(path string) string {
	return path
}

func (m mockConfig) IdentitiesForRole(role driver.IdentityRoleType) ([]*driver.ConfiguredIdentity, error) {
	return nil, nil
}

//go:norace
func TestNewKeyManagerProvider(t *testing.T) {
	testNewKeyManagerProvider(t, "./testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL, false)
	testNewKeyManagerProvider(t, "./testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS, true)
}

func testNewKeyManagerProvider(t *testing.T, configPath string, curveID math.CurveID, aries bool) {
	backend, err := kvs.NewInMemory()
	assert.NoError(t, err)
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	config, err := crypto.NewConfig(configPath)
	assert.NoError(t, err)
	keyStore, err := crypto.NewKeyStore(curveID, backend)
	assert.NoError(t, err)

	kmp := NewKeyManagerProvider(
		config.Ipk,
		curveID,
		keyStore,
		sigService,
		&mockConfig{},
		0,
		false,
	)
	assert.NotNil(t, kmp)
	idConfig := &token.IdentityConfiguration{
		ID:  "alice",
		URL: configPath,
	}
	km, err := kmp.Get(idConfig)
	assert.NoError(t, err)
	assert.NotNil(t, km)
	assert.NotNil(t, idConfig.Raw)
	signAndVerify(t, km)
	configRaw := idConfig.Raw
	checkRawContent(t, config.Ipk, idConfig.Raw)

	idConfig.URL = ""
	km, err = kmp.Get(idConfig)
	assert.NoError(t, err)
	assert.NotNil(t, km)
	assert.NotNil(t, idConfig.Raw)
	signAndVerify(t, km)
	checkRawContent(t, config.Ipk, idConfig.Raw)
	assert.Equal(t, configRaw, idConfig.Raw)

	km, err = kmp.Get(idConfig)
	assert.NoError(t, err)
	assert.NotNil(t, km)
	assert.NotNil(t, idConfig.Raw)
	signAndVerify(t, km)
	checkRawContent(t, config.Ipk, idConfig.Raw)
	assert.Equal(t, configRaw, idConfig.Raw)
}

func signAndVerify(t *testing.T, km membership.KeyManager) {
	id, _, err := km.Identity(nil)
	assert.NoError(t, err)
	signer, err := km.DeserializeSigner(id)
	assert.NoError(t, err)
	msg := []byte("message")
	sigma, err := signer.Sign(msg)
	assert.NoError(t, err)
	verifier, err := km.DeserializeVerifier(id)
	assert.NoError(t, err)
	assert.NoError(t, verifier.Verify(msg, sigma))
}

func checkRawContent(t *testing.T, ipk []byte, raw []byte) {
	conf, err := crypto.NewConfigFromRaw(ipk, raw)
	assert.NoError(t, err)
	assert.NotNil(t, conf)
	assert.NotNil(t, conf.Signer.Ski)
	assert.Nil(t, conf.Signer.Sk)
}
