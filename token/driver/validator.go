/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// ValidationAttributeID is the type of validation attribute identifier
type ValidationAttributeID = string

// ValidationAttributes is a map containing attributes generated during validation
type ValidationAttributes = map[ValidationAttributeID][]byte

// GetStateFnc models a function that returns the value for the given key from the ledger
type GetStateFnc = func(id token.ID) ([]byte, error)

// Ledger models a read-only ledger
type Ledger interface {
	// GetState returns the value for the given key
	GetState(id token.ID) ([]byte, error)
}

//go:generate counterfeiter -o mock/signature_provider.go -fake-name SignatureProvider . SignatureProvider

type SignatureProvider interface {
	// HasBeenSignedBy returns true and the verified signature if the provider contains a valid signature for the passed identity and verifier
	HasBeenSignedBy(c context.Context, id Identity, verifier Verifier) ([]byte, error)
	// Signatures returns the signatures inside this provider
	Signatures() [][]byte
}

//go:generate counterfeiter -o mock/validator.go -fake-name Validator . Validator
//go:generate counterfeiter -o mock/validator_ledger.go -fake-name ValidatorLedger . ValidatorLedger

type ValidatorLedger interface {
	GetState(id token.ID) ([]byte, error)
}

// Validator models a token request validator
type Validator interface {
	// UnmarshalActions returns the actions contained in the serialized token request
	UnmarshalActions(raw []byte) ([]interface{}, error)
	// VerifyTokenRequestFromRaw verifies the passed marshalled token request against the passed ledger and anchor.
	// The function returns additionally a map that contains information about the token request. The content of this map
	// is driver-dependant
	VerifyTokenRequestFromRaw(ctx context.Context, getState GetStateFnc, anchor TokenRequestAnchor, raw []byte) ([]interface{}, ValidationAttributes, error)
}
