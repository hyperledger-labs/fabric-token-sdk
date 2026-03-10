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
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509/crypto/csp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509/crypto/mocks"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/kvs"
	"github.com/hyperledger/fabric-lib-go/bccsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions
func generateTestCert(t *testing.T) ([]byte, *ecdsa.PrivateKey) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test.example.com",
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	return certPEM, priv
}

func TestDeserializeVerifier(t *testing.T) {
	certPEM, _ := generateTestCert(t)

	t.Run("Success", func(t *testing.T) {
		verifier, err := DeserializeVerifier(driver.Identity(certPEM))
		require.NoError(t, err)
		assert.NotNil(t, verifier)
	})

	t.Run("Invalid PEM", func(t *testing.T) {
		_, err := DeserializeVerifier(driver.Identity([]byte("invalid")))
		require.Error(t, err)
	})

	t.Run("Not ECDSA Key", func(t *testing.T) {
		// Create RSA key instead of ECDSA
		block := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: []byte("not a valid key"),
		}
		pemBytes := pem.EncodeToMemory(block)
		_, err := DeserializeVerifier(driver.Identity(pemBytes))
		require.Error(t, err)
	})
}

func TestInfo(t *testing.T) {
	certPEM, _ := generateTestCert(t)

	t.Run("Success", func(t *testing.T) {
		info, err := Info(certPEM)
		require.NoError(t, err)
		assert.Contains(t, info, "X509:")
		assert.Contains(t, info, "test.example.com")
	})

	t.Run("Invalid Certificate", func(t *testing.T) {
		_, err := Info([]byte("invalid"))
		require.Error(t, err)
	})
}

func TestGetBCCSPFromConf(t *testing.T) {
	keyStore := csp.NewKVSStore(kvs.NewTrackedMemory())

	t.Run("Nil Config - Default SW", func(t *testing.T) {
		csp, err := GetBCCSPFromConf(nil, keyStore)
		require.NoError(t, err)
		assert.NotNil(t, csp)
	})

	t.Run("SW Provider", func(t *testing.T) {
		conf := &BCCSP{
			Default: "SW",
			SW: &SoftwareProvider{
				Hash:     "SHA2",
				Security: 256,
			},
		}
		csp, err := GetBCCSPFromConf(conf, keyStore)
		require.NoError(t, err)
		assert.NotNil(t, csp)
	})

	t.Run("Invalid Provider", func(t *testing.T) {
		conf := &BCCSP{
			Default: "INVALID",
		}
		_, err := GetBCCSPFromConf(conf, keyStore)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid BCCSP.Default")
	})

	t.Run("PKCS11 Without Config", func(t *testing.T) {
		conf := &BCCSP{
			Default: "PKCS11",
		}
		_, err := GetBCCSPFromConf(conf, keyStore)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing configuration")
	})
}

func TestGetDefaultBCCSP(t *testing.T) {
	t.Run("With KeyStore", func(t *testing.T) {
		keyStore := csp.NewKVSStore(kvs.NewTrackedMemory())
		csp, err := GetDefaultBCCSP(keyStore)
		require.NoError(t, err)
		assert.NotNil(t, csp)
	})

	t.Run("Nil KeyStore", func(t *testing.T) {
		csp, err := GetDefaultBCCSP(nil)
		require.NoError(t, err)
		assert.NotNil(t, csp)
	})
}

func TestSKIMapper(t *testing.T) {
	t.Run("With Matching SKI", func(t *testing.T) {
		p11Opts := PKCS11{
			KeyIDs: []KeyIDMapping{
				{SKI: "abcd", ID: "key1"},
			},
		}
		mapper := skiMapper(p11Opts)
		result := mapper([]byte{0xab, 0xcd})
		assert.Equal(t, []byte("key1"), result)
	})

	t.Run("With AltID", func(t *testing.T) {
		p11Opts := PKCS11{
			AltID: "altkey",
		}
		mapper := skiMapper(p11Opts)
		result := mapper([]byte{0x12, 0x34})
		assert.Equal(t, []byte("altkey"), result)
	})

	t.Run("No Match Returns SKI", func(t *testing.T) {
		p11Opts := PKCS11{}
		mapper := skiMapper(p11Opts)
		ski := []byte{0x12, 0x34}
		result := mapper(ski)
		assert.Equal(t, ski, result)
	})
}

