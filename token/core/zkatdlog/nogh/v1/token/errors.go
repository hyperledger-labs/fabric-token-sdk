/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import "github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"

var (
	// ErrEmptyType is returned when a token type is empty
	ErrEmptyType = errors.New("missing Type")
	// ErrEmptyValue is returned when a token value is nil
	ErrEmptyValue = errors.New("missing Value")
	// ErrEmptyBlindingFactor is returned when a token blinding factor is nil
	ErrEmptyBlindingFactor = errors.New("missing BlindingFactor")
	// ErrMissingIssuer is returned when an issuer is required but missing
	ErrMissingIssuer = errors.New("missing Issuer")
	// ErrUnexpectedIssuer is returned when an issuer is present but should not be
	ErrUnexpectedIssuer = errors.New("issuer should not be there")
	// ErrEmptyOwner is returned when a token owner is empty
	ErrEmptyOwner = errors.New("token owner cannot be empty")
	// ErrEmptyTokenData is returned when token data is nil
	ErrEmptyTokenData = errors.New("token data cannot be empty")
	// ErrNilCommitElement is returned when trying to commit a nil element
	ErrNilCommitElement = errors.New("cannot commit a nil element")
	// ErrTokenMismatch is returned when a token does not match its metadata
	ErrTokenMismatch = errors.New("cannot retrieve token in the clear: output does not match provided opening")
	// ErrMissingFabToken is returned when FabToken is nil in UpgradeWitness
	ErrMissingFabToken = errors.New("missing FabToken")
	// ErrMissingFabTokenOwner is returned when FabToken owner is empty
	ErrMissingFabTokenOwner = errors.New("missing FabToken.Owner")
	// ErrMissingFabTokenType is returned when FabToken type is empty
	ErrMissingFabTokenType = errors.New("missing FabToken.Type")
	// ErrMissingFabTokenQuantity is returned when FabToken quantity is empty
	ErrMissingFabTokenQuantity = errors.New("missing FabToken.Quantity")
	// ErrMissingUpgradeBlindingFactor is returned when upgrade blinding factor is nil
	ErrMissingUpgradeBlindingFactor = errors.New("missing BlindingFactor")
)
