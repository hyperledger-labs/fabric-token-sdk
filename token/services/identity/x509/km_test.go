/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509/crypto/csp/kvs"
	"github.com/stretchr/testify/assert"
)

func TestDeserializer(t *testing.T) {
	keyStore := NewKeyStore(kvs.NewMemoryKVS())

	// load a full identity capable of signing as well
	fullIdentityProvider, _, err := NewKeyManager("./testdata/msp", nil, nil, keyStore)
	assert.NoError(t, err)
	assert.False(t, fullIdentityProvider.Anonymous())
	// load a full identity capable of signing as well with a custom keystore path
	fullIdentityProvider2, _, err := NewKeyManagerFromConf(nil, "./testdata/msp2", KeystoreFullFolder, nil, nil, keyStore)
	assert.NoError(t, err)
	assert.False(t, fullIdentityProvider.Anonymous())
	// load a verifying only provider
	verifyingIdentityProvider, _, err := NewKeyManager("./testdata/msp1", nil, nil, keyStore)
	assert.NoError(t, err)

	for _, provider := range []*KeyManager{fullIdentityProvider, fullIdentityProvider2} {
		id, auditInfo, err := provider.Identity(nil)
		assert.NoError(t, err)
		eID := provider.EnrollmentID()
		ai := &AuditInfo{}
		err = ai.FromBytes(auditInfo)
		assert.NoError(t, err)
		assert.Equal(t, eID, ai.EID)
		assert.Equal(t, "auditor.org1.example.com", eID)
		des := &IdentityDeserializer{}
		verifier, err := des.DeserializeVerifier(id)
		assert.NoError(t, err)
		signingIdentity := provider.SigningIdentity()
		assert.NotNil(t, signingIdentity)
		sigma, err := signingIdentity.Sign([]byte("hello worlds"))
		assert.NoError(t, err)
		assert.NotNil(t, sigma)
		err = verifier.Verify([]byte("hello worlds"), sigma)
		assert.NoError(t, err)

		// check again a verifying identity
		verifyingIdentity, _, err := verifyingIdentityProvider.Identity(nil)
		assert.NoError(t, err)
		verifier2, err := provider.DeserializeVerifier(verifyingIdentity)
		assert.NoError(t, err)
		err = verifier2.Verify([]byte("hello worlds"), sigma)
		assert.NoError(t, err)
	}

}
