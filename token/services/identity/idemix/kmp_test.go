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

func TestNewKeyManagerProvider(t *testing.T) {
	backend, err := kvs.NewInMemory()
	assert.NoError(t, err)
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	config, err := crypto.NewConfig("./testdata/idemix")
	assert.NoError(t, err)
	keyStore, err := crypto.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)

	kmp := NewKeyManagerProvider(
		config.Ipk,
		math.FP256BN_AMCL,
		keyStore,
		sigService,
		&mockConfig{},
		10,
		false,
	)
	assert.NotNil(t, kmp)
	idConfig := &token.IdentityConfiguration{
		ID:  "alice",
		URL: "./testdata/idemix",
	}
	km, err := kmp.Get(idConfig)
	assert.NoError(t, err)
	assert.NotNil(t, km)
	assert.NotNil(t, idConfig.Raw)
	signAndVerify(t, km)

	idConfig.URL = ""
	km, err = kmp.Get(idConfig)
	assert.NoError(t, err)
	assert.NotNil(t, km)
	assert.NotNil(t, idConfig.Raw)
	signAndVerify(t, km)

	km, err = kmp.Get(idConfig)
	assert.NoError(t, err)
	assert.NotNil(t, km)
	assert.NotNil(t, idConfig.Raw)
	signAndVerify(t, km)
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
