/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestECDSAVerifier(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	verifier := NewECDSAVerifier(&priv.PublicKey)
	signer := NewEcdsaSigner(priv)

	message := []byte("test message")

	// Test signing a message and then verifying it
	t.Run("Valid Signature", func(t *testing.T) {
		sigma, err := signer.Sign(message)
		require.NoError(t, err)

		err = verifier.Verify(message, sigma)
		require.NoError(t, err)
	})

	// Test failure in verifying an invalid signature
	t.Run("Invalid Signature", func(t *testing.T) {
		err := verifier.Verify(message, []byte("invalid"))
		require.Error(t, err)
	})

	// Test failure in verifying an a signature against the wrong message
	t.Run("Wrong Message", func(t *testing.T) {
		sigma, err := signer.Sign(message)
		require.NoError(t, err)

		err = verifier.Verify([]byte("wrong message"), sigma)
		require.Error(t, err)
	})
}

// To reduce risk of malleability attacks the S component of the ecdsa signature should be low.
// The following tests test the related IsLowSand ToLowS functions
func TestIsLowS(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Test the IsLowS function with a low S value
	t.Run("Low S", func(t *testing.T) {
		s := big.NewInt(100)
		isLow, err := IsLowS(&priv.PublicKey, s)
		require.NoError(t, err)
		assert.True(t, isLow)
	})

	// Test failures to use the IsLowS function with an unsupported curve
	t.Run("Unsupported Curve", func(t *testing.T) {
		// Create a key with an unsupported curve
		type unsupportedCurve struct {
			elliptic.Curve
		}
		pk := &ecdsa.PublicKey{
			Curve: unsupportedCurve{elliptic.P256()},
		}
		_, err := IsLowS(pk, big.NewInt(100))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "curve not recognized")
	})
}

func TestToLowS(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Test that using ToLowS on an already low value doesn't modify it
	t.Run("Already Low S", func(t *testing.T) {
		s := big.NewInt(100)
		result, changed, err := ToLowS(&priv.PublicKey, s)
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, s, result)
	})

	// Test that using ToLowS on a high value modifies it to a low value
	t.Run("High S Converted", func(t *testing.T) {
		// Create a high S value
		halfOrder := curveHalfOrders[priv.Curve]
		highS := new(big.Int).Add(halfOrder, big.NewInt(100))
		originalHighS := new(big.Int).Set(highS)

		result, changed, err := ToLowS(&priv.PublicKey, highS)
		require.NoError(t, err)
		assert.True(t, changed)
		// Note: ToLowS modifies the input, so we compare with the original
		assert.NotEqual(t, originalHighS, result)

		// Verify it's now low
		isLow, err := IsLowS(&priv.PublicKey, result)
		require.NoError(t, err)
		assert.True(t, isLow)
	})
}

func TestPemDecodeKey(t *testing.T) {
	// Serialize a private key (marshal and encode) and then test that this decodes to the original
	t.Run("ECDSA Private Key", func(t *testing.T) {
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
		require.NoError(t, err)

		pemBytes := pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privBytes,
		})

		key, err := PemDecodeKey(pemBytes)
		require.NoError(t, err)
		assert.NotNil(t, key)
		decodedPriv, ok := key.(*ecdsa.PrivateKey)
		assert.True(t, ok)

		// Verify the decoded key matches the original
		assert.Equal(t, priv.D, decodedPriv.D)
		assert.Equal(t, priv.X, decodedPriv.X)
		assert.Equal(t, priv.Y, decodedPriv.Y)
	})

	// Test decoding a key from a raw pem certificate
	t.Run("Certificate", func(t *testing.T) {
		certPEM, priv := GenerateTestCert(t)
		key, err := PemDecodeKey(certPEM)
		require.NoError(t, err)
		assert.NotNil(t, key)
		decodedPub, ok := key.(*ecdsa.PublicKey)
		assert.True(t, ok)

		// Verify the decoded public key matches the original
		assert.Equal(t, priv.X, decodedPub.X)
		assert.Equal(t, priv.Y, decodedPub.Y)
	})

	// Serialize a public key (marshal and encode) and then test that this decodes to the original
	t.Run("Public Key", func(t *testing.T) {
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
		require.NoError(t, err)

		pemBytes := pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubBytes,
		})

		key, err := PemDecodeKey(pemBytes)
		require.NoError(t, err)
		assert.NotNil(t, key)
		decodedPub, ok := key.(*ecdsa.PublicKey)
		assert.True(t, ok)

		// Verify the decoded public key matches the original
		assert.Equal(t, priv.X, decodedPub.X)
		assert.Equal(t, priv.Y, decodedPub.Y)
	})

	// Test failure to decode an invalid raw pem key
	t.Run("Not PEM Encoded", func(t *testing.T) {
		_, err := PemDecodeKey([]byte("not pem"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not PEM encoded")
	})

	// Test failure to decode a key of an unknown type
	t.Run("Bad Key Type", func(t *testing.T) {
		pemBytes := pem.EncodeToMemory(&pem.Block{
			Type:  "UNKNOWN",
			Bytes: []byte("data"),
		})
		_, err := PemDecodeKey(pemBytes)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad key type")
	})
}

func TestPemDecodeCert(t *testing.T) {
	// Test decoding a valid raw pem certificate
	t.Run("Valid Certificate", func(t *testing.T) {
		certPEM, _ := GenerateTestCert(t)
		cert, err := PemDecodeCert(certPEM)
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.Equal(t, "test.example.com", cert.Subject.CommonName)
	})

	// Test failure to decode an invalid pem certificate
	t.Run("Not PEM", func(t *testing.T) {
		_, err := PemDecodeCert([]byte("not pem"))
		require.Error(t, err)
	})

	// Test failure to decode a certificate of an unknown type
	t.Run("Wrong Type", func(t *testing.T) {
		pemBytes := pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: []byte("data"),
		})
		_, err := PemDecodeCert(pemBytes)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad type")
	})
}

// Test that a signed message is verified as valid
func TestEcdsaSigner(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	signer := NewEcdsaSigner(priv)

	t.Run("Sign", func(t *testing.T) {
		msg := []byte("test message")
		sig, err := signer.Sign(msg)
		require.NoError(t, err)
		assert.NotEmpty(t, sig)

		// Verify the signature
		verifier := NewECDSAVerifier(&priv.PublicKey)
		err = verifier.Verify(msg, sig)
		require.NoError(t, err)
	})
}

// Test failure to decode an invalid raw pem certificate
func TestPemDecodeCert_InvalidCert(t *testing.T) {
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: []byte("invalid cert data"),
	})

	_, err := PemDecodeCert(pemData)
	require.Error(t, err)
}