func TestUnmarshalConfig(t *testing.T) {
	t.Run("Valid Config", func(t *testing.T) {
		// Create a simple config
		origConfig := &Config{
			Version: 1,
		}
		data, err := MarshalConfig(origConfig)
		require.NoError(t, err)

		config, err := UnmarshalConfig(data)
		require.NoError(t, err)
		assert.Equal(t, uint64(1), config.Version)
	})

	t.Run("Invalid Data", func(t *testing.T) {
		_, err := UnmarshalConfig([]byte("invalid"))
		require.Error(t, err)
	})
}

func TestMarshalConfig(t *testing.T) {
	config := &Config{
		Version: 1,
	}
	data, err := MarshalConfig(config)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestToBCCSPOpts(t *testing.T) {
	t.Run("Valid Options", func(t *testing.T) {
		input := map[string]interface{}{
			"BCCSP": map[string]interface{}{
				"Default": "SW",
				"SW": map[string]interface{}{
					"Hash":     "SHA2",
					"Security": 256,
				},
			},
		}
		opts, err := ToBCCSPOpts(input)
		require.NoError(t, err)
		assert.NotNil(t, opts)
		assert.Equal(t, "SW", opts.Default)
	})

	t.Run("Empty Options", func(t *testing.T) {
		opts, err := ToBCCSPOpts(map[string]interface{}{})
		require.NoError(t, err)
		assert.Nil(t, opts)
	})
}

func TestToPKCS11OptsOpts(t *testing.T) {
	input := &PKCS11{
		Security:       256,
		Hash:           "SHA2",
		Library:        "/usr/lib/libpkcs11.so",
		Label:          "test",
		Pin:            "1234",
		SoftwareVerify: true,
		Immutable:      false,
		AltID:          "alt",
		KeyIDs: []KeyIDMapping{
			{SKI: "ski1", ID: "id1"},
		},
	}

	result := ToPKCS11OptsOpts(input)
	assert.Equal(t, 256, result.Security)
	assert.Equal(t, "SHA2", result.Hash)
	assert.Equal(t, "/usr/lib/libpkcs11.so", result.Library)
	assert.Equal(t, "test", result.Label)
	assert.Equal(t, "1234", result.Pin)
	assert.True(t, result.SoftwareVerify)
	assert.False(t, result.Immutable)
	assert.Equal(t, "alt", result.AltID)
	assert.Len(t, result.KeyIDs, 1)
}

func TestBCCSPOpts(t *testing.T) {
	t.Run("SW Provider", func(t *testing.T) {
		opts, err := BCCSPOpts("SW")
		require.NoError(t, err)
		assert.Equal(t, "SW", opts.Default)
		assert.NotNil(t, opts.SW)
		assert.Equal(t, "SHA2", opts.SW.Hash)
		assert.Equal(t, 256, opts.SW.Security)
	})
}

func TestECDSAVerifier(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	verifier := NewECDSAVerifier(&priv.PublicKey)
	signer := NewEcdsaSigner(priv)

	message := []byte("test message")

	t.Run("Valid Signature", func(t *testing.T) {
		sigma, err := signer.Sign(message)
		require.NoError(t, err)

		err = verifier.Verify(message, sigma)
		require.NoError(t, err)
	})

	t.Run("Invalid Signature", func(t *testing.T) {
		err := verifier.Verify(message, []byte("invalid"))
		require.Error(t, err)
	})

	t.Run("Wrong Message", func(t *testing.T) {
		sigma, err := signer.Sign(message)
		require.NoError(t, err)

		err = verifier.Verify([]byte("wrong message"), sigma)
		require.Error(t, err)
	})
}

func TestIsLowS(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	t.Run("Low S", func(t *testing.T) {
		s := big.NewInt(100)
		isLow, err := IsLowS(&priv.PublicKey, s)
		require.NoError(t, err)
		assert.True(t, isLow)
	})

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

	t.Run("Already Low S", func(t *testing.T) {
		s := big.NewInt(100)
		result, changed, err := ToLowS(&priv.PublicKey, s)
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, s, result)
	})

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
		_, ok := key.(*ecdsa.PrivateKey)
		assert.True(t, ok)
	})

	t.Run("Certificate", func(t *testing.T) {
		certPEM, _ := generateTestCert(t)
		key, err := PemDecodeKey(certPEM)
		require.NoError(t, err)
		assert.NotNil(t, key)
		_, ok := key.(*ecdsa.PublicKey)
		assert.True(t, ok)
	})

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
	})

	t.Run("Not PEM Encoded", func(t *testing.T) {
		_, err := PemDecodeKey([]byte("not pem"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not PEM encoded")
	})

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
	t.Run("Valid Certificate", func(t *testing.T) {
		certPEM, _ := generateTestCert(t)
		cert, err := PemDecodeCert(certPEM)
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.Equal(t, "test.example.com", cert.Subject.CommonName)
	})

	t.Run("Not PEM", func(t *testing.T) {
		_, err := PemDecodeCert([]byte("not pem"))
		require.Error(t, err)
	})

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

func TestGetEnrollmentID(t *testing.T) {
	t.Run("Simple CommonName", func(t *testing.T) {
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		template := x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				CommonName: "user123",
			},
			NotBefore: time.Now().Add(-time.Hour),
			NotAfter:  time.Now().Add(time.Hour),
		}

		derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
		require.NoError(t, err)

		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

		eid, err := GetEnrollmentID(certPEM)
		require.NoError(t, err)
		assert.Equal(t, "user123", eid)
	})

	t.Run("CommonName With @", func(t *testing.T) {
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		template := x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				CommonName: "user123@example.com",
			},
			NotBefore: time.Now().Add(-time.Hour),
			NotAfter:  time.Now().Add(time.Hour),
		}

		derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
		require.NoError(t, err)

		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

		eid, err := GetEnrollmentID(certPEM)
		require.NoError(t, err)
		assert.Equal(t, "user123", eid)
	})

	t.Run("Invalid Certificate", func(t *testing.T) {
		_, err := GetEnrollmentID([]byte("invalid"))
		require.Error(t, err)
	})
}

