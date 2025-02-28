/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeserializer(t *testing.T) {
	p, _, err := NewKeyManager("./testdata/msp", "apple", nil, nil)
	assert.NoError(t, err)
	assert.False(t, p.Anonymous())

	id, auditInfo, err := p.Identity(nil)
	assert.NoError(t, err)
	eID := p.EnrollmentID()
	ai := &AuditInfo{}
	err = ai.FromBytes(auditInfo)
	assert.NoError(t, err)

	assert.Equal(t, eID, ai.EID)
	assert.Equal(t, "auditor.org1.example.com", eID)

	des := &IdentityDeserializer{}
	verifier, err := des.DeserializeVerifier(id)
	assert.NoError(t, err)

	signingIdentity, err := p.SigningIdentity()
	assert.NoError(t, err)

	sigma, err := signingIdentity.Sign([]byte("hello worlds"))
	assert.NoError(t, err)
	assert.NotNil(t, sigma)

	err = verifier.Verify([]byte("hello worlds"), sigma)
	assert.NoError(t, err)
}
