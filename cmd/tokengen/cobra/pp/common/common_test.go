/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExtras(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Test case 1: Successfully load multiple files
	t.Run("success_multiple_files", func(t *testing.T) {
		// Create test files
		file1Path := filepath.Join(tempDir, "test1.json")
		file1Content := []byte(`{"key": "value1"}`)
		if err := os.WriteFile(file1Path, file1Content, 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		file2Path := filepath.Join(tempDir, "test2.json")
		file2Content := []byte(`{"key": "value2"}`)
		if err := os.WriteFile(file2Path, file2Content, 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		extraFiles := []string{
			"foo=" + file1Path,
			"bar=" + file2Path,
		}

		result, err := LoadExtras(extraFiles)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("expected 2 entries, got %d", len(result))
		}

		if !bytes.Equal(result["foo"], file1Content) {
			t.Errorf("expected %q for foo, got %q", string(file1Content), string(result["foo"]))
		}

		if !bytes.Equal(result["bar"], file2Content) {
			t.Errorf("expected %q for bar, got %q", string(file2Content), string(result["bar"]))
		}
	})

	// Test case 2: Empty input slice
	t.Run("empty_input", func(t *testing.T) {
		extraFiles := []string{}

		result, err := LoadExtras(extraFiles)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if len(result) != 0 {
			t.Errorf("expected empty map, got %d entries", len(result))
		}
	})

	// Test case 3: File does not exist
	t.Run("file_not_found", func(t *testing.T) {
		extraFiles := []string{
			"missing=" + filepath.Join(tempDir, "nonexistent.json"),
		}

		result, err := LoadExtras(extraFiles)
		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}

		if result != nil {
			t.Errorf("expected nil result on error, got: %v", result)
		}
	})

	// Test case 4: Invalid format - no colon
	t.Run("invalid_format_no_colon", func(t *testing.T) {
		extraFiles := []string{"foobar"}

		result, err := LoadExtras(extraFiles)
		if err == nil {
			t.Fatal("expected error for invalid format, got nil")
		}

		if result != nil {
			t.Errorf("expected nil result on error, got: %v", result)
		}
	})

	// Test case 5: Invalid format - empty key
	t.Run("invalid_format_empty_key", func(t *testing.T) {
		extraFiles := []string{"=" + filepath.Join(tempDir, "test.json")}

		result, err := LoadExtras(extraFiles)
		if err == nil {
			t.Fatal("expected error for empty key, got nil")
		}

		if result != nil {
			t.Errorf("expected nil result on error, got: %v", result)
		}
	})

	// Test case 6: Invalid format - empty filepath
	t.Run("invalid_format_empty_filepath", func(t *testing.T) {
		extraFiles := []string{"key="}

		result, err := LoadExtras(extraFiles)
		if err == nil {
			t.Fatal("expected error for empty filepath, got nil")
		}

		if result != nil {
			t.Errorf("expected nil result on error, got: %v", result)
		}
	})

	// Test case 7: Filepath with colons (e.g., Windows paths or URLs)
	t.Run("filepath_with_colons", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "test.json")
		fileContent := []byte("content")
		if err := os.WriteFile(filePath, fileContent, 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Simulate a key with filepath that might have colons
		extraFiles := []string{
			"mykey=" + filePath,
		}

		result, err := LoadExtras(extraFiles)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if !bytes.Equal(result["mykey"], fileContent) {
			t.Errorf("expected %q, got %q", string(fileContent), string(result["mykey"]))
		}
	})

	// Test case 8: Binary file content
	t.Run("binary_content", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "binary.dat")
		binaryContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		if err := os.WriteFile(filePath, binaryContent, 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		extraFiles := []string{"binary=" + filePath}

		result, err := LoadExtras(extraFiles)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if len(result["binary"]) != len(binaryContent) {
			t.Errorf("expected length %d, got %d", len(binaryContent), len(result["binary"]))
		}

		for i, b := range binaryContent {
			if result["binary"][i] != b {
				t.Errorf("byte mismatch at index %d: expected %x, got %x", i, b, result["binary"][i])
			}
		}
	})
}