func TestGetRevocationHandle(t *testing.T) {
	t.Run("Valid Certificate", func(t *testing.T) {
		certPEM, _ := generateTestCert(t)
		rh, err := GetRevocationHandle(certPEM)
		require.NoError(t, err)
		assert.NotEmpty(t, rh)
	})

	t.Run("Invalid Certificate", func(t *testing.T) {
		_, err := GetRevocationHandle([]byte("invalid"))
		require.Error(t, err)
	})
}

func TestReadFile(t *testing.T) {
	t.Run("Valid File", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.txt")
		content := []byte("test content")
		err := os.WriteFile(path, content, 0600)
		require.NoError(t, err)

		data, err := readFile(path)
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("Non-existent File", func(t *testing.T) {
		_, err := readFile("/non/existent/file")
		require.Error(t, err)
	})
}

func TestGetPemMaterialFromDir(t *testing.T) {
	t.Run("Valid Directory", func(t *testing.T) {
		dir := t.TempDir()
		certPEM, _ := generateTestCert(t)

		// Write a cert file
		err := os.WriteFile(filepath.Join(dir, "cert.pem"), certPEM, 0600)
		require.NoError(t, err)

		materials, err := getPemMaterialFromDir(dir)
		require.NoError(t, err)
		assert.Len(t, materials, 1)
	})

	t.Run("Non-existent Directory", func(t *testing.T) {
		_, err := getPemMaterialFromDir("/non/existent/dir")
		require.Error(t, err)
	})

	t.Run("Empty Directory", func(t *testing.T) {
		dir := t.TempDir()
		materials, err := getPemMaterialFromDir(dir)
		require.NoError(t, err)
		assert.Empty(t, materials)
	})

	t.Run("Directory With Non-PEM Files", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("not pem"), 0600)
		require.NoError(t, err)

		materials, err := getPemMaterialFromDir(dir)
		require.NoError(t, err)
		assert.Empty(t, materials)
	})
}

