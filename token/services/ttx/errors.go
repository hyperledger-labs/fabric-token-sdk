/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

var (
	// ErrTimeout signals that a timeout happened
	ErrTimeout = errors.New("timeout reached")
	// ErrFailedCompilingOptions signals a failure when compiling the options
	ErrFailedCompilingOptions = errors.New("failed to compiling options")
	// ErrInvalidInput signals that the input is invalid
	ErrInvalidInput = errors.New("invalid input")
	// ErrHandlingSignatureRequests signals that an error occurred while handling the signature requests
	ErrHandlingSignatureRequests = errors.New("failed to handle signature requests")
	// ErrDepNotAvailableInContext signals that a dependency is not available
	ErrDepNotAvailableInContext = errors.New("dependency not available")
	// ErrTxUnmarshalling signals that an error occurred while unmarshalling a token transaction
	ErrTxUnmarshalling = errors.New("failed to unmarshal tx")
	// ErrStorage signals a generic storage failure
	ErrStorage = errors.New("storage failure")
	// ErrSignerIdentityMismatch signals that an identity mismatch
	ErrSignerIdentityMismatch = errors.New("signer identity mismatch")
)
