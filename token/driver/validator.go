/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package driver

import "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

type GetStateFnc = func(key string) ([]byte, error)

type Ledger interface {
	GetState(key string) ([]byte, error)
}

type SignatureProvider interface {
	HasBeenSignedBy(id view.Identity, verifier Verifier) error
	// Signatures returns the signatures inside this provider
	Signatures() [][]byte
}

type Validator interface {
	VerifyTokenRequest(ledger Ledger, signatureProvider SignatureProvider, binding string, tr *TokenRequest) ([]interface{}, error)

	VerifyTokenRequestFromRaw(getState GetStateFnc, binding string, raw []byte) ([]interface{}, error)
}
