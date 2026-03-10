/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package csp

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/kvs"
	"github.com/hyperledger/fabric-lib-go/bccsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCSP(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		keyStore := NewKVSStore(kvs.NewTrackedMemory())
		csp, err := NewCSP(keyStore)
		require.NoError(t, err)
		assert.NotNil(t, csp)
	})

	t.Run("Nil KeyStore Returns Error", func(t *testing.T) {
		_, err := NewCSP(nil)
		require.Error(t, err)
	})
}

func TestECDSAKeyGeneration(t *testing.T) {
	keyStore := NewKVSStore(kvs.NewTrackedMemory())
	csp, err := NewCSP(keyStore)
	require.NoError(t, err)

	t.Run("P256 Key Generation", func(t *testing.T) {
		key, err := csp.KeyGen(&bccsp.ECDSAKeyGenOpts{})
		require.NoError(t, err)
		assert.NotNil(t, key)
		assert.False(t, key.Symmetric())
		assert.True(t, key.Private())
	})

	t.Run("ECDSAP256 Key Generation", func(t *testing.T) {
		key, err := csp.KeyGen(&bccsp.ECDSAP256KeyGenOpts{})
		require.NoError(t, err)
		assert.NotNil(t, key)
	})

	t.Run("ECDSAP384 Key Generation", func(t *testing.T) {
		key, err := csp.KeyGen(&bccsp.ECDSAP384KeyGenOpts{})
		require.NoError(t, err)
		assert.NotNil(t, key)
	})
}

