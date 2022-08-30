/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

// GetStateFnc models a function that returns the value for the given key from the ledger
type GetStateFnc = func(key string) ([]byte, error)

// Ledger models a read-only ledger
type Ledger interface {
	// GetState returns the value for the given key
	GetState(key string) ([]byte, error)
}

type SignatureProvider interface {
	// HasBeenSignedBy returns true and the verified signature if the provider contains a valid signature for the passed identity and verifier
	HasBeenSignedBy(id view.Identity, verifier Verifier) ([]byte, error)
	// Signatures returns the signatures inside this provider
	Signatures() [][]byte
}

// Validator models a token request validator
type Validator interface {
	// VerifyTokenRequestFromRaw verifies the passed marshalled token request against the passed ledger and anchor
	VerifyTokenRequestFromRaw(getState GetStateFnc, anchor string, raw []byte) ([]interface{}, error)
}
