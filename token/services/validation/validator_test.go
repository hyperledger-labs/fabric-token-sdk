/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAmount(t *testing.T) {
	tests := []struct {
		name     string
		value    uint64
		maxValue uint64
		wantErr  bool
	}{
		{"zero value", 0, 1000, true},
		{"positive value within limit", 500, 1000, false},
		{"max value", 1000, 1000, false},
		{"exceeds max", 1001, 1000, true},
		{"no max limit", 1001, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAmount(tt.value, tt.maxValue)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAddress(t *testing.T) {
	tests := []struct {
		name    string
		address []byte
		wantErr bool
	}{
		{"empty address", []byte{}, true},
		{"nil address", nil, true},
		{"valid address", []byte("valid-address"), false},
		{"address too long", make([]byte, MaxAddressLength+1), true},
		{"max length address", make([]byte, MaxAddressLength), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAddress(tt.address)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTokenType(t *testing.T) {
	tests := []struct {
		name      string
		tokenType string
		wantErr   bool
	}{
		{"empty type", "", true},
		{"valid type", "USD", false},
		{"valid type EUR", "EUR", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTokenType(tt.tokenType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMetadata(t *testing.T) {
	t.Run("nil metadata", func(t *testing.T) {
		err := ValidateMetadata(nil)
		assert.NoError(t, err)
	})

	t.Run("empty metadata", func(t *testing.T) {
		err := ValidateMetadata(map[interface{}]interface{}{})
		assert.NoError(t, err)
	})

	t.Run("valid metadata with byte values", func(t *testing.T) {
		metadata := map[interface{}]interface{}{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
		}
		err := ValidateMetadata(metadata)
		assert.NoError(t, err)
	})

	t.Run("valid metadata with non-byte values", func(t *testing.T) {
		metadata := map[interface{}]interface{}{
			"key1": "string value",
			"key2": 12345,
		}
		err := ValidateMetadata(metadata)
		assert.NoError(t, err)
	})

	t.Run("empty key", func(t *testing.T) {
		metadata := map[interface{}]interface{}{
			"": []byte("value"),
		}
		err := ValidateMetadata(metadata)
		assert.Error(t, err)
	})

	t.Run("value too large", func(t *testing.T) {
		metadata := map[interface{}]interface{}{
			"key1": make([]byte, MaxMetadataSize+1),
		}
		err := ValidateMetadata(metadata)
		assert.Error(t, err)
	})
}

func TestValidateTransferValues(t *testing.T) {
	t.Run("empty values", func(t *testing.T) {
		err := ValidateTransferValues([]uint64{}, [][]byte{[]byte("owner")}, 1000)
		assert.Error(t, err)
	})

	t.Run("empty owners", func(t *testing.T) {
		err := ValidateTransferValues([]uint64{100}, [][]byte{}, 1000)
		assert.Error(t, err)
	})

	t.Run("mismatched lengths", func(t *testing.T) {
		err := ValidateTransferValues([]uint64{100, 200}, [][]byte{[]byte("owner1")}, 1000)
		assert.Error(t, err)
	})

	t.Run("zero value", func(t *testing.T) {
		err := ValidateTransferValues([]uint64{0}, [][]byte{[]byte("owner")}, 1000)
		assert.Error(t, err)
	})

	t.Run("exceeds max value", func(t *testing.T) {
		err := ValidateTransferValues([]uint64{1001}, [][]byte{[]byte("owner")}, 1000)
		assert.Error(t, err)
	})

	t.Run("empty owner", func(t *testing.T) {
		err := ValidateTransferValues([]uint64{100}, [][]byte{{}}, 1000)
		assert.Error(t, err)
	})

	t.Run("valid transfer", func(t *testing.T) {
		err := ValidateTransferValues([]uint64{100, 200}, [][]byte{[]byte("owner1"), []byte("owner2")}, 1000)
		assert.NoError(t, err)
	})
}

func TestValidateRedeemValue(t *testing.T) {
	tests := []struct {
		name     string
		value    uint64
		maxValue uint64
		wantErr  bool
	}{
		{"zero value", 0, 1000, true},
		{"positive value", 100, 1000, false},
		{"exceeds max", 1001, 1000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRedeemValue(tt.value, tt.maxValue)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestErrorTypes(t *testing.T) {
	t.Run("InvalidAmountError", func(t *testing.T) {
		err := NewInvalidAmountError("test message")
		assert.Contains(t, err.Error(), "test message")
	})

	t.Run("InvalidAddressError", func(t *testing.T) {
		err := NewInvalidAddressError("test message")
		assert.Contains(t, err.Error(), "test message")
	})

	t.Run("InvalidMetadataError", func(t *testing.T) {
		err := NewInvalidMetadataError("test message")
		assert.Contains(t, err.Error(), "test message")
	})

	t.Run("InvalidTokenTypeError", func(t *testing.T) {
		err := NewInvalidTokenTypeError("test message")
		assert.Contains(t, err.Error(), "test message")
	})

	t.Run("ValidationError", func(t *testing.T) {
		err := NewValidationError("test message")
		assert.Contains(t, err.Error(), "test message")
	})
}
