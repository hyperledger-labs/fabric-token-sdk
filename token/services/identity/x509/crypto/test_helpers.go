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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// generateTestCertWithCNAndKey is the base helper that generates a test certificate with a given CommonName
// and returns both the PEM-encoded certificate and the matching private key.
func generateTestCertWithCNAndKey(t *testing.T, cn string) ([]byte, *ecdsa.PrivateKey) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: cn,
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

// GenerateTestCertWithCN generates a test certificate with a given CommonName.
// Returns only the PEM-encoded certificate (without the private key).
// This is a convenience wrapper for tests that only need the certificate.
func GenerateTestCertWithCN(t *testing.T, cn string) []byte {
	t.Helper()
	certPEM, _ := generateTestCertWithCNAndKey(t, cn)

	return certPEM
}

// generateTestCert generates a test certificate with a default CommonName and returns both
// the PEM-encoded certificate and the matching private key.
// This is a convenience wrapper for tests that need both the certificate and key.
func GenerateTestCert(t *testing.T) ([]byte, *ecdsa.PrivateKey) {
	t.Helper()

	return generateTestCertWithCNAndKey(t, "test.example.com")
}