func TestGetHashOpt(t *testing.T) {
	t.Run("SHA2", func(t *testing.T) {
		opt, err := getHashOpt(bccsp.SHA2)
		require.NoError(t, err)
		assert.NotNil(t, opt)
	})

	t.Run("SHA3", func(t *testing.T) {
		opt, err := getHashOpt(bccsp.SHA3)
		require.NoError(t, err)
		assert.NotNil(t, opt)
	})

	t.Run("Unknown Hash Family", func(t *testing.T) {
		_, err := getHashOpt("UNKNOWN")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hash family not recognized")
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("Valid MSP Directory", func(t *testing.T) {
		config, err := LoadConfig("../testdata/msp", "")
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, ProtobufProtocolVersionV1, config.Version)
		assert.NotNil(t, config.SigningIdentity)
		assert.NotEmpty(t, config.SigningIdentity.PublicSigner)
	})

	t.Run("Custom KeyStore Directory", func(t *testing.T) {
		config, err := LoadConfig("../testdata/msp2", "keystoreFull")
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.NotNil(t, config.SigningIdentity.PrivateSigner)
	})

	t.Run("Non-existent Directory", func(t *testing.T) {
		_, err := LoadConfig("/non/existent/path", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not load a valid signer certificate")
	})

	t.Run("Directory Without Certificates", func(t *testing.T) {
		dir := t.TempDir()
		signcertDir := filepath.Join(dir, SignCertsDirName)
		err := os.MkdirAll(signcertDir, 0750)
		require.NoError(t, err)

		_, err = LoadConfig(dir, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no signer certificate found")
	})
}

func TestLoadConfigWithIdentityInfo(t *testing.T) {
	certPEM, _ := generateTestCert(t)

	t.Run("Valid Identity Info", func(t *testing.T) {
		info := &SigningIdentityInfo{
			PublicSigner: certPEM,
			PrivateSigner: &KeyInfo{
				KeyIdentifier: "test-key",
				KeyMaterial:   []byte("key-material"),
			},
		}

		config, err := LoadConfigWithIdentityInfo(info)
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, ProtobufProtocolVersionV1, config.Version)
		assert.Equal(t, bccsp.SHA2, config.CryptoConfig.SignatureHashFamily)
	})
}

func TestRemovePrivateSigner(t *testing.T) {
	certPEM, _ := generateTestCert(t)
	config := &Config{
		Version: ProtobufProtocolVersionV1,
		SigningIdentity: &SigningIdentityInfo{
			PublicSigner: certPEM,
			PrivateSigner: &KeyInfo{
				KeyIdentifier: "test",
				KeyMaterial:   []byte("material"),
			},
		},
	}

	result, err := RemovePrivateSigner(config)
	require.NoError(t, err)
	assert.Nil(t, result.SigningIdentity.PrivateSigner)
}

func TestSerializeIdentity(t *testing.T) {
	t.Run("Valid Config", func(t *testing.T) {
		config, err := LoadConfig("../testdata/msp", "")
		require.NoError(t, err)

		serialized, err := SerializeIdentity(config)
		require.NoError(t, err)
		assert.NotEmpty(t, serialized)
	})

	t.Run("Invalid Config", func(t *testing.T) {
		config := &Config{
			SigningIdentity: &SigningIdentityInfo{
				PublicSigner: []byte("invalid"),
			},
			CryptoConfig: &CryptoConfig{
				SignatureHashFamily: bccsp.SHA2,
			},
		}

		_, err := SerializeIdentity(config)
		require.Error(t, err)
	})
}

func TestGetSigningIdentity(t *testing.T) {
	keyStore := csp.NewKVSStore(kvs.NewTrackedMemory())

	t.Run("Valid Config", func(t *testing.T) {
		config, err := LoadConfig("../testdata/msp", "")
		require.NoError(t, err)

		identity, err := GetSigningIdentity(config, nil, keyStore)
		require.NoError(t, err)
		assert.NotNil(t, identity)

		// Test signing
		msg := []byte("test message")
		sig, err := identity.Sign(msg)
		require.NoError(t, err)
		assert.NotEmpty(t, sig)

		// Test verification
		err = identity.Verify(msg, sig)
		require.NoError(t, err)
	})

	t.Run("Invalid Config", func(t *testing.T) {
		config := &Config{
			SigningIdentity: &SigningIdentityInfo{
				PublicSigner: []byte("invalid"),
			},
			CryptoConfig: &CryptoConfig{
				SignatureHashFamily: bccsp.SHA2,
			},
		}

		_, err := GetSigningIdentity(config, nil, keyStore)
		require.Error(t, err)
	})
}

func TestDeserializeIdentity(t *testing.T) {
	keyStore := csp.NewKVSStore(kvs.NewTrackedMemory())

	t.Run("Valid Identity", func(t *testing.T) {
		// First, get a valid signing identity
		config, err := LoadConfig("../testdata/msp", "")
		require.NoError(t, err)

		origIdentity, err := GetSigningIdentity(config, nil, keyStore)
		require.NoError(t, err)

		// Serialize it
		serialized, err := origIdentity.Serialize()
		require.NoError(t, err)

		// Deserialize it
		identity, err := DeserializeIdentity(serialized, nil, keyStore)
		require.NoError(t, err)
		assert.NotNil(t, identity)

		// Test that it can sign
		msg := []byte("test message")
		sig, err := identity.Sign(msg)
		require.NoError(t, err)
		assert.NotEmpty(t, sig)
	})

	t.Run("Invalid Identity", func(t *testing.T) {
		_, err := DeserializeIdentity([]byte("invalid"), nil, keyStore)
		require.Error(t, err)
	})
}

func TestIdentityFactory(t *testing.T) {
	keyStore := csp.NewKVSStore(kvs.NewTrackedMemory())
	cspInstance, err := GetDefaultBCCSP(keyStore)
	require.NoError(t, err)

	factory := NewIdentityFactory(cspInstance, bccsp.SHA2)
	assert.NotNil(t, factory)

	t.Run("GetFullIdentity", func(t *testing.T) {
		config, err := LoadConfig("../testdata/msp", "")
		require.NoError(t, err)

		identity, err := factory.GetFullIdentity(config.SigningIdentity)
		require.NoError(t, err)
		assert.NotNil(t, identity)

		// Test signing and verification
		msg := []byte("test")
		sig, err := identity.Sign(msg)
		require.NoError(t, err)
		err = identity.Verify(msg, sig)
		require.NoError(t, err)
	})

	t.Run("GetIdentity", func(t *testing.T) {
		config, err := LoadConfig("../testdata/msp1", "")
		require.NoError(t, err)

		identity, err := factory.GetIdentity(config.SigningIdentity)
		require.NoError(t, err)
		assert.NotNil(t, identity)
	})

	t.Run("DeserializeFullIdentity", func(t *testing.T) {
		config, err := LoadConfig("../testdata/msp", "")
		require.NoError(t, err)

		origIdentity, err := factory.GetFullIdentity(config.SigningIdentity)
		require.NoError(t, err)

		serialized, err := origIdentity.Serialize()
		require.NoError(t, err)

		identity, err := factory.DeserializeFullIdentity(serialized)
		require.NoError(t, err)
		assert.NotNil(t, identity)
	})
}

func TestVerifyingIdentity(t *testing.T) {
	keyStore := csp.NewKVSStore(kvs.NewTrackedMemory())
	cspInstance, err := GetDefaultBCCSP(keyStore)
	require.NoError(t, err)

	factory := NewIdentityFactory(cspInstance, bccsp.SHA2)

	config, err := LoadConfig("../testdata/msp1", "")
	require.NoError(t, err)

	identity, err := factory.GetIdentity(config.SigningIdentity)
	require.NoError(t, err)

	t.Run("Serialize", func(t *testing.T) {
		serialized, err := identity.Serialize()
		require.NoError(t, err)
		assert.NotEmpty(t, serialized)
	})

	t.Run("Verify Valid Signature", func(t *testing.T) {
		// Get a full identity to create a signature
		fullConfig, err := LoadConfig("../testdata/msp", "")
		require.NoError(t, err)

		fullIdentity, err := factory.GetFullIdentity(fullConfig.SigningIdentity)
		require.NoError(t, err)

		msg := []byte("test message for verification")
		sig, err := fullIdentity.Sign(msg)
		require.NoError(t, err)

		// Now verify with the verifying identity
		err = identity.Verify(msg, sig)
		require.NoError(t, err)
	})

	t.Run("Verify Invalid Signature", func(t *testing.T) {
		msg := []byte("test message")
		invalidSig := []byte("invalid signature")

		err := identity.Verify(msg, invalidSig)
		require.Error(t, err)
	})
}

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

func TestFullIdentity_Verify(t *testing.T) {
	keyStore := csp.NewKVSStore(kvs.NewTrackedMemory())
	cspInstance, err := GetDefaultBCCSP(keyStore)
	require.NoError(t, err)

	factory := NewIdentityFactory(cspInstance, bccsp.SHA2)
	config, err := LoadConfig("../testdata/msp", "")
	require.NoError(t, err)

	identity, err := factory.GetFullIdentity(config.SigningIdentity)
	require.NoError(t, err)

	t.Run("Verify Own Signature", func(t *testing.T) {
		msg := []byte("test message")
		sig, err := identity.Sign(msg)
		require.NoError(t, err)

		err = identity.Verify(msg, sig)
		require.NoError(t, err)
	})

	t.Run("Verify Invalid Signature", func(t *testing.T) {
		msg := []byte("test message")
		err := identity.Verify(msg, []byte("invalid"))
		require.Error(t, err)
	})

	t.Run("Verify Wrong Message", func(t *testing.T) {
		msg := []byte("test message")
		sig, err := identity.Sign(msg)
		require.NoError(t, err)

		err = identity.Verify([]byte("wrong message"), sig)
		require.Error(t, err)
	})
}

func TestMarshalConfig_EmptyConfig(t *testing.T) {
	// Test with empty config - should still marshal successfully
	config := &Config{
		Version: 1,
	}
	data, err := MarshalConfig(config)
	require.NoError(t, err)
	assert.NotNil(t, data)
}

func TestGetPKCS11BCCSP_Error(t *testing.T) {
	t.Run("Nil PKCS11 Config", func(t *testing.T) {
		conf := &BCCSP{
			Default: "PKCS11",
			PKCS11:  nil,
		}
		_, err := GetPKCS11BCCSP(conf, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing configuration")
	})
}

func TestReadPemFile_Errors(t *testing.T) {
	t.Run("File With Invalid PEM Type", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "invalid.pem")

		// Write a PEM with invalid type
		pemData := pem.EncodeToMemory(&pem.Block{
			Type:  "INVALID TYPE",
			Bytes: []byte("data"),
		})
		err := os.WriteFile(path, pemData, 0600)
		require.NoError(t, err)

		_, err = readPemFile(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected PEM block type")
	})
}

func TestGetEnrollmentID_WithAt(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "user@domain.com",
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	eid, err := GetEnrollmentID(certPEM)
	require.NoError(t, err)
	assert.Equal(t, "user", eid)
}

func TestPemDecodeCert_InvalidCert(t *testing.T) {
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: []byte("invalid cert data"),
	})

	_, err := PemDecodeCert(pemData)
	require.Error(t, err)
}

