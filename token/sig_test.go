/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/stretchr/testify/assert"
)

func TestSignatureService_AuditorVerifier(t *testing.T) {
	deserializer := &mock.Deserializer{}
	ip := &mock.IdentityProvider{}

	service := &SignatureService{
		deserializer: deserializer,
		ip:           ip,
	}

	expectedVerifier := &mock.Verifier{}
	deserializer.GetAuditorVerifierReturns(expectedVerifier, nil)

	id := []byte("auditor_identity")
	verifier, err := service.AuditorVerifier(id)

	assert.NoError(t, err)
	assert.Equal(t, expectedVerifier, verifier)
}

func TestSignatureService_IssuerVerifier(t *testing.T) {
	deserializer := &mock.Deserializer{}
	ip := &mock.IdentityProvider{}

	service := &SignatureService{
		deserializer: deserializer,
		ip:           ip,
	}

	expectedVerifier := &mock.Verifier{}
	deserializer.GetIssuerVerifierReturns(expectedVerifier, nil)

	id := []byte("issuer_identity")
	verifier, err := service.IssuerVerifier(id)

	assert.NoError(t, err)
	assert.Equal(t, expectedVerifier, verifier)
}

func TestSignatureService_OwnerVerifier(t *testing.T) {
	deserializer := &mock.Deserializer{}
	ip := &mock.IdentityProvider{}

	service := &SignatureService{
		deserializer: deserializer,
		ip:           ip,
	}

	expectedVerifier := &mock.Verifier{}
	deserializer.GetOwnerVerifierReturns(expectedVerifier, nil)

	id := []byte("owner_identity")
	verifier, err := service.OwnerVerifier(id)

	assert.NoError(t, err)
	assert.Equal(t, expectedVerifier, verifier)
}

func TestSignatureService_GetSigner(t *testing.T) {
	deserializer := &mock.Deserializer{}
	ip := &mock.IdentityProvider{}

	service := &SignatureService{
		deserializer: deserializer,
		ip:           ip,
	}

	expectedSigner := &mock.Signer{}
	ip.GetSignerReturns(expectedSigner, nil)

	id := []byte("identity")
	signer, err := service.GetSigner(id)

	assert.NoError(t, err)
	assert.Equal(t, expectedSigner, signer)
}

func TestSignatureService_RegisterSigner(t *testing.T) {
	deserializer := &mock.Deserializer{}
	ip := &mock.IdentityProvider{}

	service := &SignatureService{
		deserializer: deserializer,
		ip:           ip,
	}

	id := []byte("identity")
	signer := &mock.Signer{}
	verifier := &mock.Verifier{}

	ip.RegisterSignerReturns(nil)

	err := service.RegisterSigner(id, signer, verifier)

	assert.NoError(t, err)
}

func TestSignatureService_IsMe(t *testing.T) {
	deserializer := &mock.Deserializer{}
	ip := &mock.IdentityProvider{}

	service := &SignatureService{
		deserializer: deserializer,
		ip:           ip,
	}

	ip.IsMeReturns(true)

	id := []byte("identity")
	isMe := service.IsMe(id)

	assert.True(t, isMe)
}
