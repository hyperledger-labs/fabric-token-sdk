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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestReadFile(t *testing.T) {
	// Test reading the content of a file
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

	// Test failure to read from a non-existent file
	t.Run("Non-existent File", func(t *testing.T) {
		_, err := readFile("/non/existent/file")
		require.Error(t, err)
	})
}

func TestGetPemMaterialFromDir(t *testing.T) {
	// Test loading crypto material from a file in a given folder
	t.Run("Valid Directory", func(t *testing.T) {
		dir := t.TempDir()
		certPEM, _ := GenerateTestCert(t)

		// Write a cert file
		err := os.WriteFile(filepath.Join(dir, "cert.pem"), certPEM, 0600)
		require.NoError(t, err)

		materials, err := getPemMaterialFromDir(dir)
		require.NoError(t, err)
		assert.Len(t, materials, 1)
	})

	// Test failure to load crypto material from a non-existent folder
	t.Run("Non-existent Directory", func(t *testing.T) {
		_, err := getPemMaterialFromDir("/non/existent/dir")
		require.Error(t, err)
	})

	// Test failure to load crypto material from an empty folder
	t.Run("Empty Directory", func(t *testing.T) {
		dir := t.TempDir()
		materials, err := getPemMaterialFromDir(dir)
		require.NoError(t, err)
		assert.Empty(t, materials)
	})

	// Test failure to load crypto material from a folder with no valid pem files
	t.Run("Directory With Non-PEM Files", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("not pem"), 0600)
		require.NoError(t, err)

		materials, err := getPemMaterialFromDir(dir)
		require.NoError(t, err)
		assert.Empty(t, materials)
	})
}

// Test failure in attempt to read a pem file from a file that includes invalid pem data
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
