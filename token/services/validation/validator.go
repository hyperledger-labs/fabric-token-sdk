/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validation

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

const (
	// MaxMetadataSize is the maximum size of metadata in bytes (10KB)
	MaxMetadataSize = 10 * 1024
	// MaxAddressLength is the maximum length of an address
	MaxAddressLength = 256
)

// InvalidAmountError indicates a token amount validation failure
type InvalidAmountError struct {
	Message string
	Value   uint64
}

func (e *InvalidAmountError) Error() string {
	return e.Message
}

// InvalidAddressError indicates an address validation failure
type InvalidAddressError struct {
	Message string
	Address []byte
}

func (e *InvalidAddressError) Error() string {
	return e.Message
}

// InvalidMetadataError indicates a metadata validation failure
type InvalidMetadataError struct {
	Message string
	Key     string
}

func (e *InvalidMetadataError) Error() string {
	return e.Message
}

// InvalidTokenTypeError indicates a token type validation failure
type InvalidTokenTypeError struct {
	Message string
	Type    string
}

func (e *InvalidTokenTypeError) Error() string {
	return e.Message
}

// ValidationError is a generic validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NewInvalidAmountError creates a new InvalidAmountError
func NewInvalidAmountError(message string, value uint64) *InvalidAmountError {
	return &InvalidAmountError{Message: message, Value: value}
}

// NewInvalidAddressError creates a new InvalidAddressError
func NewInvalidAddressError(message string, address []byte) *InvalidAddressError {
	return &InvalidAddressError{Message: message, Address: address}
}

// NewInvalidMetadataError creates a new InvalidMetadataError
func NewInvalidMetadataError(message, key string) *InvalidMetadataError {
	return &InvalidMetadataError{Message: message, Key: key}
}

// NewInvalidTokenTypeError creates a new InvalidTokenTypeError
func NewInvalidTokenTypeError(message, tokenType string) *InvalidTokenTypeError {
	return &InvalidTokenTypeError{Message: message, Type: tokenType}
}

// NewValidationError creates a new ValidationError
func NewValidationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}

// ValidateAmount validates a token amount value
func ValidateAmount(value uint64, maxValue uint64) error {
	if value == 0 {
		return NewInvalidAmountError("token amount must be greater than zero", value)
	}

	if maxValue > 0 && value > maxValue {
		return NewInvalidAmountError("token amount exceeds maximum allowed value", value)
	}

	return nil
}

// ValidateAddress validates a recipient address
func ValidateAddress(address []byte) error {
	if len(address) == 0 {
		return NewInvalidAddressError("address cannot be empty", nil)
	}

	if len(address) > MaxAddressLength {
		return NewInvalidAddressError("address exceeds maximum length", address)
	}

	return nil
}

// ValidateTokenType validates a token type
func ValidateTokenType(tokenType string) error {
	if tokenType == "" {
		return NewInvalidTokenTypeError("token type cannot be empty", tokenType)
	}

	return nil
}

// ValidateMetadata validates metadata fields
func ValidateMetadata(metadata map[interface{}]interface{}) error {
	if metadata == nil {
		return nil
	}

	for key, value := range metadata {
		keyStr, isString := key.(string)
		if key == nil || (isString && keyStr == "") {
			return NewInvalidMetadataError("metadata key cannot be empty", "")
		}

		// Check size for byte slice values
		if bytes, ok := value.([]byte); ok {
			if len(bytes) > MaxMetadataSize {
				return NewInvalidMetadataError("metadata value exceeds maximum size", keyStr)
			}
		}
	}

	return nil
}

// ValidateTransferValues validates transfer values and owners
func ValidateTransferValues(values []uint64, owners [][]byte, maxValue uint64) error {
	if len(values) == 0 {
		return NewValidationError("values cannot be empty")
	}

	if len(owners) == 0 {
		return NewValidationError("owners cannot be empty")
	}

	if len(values) != len(owners) {
		return NewValidationError("values and owners must have the same length")
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
