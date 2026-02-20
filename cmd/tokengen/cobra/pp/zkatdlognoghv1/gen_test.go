/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package zkatdlognoghv1

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd(t *testing.T) {
	cmd := Cmd()
	assert.NotNil(t, cmd)

	t.Run("trailing_args", func(t *testing.T) {
		cmd.SetArgs([]string{"extra"})
		err := cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "trailing args detected")
	})
}

func TestGen(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create dummy Idemix Issuer Public Key
	idemixDir := filepath.Join(tempDir, "idemix")
	err := os.MkdirAll(filepath.Join(idemixDir, "msp"), 0750)
	require.NoError(t, err)
	ipkPath := filepath.Join(idemixDir, "msp", "IssuerPublicKey")
	err = os.WriteFile(ipkPath, []byte("dummy ipk"), 0644)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		args := &GeneratorArgs{
			IdemixMSPDir: idemixDir,
			OutputDir:    tempDir,
			BitLength:    64,
		}
		raw, err := Gen(args)
		require.NoError(t, err)
		assert.NotNil(t, raw)
		assert.FileExists(t, filepath.Join(tempDir, "zkatdlognoghv1_pp.json"))
	})

	t.Run("success_aries", func(t *testing.T) {
		args := &GeneratorArgs{
			IdemixMSPDir: idemixDir,
			OutputDir:    tempDir,
			BitLength:    64,
			Aries:        true,
		}
		raw, err := Gen(args)
		require.NoError(t, err)
		assert.NotNil(t, raw)
	})

	t.Run("success_full", func(t *testing.T) {
		// Create dummy cert
		certDir := filepath.Join(tempDir, "cert")
		err := os.MkdirAll(filepath.Join(certDir, "signcerts"), 0750)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(certDir, "signcerts", "cert.crt"), generateZKATTestCertificate(t), 0644)
		require.NoError(t, err)

		extraFile := filepath.Join(tempDir, "extra.txt")
		err = os.WriteFile(extraFile, []byte("extra content"), 0644)
		require.NoError(t, err)

		Extras = []string{"key=" + extraFile}
		args := &GeneratorArgs{
			IdemixMSPDir: idemixDir,
			OutputDir:    tempDir,
			Issuers:      []string{certDir},
			Auditors:     []string{certDir},
			BitLength:    64,
		}
		raw, err := Gen(args)
		require.NoError(t, err)
		assert.NotNil(t, raw)
		Extras = nil // reset
	})

	t.Run("success_cc", func(t *testing.T) {
		args := &GeneratorArgs{
			IdemixMSPDir:      idemixDir,
			OutputDir:         tempDir,
			BitLength:         64,
			GenerateCCPackage: true,
		}
		raw, err := Gen(args)
		require.NoError(t, err)
		assert.NotNil(t, raw)
	})

	t.Run("success_with_version", func(t *testing.T) {
		args := &GeneratorArgs{
			IdemixMSPDir: idemixDir,
			OutputDir:    tempDir,
			BitLength:    64,
			Version:      2,
		}
		raw, err := Gen(args)
		require.NoError(t, err)
		assert.NotNil(t, raw)
		assert.FileExists(t, filepath.Join(tempDir, "zkatdlognoghv2_pp.json"))
	})
}

func generateZKATTestCertificate(t *testing.T) []byte {
	t.Helper()

	return []byte("-----BEGIN CERTIFICATE-----\n" +
		"MIICQDCCAamgAwIBAgIRAM9vjgm8Ze+1zGO/xI6uiVsFADAKBggqhkjOPQQDAzBz\n" +
		"MQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2Fu\n" +
		"IEZyYW5jaXNjbzEZMBcGA1UEChMQSHlwZXJsZWRnZXIgRmFicmljMRkwFwYDVQQD\n" +
		"ExBmYWJyaWMta2Euc2VydmVyMB4XDTI2MDIyMDA3NTY0M1oXDTI3MDIyMDA3NTY0\n" +
		"M1owBzEFMEMGA1UEAxM0Y2VydGlmaWNhdGUgd2l0aCBubyBvcmdhbml6YXRpb24g\n" +
		"dW5pdCBvciBvcmdhbml6YXRpb24wWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARL\n" +
		"B8WndVQsKgE0JWRBwj1khvVAhXWuss1/bgYnPU66gceTzsdUp54wuXBVdSa+3TYn\n" +
		"YBw5/JZVoP2JKhAH6xXFwo06MDgwDgYDVR0PAQH/BAQDAgWgMB0GA1UdJQQWMBQG\n" +
		"CCsGAQUFBwMBBggrBgEFBQcDAjAMBgNVHRMBAf8EAjAAMAoGCCqGSM49BAMDA2gA\n" +
		"MGUCMQCXz0fXfG+vY+T+I8Kj0fXfG+vY+T+I8Kj0fXfG+vY+T+I8Kj0fXfG+vY+T\n" +
		"+I8Kj0fXfG+vY+T+I8Kj0fXfG+vY+T+I8Kj0fXfG+vY+T+I8Kj0fXfG+vY+T+I8=\n" +
		"-----END CERTIFICATE-----\n")
}
