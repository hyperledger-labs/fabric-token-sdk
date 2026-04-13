/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validation

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

// Error codes for validation failures
const (
	// Amount validation errors
	ErrCodeInvalidAmount       = "invalid-amount"
	ErrCodeNegativeAmount      = "negative-amount"
	ErrCodeZeroAmount          = "zero-amount"
	ErrCodeAmountOverflow      = "amount-overflow"
	ErrCodeAmountExceedsMax    = "amount-exceeds-max"

	// Address validation errors
	ErrCodeInvalidAddress      = "invalid-address"
	ErrCodeEmptyAddress        = "empty-address"
	ErrCodeMalformedAddress    = "malformed-address"

	// Metadata validation errors
	ErrCodeInvalidMetadata     = "invalid-metadata"
	ErrCodeMetadataTooLarge    = "metadata-too-large"
	ErrCodeMetadataTypeMismatch = "metadata-type-mismatch"

	// Token type validation errors
	ErrCodeInvalidTokenType    = "invalid-token-type"
	ErrCodeEmptyTokenType      = "empty-token-type"
	ErrCodeUnknownTokenType    = "unknown-token-type"

	// General validation errors
	ErrCodeInvalidInput        = "invalid-input"
	ErrCodeEmptyWallet         = "empty-wallet"
)

// MaxMetadataSize is the maximum size of metadata in bytes
const MaxMetadataSize = 1024 * 10 // 10KB

// MaxAddressLength is the maximum length of an address
const MaxAddressLength = 256

// InvalidAmountError indicates a token amount validation failure
type InvalidAmountError struct {
	Code    string
	Message string
	Value   interface{}
}

func (e *InvalidAmountError) Error() string {
	return errors.Errorf("%s: %s", e.Code, e.Message).Error()
}

// InvalidAddressError indicates an address validation failure
type InvalidAddressError struct {
	Code    string
	Message string
	Address interface{}
}

func (e *InvalidAddressError) Error() string {
	return errors.Errorf("%s: %s", e.Code, e.Message).Error()
}

// InvalidMetadataError indicates a metadata validation failure
type InvalidMetadataError struct {
	Code    string
	Message string
	Key     string
}

func (e *InvalidMetadataError) Error() string {
	return errors.Errorf("%s: %s", e.Code, e.Message).Error()
}

// InvalidTokenTypeError indicates a token type validation failure
type InvalidTokenTypeError struct {
	Code    string
	Message string
	Type    string
}

func (e *InvalidTokenTypeError) Error() string {
	return errors.Errorf("%s: %s", e.Code, e.Message).Error()
}

// ValidationError is a generic validation error with a code
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string {
	return errors.Errorf("%s: %s", e.Code, e.Message).Error()
}

// NewInvalidAmountError creates a new InvalidAmountError
func NewInvalidAmountError(code, message string, value interface{}) *InvalidAmountError {
	return &InvalidAmountError{Code: code, Message: message, Value: value}
}

// NewInvalidAddressError creates a new InvalidAddressError
func NewInvalidAddressError(code, message string, address interface{}) *InvalidAddressError {
	return &InvalidAddressError{Code: code, Message: message, Address: address}
}

// NewInvalidMetadataError creates a new InvalidMetadataError
func NewInvalidMetadataError(code, message, key string) *InvalidMetadataError {
	return &InvalidMetadataError{Code: code, Message: message, Key: key}
}

// NewInvalidTokenTypeError creates a new InvalidTokenTypeError
func NewInvalidTokenTypeError(code, message, tokenType string) *InvalidTokenTypeError {
	return &InvalidTokenTypeError{Code: code, Message: message, Type: tokenType}
}

// NewValidationError creates a new ValidationError
func NewValidationError(code, message string) *ValidationError {
	return &ValidationError{Code: code, Message: message}
}

// ValidateAmount validates a token amount value
func ValidateAmount(value uint64, maxValue uint64) error {
	if value == 0 {
		return NewInvalidAmountError(ErrCodeZeroAmount, "token amount must be greater than zero", value)
	}

	if maxValue > 0 && value > maxValue {
		return NewInvalidAmountError(ErrCodeAmountExceedsMax, "token amount exceeds maximum allowed value", value)
	}

	return nil
}

// ValidateAddress validates a recipient address
func ValidateAddress(address []byte) error {
	if len(address) == 0 {
		return NewInvalidAddressError(ErrCodeEmptyAddress, "address cannot be empty", nil)
	}

	if len(address) > MaxAddressLength {
		return NewInvalidAddressError(ErrCodeMalformedAddress, "address exceeds maximum length", len(address))
	}

	return nil
}

// ValidateTokenType validates a token type
func ValidateTokenType(tokenType string) error {
	if tokenType == "" {
		return NewInvalidTokenTypeError(ErrCodeEmptyTokenType, "token type cannot be empty", tokenType)
	}

	return nil
}

// ValidateMetadata validates metadata fields
func ValidateMetadata(metadata map[string]interface{}) error {
	if metadata == nil {
		return nil
	}

	for key, value := range metadata {
		if len(key) == 0 {
			return NewInvalidMetadataError(ErrCodeInvalidMetadata, "metadata key cannot be empty", key)
		}

		// Check size for byte slice values
		if bytes, ok := value.([]byte); ok {
			if len(bytes) > MaxMetadataSize {
				return NewInvalidMetadataError(ErrCodeMetadataTooLarge, "metadata value exceeds maximum size", key)
			}
		}
	}

	return nil
}

// ValidateTransferValues validates transfer values and owners
func ValidateTransferValues(values []uint64, owners [][]byte, maxValue uint64) error {
	if len(values) == 0 {
		return NewValidationError(ErrCodeInvalidInput, "values cannot be empty")
	}

	if len(owners) == 0 {
		return NewValidationError(ErrCodeInvalidInput, "owners cannot be empty")
	}

	if len(values) != len(owners) {
		return NewValidationError(ErrCodeInvalidInput, "values and owners must have the same length")
	}

	for i, v := range values {
		if err := ValidateAmount(v, maxValue); err != nil {
			return errors.Wrapf(err, "value at index %d", i)
		}
	}

	for i, o := range owners {
		if err := ValidateAddress(o); err != nil {
			return errors.Wrapf(err, "owner at index %d", i)
		}
	}

	return nil
}

// ValidateRedeemValue validates a redeem value
func ValidateRedeemValue(value uint64, maxValue uint64) error {
	return ValidateAmount(value, maxValue)
}