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

func TestUpdateCmd(t *testing.T) {
	cmd := UpdateCmd()
	assert.NotNil(t, cmd)

	t.Run("trailing_args", func(t *testing.T) {
		cmd.SetArgs([]string{"extra"})
		err := cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "trailing args detected")
	})
}

func TestUpdate(t *testing.T) {
	wd, _ := os.Getwd()
	testdataPath := filepath.Join(wd, "..", "..", "..", "testdata", "zkatdlognoghv1_pp.json")

	t.Run("success", func(t *testing.T) {
		tempDir := t.TempDir()
		args := &UpdateArgs{
			InputFile: testdataPath,
			OutputDir: tempDir,
		}
		err := Update(args)
		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(tempDir, "zkatdlognoghv1_pp.json"))
	})

	t.Run("fail_read_input", func(t *testing.T) {
		tempDir := t.TempDir()
		args := &UpdateArgs{
			InputFile: "nonexistent",
			OutputDir: tempDir,
		}
		err := Update(args)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read input file")
	})

	t.Run("success_full", func(t *testing.T) {
		tempDir := t.TempDir()
		// Create dummy cert
		certDir := filepath.Join(tempDir, "cert")
		err := os.MkdirAll(filepath.Join(certDir, "signcerts"), 0750)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(certDir, "signcerts", "cert.crt"), generateUpdateTestCertificate(t), 0644)
		require.NoError(t, err)

		args := &UpdateArgs{
			InputFile: testdataPath,
			OutputDir: tempDir,
			Auditors:  []string{certDir},
			Issuers:   []string{certDir},
		}
		err = Update(args)
		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(tempDir, "zkatdlognoghv1_pp.json"))
	})

	t.Run("panic_recover", func(t *testing.T) {
		// This should trigger a panic when accessing args.InputFile and be caught by recover
		_ = Update(nil)
	})
}

func generateUpdateTestCertificate(t *testing.T) []byte {
	t.Helper()
	// Simple PEM certificate for testing
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