// Additional tests for improved coverage

func TestBCCSPOpts_PKCS11(t *testing.T) {
	t.Run("PKCS11 Provider Without Library", func(t *testing.T) {
		// This will panic because PKCS11 is not included in build
		// We test that the panic happens as expected
		defer func() {
			r := recover()
			assert.NotNil(t, r)
			assert.Contains(t, r.(string), "pkcs11 not included")
		}()

		_, _ = BCCSPOpts("PKCS11")
		t.Fatal("Expected panic but didn't get one")
	})
}

func TestGetPKCS11BCCSP_WithKeyStore(t *testing.T) {
	t.Run("With Valid KeyStore", func(t *testing.T) {
		conf := &BCCSP{
			Default: "PKCS11",
			PKCS11: &PKCS11{
				Library:  "/usr/lib/softhsm/libsofthsm2.so",
				Label:    "test",
				Pin:      "1234",
				Security: 256,
				Hash:     "SHA2",
			},
		}

		ks := &mocks.KeyStore{}
		ks.ReadOnlyReturns(false)
		ks.GetKeyReturns(nil, errors.New("not found"))
		ks.StoreKeyReturns(nil)

		// This will panic because PKCS11 is not included in build
		defer func() {
			r := recover()
			assert.NotNil(t, r)
			// Verify no KeyStore methods were called before panic
			assert.Equal(t, 0, ks.ReadOnlyCallCount())
			assert.Equal(t, 0, ks.GetKeyCallCount())
			assert.Equal(t, 0, ks.StoreKeyCallCount())
		}()

		_, _ = GetPKCS11BCCSP(conf, ks)
		t.Fatal("Expected panic but didn't get one")
	})
}