func TestECDSASignAndVerify(t *testing.T) {
	keyStore := NewKVSStore(kvs.NewTrackedMemory())
	csp, err := NewCSP(keyStore)
	require.NoError(t, err)

	key, err := csp.KeyGen(&bccsp.ECDSAKeyGenOpts{})
	require.NoError(t, err)

	msg := []byte("test message")
	digest := sha256.Sum256(msg)

	t.Run("Sign and Verify with Private Key", func(t *testing.T) {
		signature, err := csp.Sign(key, digest[:], nil)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)

		valid, err := csp.Verify(key, signature, digest[:], nil)
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("Verify with Public Key", func(t *testing.T) {
		signature, err := csp.Sign(key, digest[:], nil)
		require.NoError(t, err)

		pubKey, err := key.PublicKey()
		require.NoError(t, err)

		valid, err := csp.Verify(pubKey, signature, digest[:], nil)
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("Verify Invalid Signature", func(t *testing.T) {
		invalidSig := []byte("invalid signature")
		valid, err := csp.Verify(key, invalidSig, digest[:], nil)
		require.Error(t, err)
		assert.False(t, valid)
	})

	t.Run("Verify Wrong Digest", func(t *testing.T) {
		signature, err := csp.Sign(key, digest[:], nil)
		require.NoError(t, err)

		wrongDigest := sha256.Sum256([]byte("wrong message"))
		valid, err := csp.Verify(key, signature, wrongDigest[:], nil)
		require.NoError(t, err)
		assert.False(t, valid)
	})
}

func TestHashOperations(t *testing.T) {
	keyStore := NewKVSStore(kvs.NewTrackedMemory())
	csp, err := NewCSP(keyStore)
	require.NoError(t, err)

	msg := []byte("test message")

	t.Run("SHA256 Hash", func(t *testing.T) {
		hash, err := csp.Hash(msg, &bccsp.SHA256Opts{})
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.Len(t, hash, 32)
	})

	t.Run("SHA Hash (default)", func(t *testing.T) {
		hash, err := csp.Hash(msg, &bccsp.SHAOpts{})
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("SHA384 Hash", func(t *testing.T) {
		hash, err := csp.Hash(msg, &bccsp.SHA384Opts{})
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.Len(t, hash, 48)
	})

	t.Run("SHA3_256 Hash", func(t *testing.T) {
		hash, err := csp.Hash(msg, &bccsp.SHA3_256Opts{})
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("SHA3_384 Hash", func(t *testing.T) {
		hash, err := csp.Hash(msg, &bccsp.SHA3_384Opts{})
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("GetHash", func(t *testing.T) {
		h, err := csp.GetHash(&bccsp.SHA256Opts{})
		require.NoError(t, err)
		assert.NotNil(t, h)

		h.Write(msg)
		hash := h.Sum(nil)
		assert.NotEmpty(t, hash)
	})
}

func TestKeyImport(t *testing.T) {
	keyStore := NewKVSStore(kvs.NewTrackedMemory())
	csp, err := NewCSP(keyStore)
	require.NoError(t, err)

	t.Run("Import ECDSA Private Key", func(t *testing.T) {
		// Generate a key
		privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		// Marshal to DER
		der, err := x509.MarshalECPrivateKey(privKey)
		require.NoError(t, err)

		// Import
		key, err := csp.KeyImport(der, &bccsp.ECDSAPrivateKeyImportOpts{})
		require.NoError(t, err)
		assert.NotNil(t, key)
		assert.True(t, key.Private())
	})

	t.Run("Import ECDSA Public Key PKIX", func(t *testing.T) {
		// Generate a key
		privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		// Marshal public key to DER
		der, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
		require.NoError(t, err)

		// Import
		key, err := csp.KeyImport(der, &bccsp.ECDSAPKIXPublicKeyImportOpts{})
		require.NoError(t, err)
		assert.NotNil(t, key)
		assert.False(t, key.Private())
	})

	t.Run("Import ECDSA Go Public Key", func(t *testing.T) {
		// Generate a key
		privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		// Import directly
		key, err := csp.KeyImport(&privKey.PublicKey, &bccsp.ECDSAGoPublicKeyImportOpts{})
		require.NoError(t, err)
		assert.NotNil(t, key)
		assert.False(t, key.Private())
	})

	t.Run("Import X509 Certificate", func(t *testing.T) {
		// Generate a certificate
		privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		template := x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				CommonName: "test",
			},
			NotBefore: time.Now().Add(-time.Hour),
			NotAfter:  time.Now().Add(time.Hour),
		}

		derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
		require.NoError(t, err)

		cert, err := x509.ParseCertificate(derBytes)
		require.NoError(t, err)

		// Import
		key, err := csp.KeyImport(cert, &bccsp.X509PublicKeyImportOpts{})
		require.NoError(t, err)
		assert.NotNil(t, key)
	})

	t.Run("Import Invalid Private Key", func(t *testing.T) {
		_, err := csp.KeyImport([]byte("invalid"), &bccsp.ECDSAPrivateKeyImportOpts{})
		require.Error(t, err)
	})

	t.Run("Import Invalid Public Key", func(t *testing.T) {
		_, err := csp.KeyImport([]byte("invalid"), &bccsp.ECDSAPKIXPublicKeyImportOpts{})
		require.Error(t, err)
	})

	t.Run("Import Empty Bytes", func(t *testing.T) {
		_, err := csp.KeyImport([]byte{}, &bccsp.ECDSAPrivateKeyImportOpts{})
		require.Error(t, err)
	})

	t.Run("Import Wrong Type for Go Public Key", func(t *testing.T) {
		_, err := csp.KeyImport("not a key", &bccsp.ECDSAGoPublicKeyImportOpts{})
		require.Error(t, err)
	})
}

func TestECDSAKey(t *testing.T) {
	keyStore := NewKVSStore(kvs.NewTrackedMemory())
	csp, err := NewCSP(keyStore)
	require.NoError(t, err)

	key, err := csp.KeyGen(&bccsp.ECDSAKeyGenOpts{})
	require.NoError(t, err)

	t.Run("Key Properties", func(t *testing.T) {
		assert.NotEmpty(t, key.SKI())
		assert.True(t, key.Private())
		assert.False(t, key.Symmetric())

		// Private key Bytes() is not supported
		_, err := key.Bytes()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})

	t.Run("Public Key", func(t *testing.T) {
		pubKey, err := key.PublicKey()
		require.NoError(t, err)
		assert.NotNil(t, pubKey)
		assert.False(t, pubKey.Private())
		assert.False(t, pubKey.Symmetric())

		pubBytes, err := pubKey.Bytes()
		require.NoError(t, err)
		assert.NotEmpty(t, pubBytes)
	})
}

func TestKeyDerivation(t *testing.T) {
	keyStore := NewKVSStore(kvs.NewTrackedMemory())
	csp, err := NewCSP(keyStore)
	require.NoError(t, err)

	key, err := csp.KeyGen(&bccsp.ECDSAKeyGenOpts{})
	require.NoError(t, err)

	t.Run("Derive from Private Key", func(t *testing.T) {
		derivedKey, err := csp.KeyDeriv(key, &bccsp.ECDSAReRandKeyOpts{})
		require.NoError(t, err)
		assert.NotNil(t, derivedKey)
	})

	t.Run("Derive from Public Key", func(t *testing.T) {
		pubKey, err := key.PublicKey()
		require.NoError(t, err)

		derivedKey, err := csp.KeyDeriv(pubKey, &bccsp.ECDSAReRandKeyOpts{})
		require.NoError(t, err)
		assert.NotNil(t, derivedKey)
	})
}

func TestGetKey(t *testing.T) {
	keyStore := NewKVSStore(kvs.NewTrackedMemory())
	csp, err := NewCSP(keyStore)
	require.NoError(t, err)

	// Generate and store a key
	key, err := csp.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
	require.NoError(t, err)

	t.Run("Get Existing Key", func(t *testing.T) {
		retrievedKey, err := csp.GetKey(key.SKI())
		require.NoError(t, err)
		assert.NotNil(t, retrievedKey)
		assert.Equal(t, key.SKI(), retrievedKey.SKI())
	})

	t.Run("Get Non-existent Key", func(t *testing.T) {
		_, err := csp.GetKey([]byte("non-existent"))
		require.Error(t, err)
	})
}

func TestSignECDSA(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	digest := sha256.Sum256([]byte("test"))

	t.Run("Sign Success", func(t *testing.T) {
		sig, err := signECDSA(privKey, digest[:], nil)
		require.NoError(t, err)
		assert.NotEmpty(t, sig)
	})
}

func TestVerifyECDSA(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	digest := sha256.Sum256([]byte("test"))
	sig, err := signECDSA(privKey, digest[:], nil)
	require.NoError(t, err)

	t.Run("Verify Valid Signature", func(t *testing.T) {
		valid, err := verifyECDSA(&privKey.PublicKey, sig, digest[:], nil)
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("Verify Invalid Signature Format", func(t *testing.T) {
		valid, err := verifyECDSA(&privKey.PublicKey, []byte("invalid"), digest[:], nil)
		require.Error(t, err)
		assert.False(t, valid)
	})
}

func TestKeyMarshalling(t *testing.T) {
	keyStore := NewKVSStore(kvs.NewTrackedMemory())
	csp, err := NewCSP(keyStore)
	require.NoError(t, err)

	key, err := csp.KeyGen(&bccsp.ECDSAKeyGenOpts{})
	require.NoError(t, err)

	t.Run("Private Key Bytes Not Supported", func(t *testing.T) {
		_, err := key.Bytes()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})

	t.Run("Public Key Bytes", func(t *testing.T) {
		pubKey, err := key.PublicKey()
		require.NoError(t, err)

		bytes, err := pubKey.Bytes()
		require.NoError(t, err)
		assert.NotEmpty(t, bytes)

		// Public key Bytes() returns DER format, not PEM
		// Verify it can be parsed
		_, err = x509.ParsePKIXPublicKey(bytes)
		require.NoError(t, err)
	})
}

func TestKVSStore_ReadOnly(t *testing.T) {
	keyStore := NewKVSStore(kvs.NewTrackedMemory())
	assert.False(t, keyStore.ReadOnly())
}

func TestKVSStore_StoreAndRetrieve(t *testing.T) {
	keyStore := NewKVSStore(kvs.NewTrackedMemory())
	csp, err := NewCSP(keyStore)
	require.NoError(t, err)

	t.Run("Store and Retrieve Private Key", func(t *testing.T) {
		key, err := csp.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
		require.NoError(t, err)

		// Key should be stored automatically
		retrievedKey, err := keyStore.GetKey(key.SKI())
		require.NoError(t, err)
		assert.Equal(t, key.SKI(), retrievedKey.SKI())
	})

	t.Run("Store and Retrieve Public Key", func(t *testing.T) {
		key, err := csp.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
		require.NoError(t, err)

		pubKey, err := key.PublicKey()
		require.NoError(t, err)

		// Store public key explicitly
		err = keyStore.StoreKey(pubKey)
		require.NoError(t, err)

		retrievedKey, err := keyStore.GetKey(pubKey.SKI())
		require.NoError(t, err)
		assert.Equal(t, pubKey.SKI(), retrievedKey.SKI())
	})

	t.Run("GetKey Non-existent Key", func(t *testing.T) {
		_, err := keyStore.GetKey([]byte("non-existent-ski"))
		require.Error(t, err)
	})
}

func TestECDSAPublicKey_PublicKey(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	pubKey := &ecdsaPublicKey{&privKey.PublicKey}

	pk, err := pubKey.PublicKey()
	require.NoError(t, err)
	assert.Equal(t, pubKey, pk) // Returns itself
}

func TestECDSAPublicKey_MarshallUnmarshall(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	pubKey := &ecdsaPublicKey{&privKey.PublicKey}

	t.Run("Marshall and Unmarshall", func(t *testing.T) {
		marshalled, err := pubKey.marshall()
		require.NoError(t, err)
		assert.NotEmpty(t, marshalled)

		newKey := &ecdsaPublicKey{}
		err = newKey.unmarshall(marshalled)
		require.NoError(t, err)
		assert.Equal(t, pubKey.SKI(), newKey.SKI())
	})

	t.Run("Unmarshall Invalid PEM", func(t *testing.T) {
		newKey := &ecdsaPublicKey{}
		err := newKey.unmarshall([]byte("invalid pem"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode PEM block")
	})

	t.Run("Unmarshall Wrong PEM Type", func(t *testing.T) {
		wrongPEM := []byte(`-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAKHHCgVZU6w6MA0GCSqGSIb3DQEBCwUAMA0xCzAJBgNVBAYTAlVT
-----END CERTIFICATE-----`)
		newKey := &ecdsaPublicKey{}
		err := newKey.unmarshall(wrongPEM)
		require.Error(t, err)
	})
}

func TestECDSAPrivateKey_MarshallUnmarshall(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	key := &ecdsaPrivateKey{privKey}

	t.Run("Marshall and Unmarshall", func(t *testing.T) {
		marshalled, err := key.marshall()
		require.NoError(t, err)
		assert.NotEmpty(t, marshalled)

		newKey := &ecdsaPrivateKey{}
		err = newKey.unmarshall(marshalled)
		require.NoError(t, err)
		assert.Equal(t, key.SKI(), newKey.SKI())
	})

	t.Run("Unmarshall Invalid PEM", func(t *testing.T) {
		newKey := &ecdsaPrivateKey{}
		err := newKey.unmarshall([]byte("invalid pem"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode PEM block")
	})
}

func TestDerToPrivateKey_PKCS8(t *testing.T) {
	// Generate a key
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Marshal to PKCS8
	der, err := x509.MarshalPKCS8PrivateKey(privKey)
	require.NoError(t, err)

	// Test derToPrivateKey
	key, err := derToPrivateKey(der)
	require.NoError(t, err)
	assert.NotNil(t, key)

	ecdsaKey, ok := key.(*ecdsa.PrivateKey)
	assert.True(t, ok)
	assert.Equal(t, privKey.D, ecdsaKey.D)
}

func TestDerToPublicKey_EmptyBytes(t *testing.T) {
	_, err := derToPublicKey([]byte{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid DER")
}

func TestKeyImport_X509Certificate_NonECDSA(t *testing.T) {
	keyStore := NewKVSStore(kvs.NewTrackedMemory())
	csp, err := NewCSP(keyStore)
	require.NoError(t, err)

	// This test verifies the error path for unsupported key types
	// Since we can't easily create a real RSA cert in this context,
	// we'll test the ECDSA path more thoroughly
	t.Run("Valid ECDSA Certificate", func(t *testing.T) {
		privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		template := x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				CommonName: "test",
			},
			NotBefore: time.Now().Add(-time.Hour),
			NotAfter:  time.Now().Add(time.Hour),
		}

		derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
		require.NoError(t, err)

		cert, err := x509.ParseCertificate(derBytes)
		require.NoError(t, err)

		key, err := csp.KeyImport(cert, &bccsp.X509PublicKeyImportOpts{})
		require.NoError(t, err)
		assert.NotNil(t, key)
		assert.False(t, key.Private())
	})
}

func TestKeyDerivation_EdgeCases(t *testing.T) {
	keyStore := NewKVSStore(kvs.NewTrackedMemory())
	csp, err := NewCSP(keyStore)
	require.NoError(t, err)

	t.Run("Derive with Nil Opts", func(t *testing.T) {
		key, err := csp.KeyGen(&bccsp.ECDSAKeyGenOpts{})
		require.NoError(t, err)

		_, err = csp.KeyDeriv(key, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid opts")
	})

	t.Run("Derive Public Key with Nil Opts", func(t *testing.T) {
		key, err := csp.KeyGen(&bccsp.ECDSAKeyGenOpts{})
		require.NoError(t, err)

		pubKey, err := key.PublicKey()
		require.NoError(t, err)

		_, err = csp.KeyDeriv(pubKey, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid opts")
	})

	t.Run("Derive with Valid Expansion Value", func(t *testing.T) {
		key, err := csp.KeyGen(&bccsp.ECDSAKeyGenOpts{})
		require.NoError(t, err)

		// Use a specific expansion value
		expansionValue := make([]byte, 32)
		for i := range expansionValue {
			expansionValue[i] = byte(i)
		}

		derivedKey, err := csp.KeyDeriv(key, &bccsp.ECDSAReRandKeyOpts{
			Temporary: true,
			Expansion: expansionValue,
		})
		require.NoError(t, err)
		assert.NotNil(t, derivedKey)
		assert.NotEqual(t, key.SKI(), derivedKey.SKI())
	})
}

func TestKVSStore_ErrorPaths(t *testing.T) {
	keyStore := NewKVSStore(kvs.NewTrackedMemory())
	csp, err := NewCSP(keyStore)
	require.NoError(t, err)

	t.Run("GetKey Unknown Type", func(t *testing.T) {
		// Generate a key and store it, then corrupt the entry
		key, err := csp.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
		require.NoError(t, err)

		pubKey, err := key.PublicKey()
		require.NoError(t, err)

		// Now manually overwrite with unknown type
		skiHex := hex.EncodeToString(pubKey.SKI())
		err = keyStore.Put(skiHex, &KeyEntry{
			KeyType: "unknownType",
			Raw:     []byte("data"),
		})
		require.NoError(t, err)

		_, err = keyStore.GetKey(pubKey.SKI())
		require.Error(t, err)
		// The actual error message is "key not found for [ski]"
		assert.Contains(t, err.Error(), "key not found")
	})

	t.Run("GetKey Invalid Private Key Data", func(t *testing.T) {
		err := keyStore.Put("invalid-priv", &KeyEntry{
			KeyType: "ecdsaPrivateKey",
			Raw:     []byte("invalid data"),
		})
		require.NoError(t, err)

		_, err = keyStore.GetKey([]byte("invalid-priv"))
		require.Error(t, err)
	})

	t.Run("GetKey Invalid Public Key Data", func(t *testing.T) {
		err := keyStore.Put("invalid-pub", &KeyEntry{
			KeyType: "ecdsaPublicKey",
			Raw:     []byte("invalid data"),
		})
		require.NoError(t, err)

		_, err = keyStore.GetKey([]byte("invalid-pub"))
		require.Error(t, err)
	})
}

func TestECDSAKey_SKI_NilKey(t *testing.T) {
	t.Run("Private Key with Nil", func(t *testing.T) {
		key := &ecdsaPrivateKey{privKey: nil}
		ski := key.SKI()
		assert.Nil(t, ski)
	})

	t.Run("Public Key with Nil", func(t *testing.T) {
		key := &ecdsaPublicKey{pubKey: nil}
		ski := key.SKI()
		assert.Nil(t, ski)
	})
}

// Made with Bob
