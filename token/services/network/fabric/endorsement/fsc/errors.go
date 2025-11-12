/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fsc

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

var (
	// ErrInvalidTransient signals that the transient values are invalid
	ErrInvalidTransient = errors.New("invalid transient")
	// ErrInvalidProposal signals that the proposal is invalid
	ErrInvalidProposal = errors.New("invalid proposal")
)
