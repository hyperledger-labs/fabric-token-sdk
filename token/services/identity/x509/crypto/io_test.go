/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/test-go/testify/require"
)

func writeTempFile(t *testing.T, content []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pem")
	err := os.WriteFile(path, content, 0600)
	require.NoError(t, err)

	return path
}

func TestReadPemFile_ValidCert(t *testing.T) {
	// Generate a private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "failed to generate RSA key")

	// Create a simple certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test.local",
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}

	// Self-sign the certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err, "failed to create certificate")

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	file := writeTempFile(t, certPEM)

	_, err = readPemFile(file)
	require.NoError(t, err)
}

func TestReadPemFile_InvalidType(t *testing.T) {
	pemData := `-----BEGIN UNKNOWN-----
abcd
-----END UNKNOWN-----`
	file := writeTempFile(t, []byte(pemData))

	_, err := readPemFile(file)
	require.Error(t, err, "expected error for unknown PEM type")
}

func TestReadPemFile_NoPemContent(t *testing.T) {
	file := writeTempFile(t, []byte("not a pem file"))

	_, err := readPemFile(file)
	require.Error(t, err, "expected error for non-PEM type")
}