func TestGetDefaultBCCSP_WithNilKeyStore(t *testing.T) {
	t.Run("Nil KeyStore Creates Dummy", func(t *testing.T) {
		csp, err := GetDefaultBCCSP(nil)
		require.NoError(t, err)
		assert.NotNil(t, csp)
	})
}

func TestToBCCSPOpts_EdgeCases(t *testing.T) {
	t.Run("Empty Map", func(t *testing.T) {
		opts, err := ToBCCSPOpts(map[string]interface{}{})
		require.NoError(t, err)
		assert.Nil(t, opts)
	})

	t.Run("With BCCSP Config", func(t *testing.T) {
		input := map[string]interface{}{
			"BCCSP": map[string]interface{}{
				"Default": "SW",
				"SW": map[string]interface{}{
					"Hash":     "SHA2",
					"Security": 256,
				},
			},
		}
		opts, err := ToBCCSPOpts(input)
		require.NoError(t, err)
		assert.NotNil(t, opts)
		assert.Equal(t, "SW", opts.Default)
	})
}

func TestVerifyingIdentity_Serialize(t *testing.T) {
	t.Run("Serialize With Valid Cert", func(t *testing.T) {
		// Even with empty Raw bytes, pem.EncodeToMemory doesn't fail
		// It just encodes an empty certificate
		certPEM, _ := generateTestCert(t)
		cert, err := PemDecodeCert(certPEM)
		require.NoError(t, err)

		vi := &verifyingIdentity{
			cert: cert,
		}

		// This should succeed
		serialized, err := vi.Serialize()
		require.NoError(t, err)
		assert.NotEmpty(t, serialized)
	})
}

