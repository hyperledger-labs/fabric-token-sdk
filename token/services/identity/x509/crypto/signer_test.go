/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509/crypto/csp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509/crypto/csp/kvs"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/stretchr/testify/assert"
)

func TestSKIBasedSigner_Sign(t *testing.T) {
	csp, err := GetDefaultBCCSP(csp.NewKVSStore(kvs.NewMemory()))
	assert.NoError(t, err)

	key, err := csp.KeyGen(&bccsp.ECDSAKeyGenOpts{})
	assert.NoError(t, err)

	signer, err := NewSKIBasedSigner(csp, key.SKI(), nil)
	assert.NoError(t, err)

	message := []byte("message")
	sigma, err := signer.Sign(nil, message, nil)
	assert.NoError(t, err)
	assert.NotNil(t, sigma)

	pk, err := key.PublicKey()
	assert.NoError(t, err)

	valid, err := csp.Verify(pk, sigma, message, nil)
	assert.NoError(t, err)
	assert.True(t, valid)
}
