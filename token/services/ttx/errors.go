/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

var (
	// ErrFailedCompilingOptions signals a failure when compiling the options
	ErrFailedCompilingOptions = errors.New("failed to compiling options")
	// ErrInvalidInput signals that the input is invalid
	ErrInvalidInput = errors.New("invalid input")
)
