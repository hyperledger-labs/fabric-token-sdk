/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transfer

import "github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"

var (
	// ErrMismatchedTokensOpenings is returned when the number of tokens to be spent does not match the number of openings
	ErrMismatchedTokensOpenings = errors.New("number of tokens to be spent does not match number of openings")
	// ErrMismatchedValuesRecipients is returned when the number of values does not match the number of recipients
	ErrMismatchedValuesRecipients = errors.New("number of values does not match number of recipients")
	// ErrMismatchedTokenTypes is returned when inputs of different token types are chosen for a transfer
	ErrMismatchedTokenTypes = errors.New("cannot generate transfer: please choose inputs of the same token type")
	// ErrMismatchedRecipientsOutputs is returned when the number of recipients does not match the number of outputs
	ErrMismatchedRecipientsOutputs = errors.New("number of recipients does not match number of outputs")
	// ErrMismatchedInputsTokens is returned when the number of inputs does not match the number of input tokens
	ErrMismatchedInputsTokens = errors.New("number of inputs does not match number of input tokens")
	// ErrInvalidInputs is returned when the number of token inputs is invalid
	ErrInvalidInputs = errors.New("invalid number of token inputs, expected at least 1")
	// ErrEmptyInput is returned when an input is empty
	ErrEmptyInput = errors.New("invalid input, empty input")
	// ErrEmptyInputID is returned when an input's ID is empty
	ErrEmptyInputID = errors.New("invalid input's ID, it is empty")
	// ErrEmptyInputTxID is returned when an input's ID tx id is empty
	ErrEmptyInputTxID = errors.New("invalid input's ID, tx id is empty")
	// ErrEmptyInputToken is returned when an input's token is empty
	ErrEmptyInputToken = errors.New("invalid input's token, empty token")
	// ErrInvalidOutputs is returned when the number of token outputs is invalid
	ErrInvalidOutputs = errors.New("invalid number of token outputs, expected at least 1")
	// ErrEmptyOutputToken is returned when an output token is empty
	ErrEmptyOutputToken = errors.New("invalid output token, empty token")
	// ErrMissingIssuer is returned when an issuer is expected for a redeem action but missing
	ErrMissingIssuer = errors.New("expected issuer for a redeem action")
	// ErrInvalidVersion is returned when a transfer version is invalid
	ErrInvalidVersion = errors.New("invalid transfer version")
	// ErrInvalidTransferProof is returned when a transfer proof is invalid
	ErrInvalidTransferProof = errors.New("invalid transfer proof")
	// ErrMissingTypeAndSumProof is returned when a type-and-sum proof is missing from a transfer proof
	ErrMissingTypeAndSumProof = errors.New("invalid transfer proof: missing type-and-sum proof")
	// ErrMissingRangeProof is returned when a range proof is missing from a transfer proof
	ErrMissingRangeProof = errors.New("invalid transfer proof: missing range proof")
	// ErrInvalidTokenWitness is returned when a token witness is invalid
	ErrInvalidTokenWitness = errors.New("invalid token witness")
	// ErrInvalidTokenWitnessValue is returned when a token witness value is invalid
	ErrInvalidTokenWitnessValue = errors.New("invalid token witness value")
	// ErrInvalidSumAndTypeProof is returned when a sum and type proof is invalid
	ErrInvalidSumAndTypeProof = errors.New("invalid sum and type proof")
	// ErrMissingSumAndTypeComponents is returned when components are missing from a sum and type proof
	ErrMissingSumAndTypeComponents = errors.New("invalid sum and type proof: missing components")
	// ErrMissingSumAndTypeInputValue is returned when an input value is missing from a sum and type proof
	ErrMissingSumAndTypeInputValue = errors.New("invalid sum and type proof: missing input value")
	// ErrSumAndTypeChallengeMismatch is returned when there is a challenge mismatch in a sum and type proof
	ErrSumAndTypeChallengeMismatch = errors.New("invalid sum and type proof: challenge mismatch")
)