func TestFullIdentity_SignError(t *testing.T) {
	t.Run("Sign With Invalid Hash Family", func(t *testing.T) {
		csp, err := GetDefaultBCCSP(nil)
		require.NoError(t, err)

		// Generate a key
		key, err := csp.KeyGen(&bccsp.ECDSAP256KeyGenOpts{Temporary: true})
		require.NoError(t, err)

		certPEM, priv := generateTestCert(t)
		cert, err := PemDecodeCert(certPEM)
		require.NoError(t, err)

		signer, err := NewSKIBasedSigner(csp, key.SKI(), &priv.PublicKey)
		require.NoError(t, err)

		fi := &fullIdentity{
			verifyingIdentity: &verifyingIdentity{
				bccsp:               csp,
				SignatureHashFamily: "INVALID",
				cert:                cert,
				pk:                  key,
			},
			signer: signer,
		}

		_, err = fi.Sign([]byte("test message"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hash")
	})
}

func TestFullIdentity_VerifyError(t *testing.T) {
	t.Run("Verify With Invalid Hash Family", func(t *testing.T) {
		csp, err := GetDefaultBCCSP(nil)
		require.NoError(t, err)

		key, err := csp.KeyGen(&bccsp.ECDSAP256KeyGenOpts{Temporary: true})
		require.NoError(t, err)

		certPEM, _ := generateTestCert(t)
		cert, err := PemDecodeCert(certPEM)
		require.NoError(t, err)

		fi := &fullIdentity{
			verifyingIdentity: &verifyingIdentity{
				bccsp:               csp,
				SignatureHashFamily: "INVALID",
				cert:                cert,
				pk:                  key,
			},
		}

		err = fi.Verify([]byte("message"), []byte("signature"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hash")
	})
}

func TestIdentityFactory_GetFullIdentity_Errors(t *testing.T) {
	csp, err := GetDefaultBCCSP(nil)
	require.NoError(t, err)

	factory := NewIdentityFactory(csp, bccsp.SHA2)

	t.Run("Nil SigningIdentityInfo", func(t *testing.T) {
		_, err := factory.GetFullIdentity(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("Invalid Certificate", func(t *testing.T) {
		sidInfo := &SigningIdentityInfo{
			PublicSigner: []byte("invalid cert"),
		}
		_, err := factory.GetFullIdentity(sidInfo)
		require.Error(t, err)
	})

	t.Run("Missing Private Key Material", func(t *testing.T) {
		certPEM, _ := generateTestCert(t)

		sidInfo := &SigningIdentityInfo{
			PublicSigner: certPEM,
			PrivateSigner: &KeyInfo{
				KeyIdentifier: "test",
				KeyMaterial:   nil, // No key material
			},
		}

		_, err := factory.GetFullIdentity(sidInfo)
		require.Error(t, err)
	})

	t.Run("Invalid PEM Private Key", func(t *testing.T) {
		certPEM, _ := generateTestCert(t)

		sidInfo := &SigningIdentityInfo{
			PublicSigner: certPEM,
			PrivateSigner: &KeyInfo{
				KeyIdentifier: "test",
				KeyMaterial:   []byte("not a pem key"),
			},
		}

		_, err := factory.GetFullIdentity(sidInfo)
		require.Error(t, err)
	})
}

func TestIdentityFactory_DeserializeFullIdentity_Errors(t *testing.T) {
	csp, err := GetDefaultBCCSP(nil)
	require.NoError(t, err)

	factory := NewIdentityFactory(csp, bccsp.SHA2)

	t.Run("Empty Identity", func(t *testing.T) {
		_, err := factory.DeserializeFullIdentity([]byte{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("Key Not Found in KeyStore", func(t *testing.T) {
		certPEM, _ := generateTestCert(t)

		_, err := factory.DeserializeFullIdentity(certPEM)
		require.Error(t, err)
	})
}

func TestIdentityFactory_GetIdentity_Errors(t *testing.T) {
	csp, err := GetDefaultBCCSP(nil)
	require.NoError(t, err)

	factory := NewIdentityFactory(csp, bccsp.SHA2)

	t.Run("Nil SigningIdentityInfo", func(t *testing.T) {
		_, err := factory.GetIdentity(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("Invalid Certificate", func(t *testing.T) {
		sidInfo := &SigningIdentityInfo{
			PublicSigner: []byte("invalid"),
		}
		_, err := factory.GetIdentity(sidInfo)
		require.Error(t, err)
	})
}

func TestGetIdentityFromConf_Errors(t *testing.T) {
	csp, err := GetDefaultBCCSP(nil)
	require.NoError(t, err)

	factory := NewIdentityFactory(csp, bccsp.SHA2)

	t.Run("Invalid PEM", func(t *testing.T) {
		_, _, _, err := factory.getIdentityFromConf([]byte("not a pem"))
		require.Error(t, err)
	})
}

func TestGetCertFromPem_Errors(t *testing.T) {
	csp, err := GetDefaultBCCSP(nil)
	require.NoError(t, err)

	factory := NewIdentityFactory(csp, bccsp.SHA2)

	t.Run("Nil Bytes", func(t *testing.T) {
		_, err := factory.getCertFromPem(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("Invalid PEM", func(t *testing.T) {
		_, err := factory.getCertFromPem([]byte("not pem"))
		require.Error(t, err)
	})

	t.Run("Invalid Certificate", func(t *testing.T) {
		invalidPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: []byte("invalid cert bytes"),
		})
		_, err := factory.getCertFromPem(invalidPEM)
		require.Error(t, err)
	})
}

func TestGetHashOpt_InvalidFamily(t *testing.T) {
	_, err := getHashOpt("INVALID")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not recognized")
}

// Made with Bob
