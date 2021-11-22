/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	tokenapi "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type Verifier interface {
	Verify(message, sigma []byte) error
}

type Ledger interface {
	GetState(key string) ([]byte, error)
}

type SignatureProvider interface {
	HasBeenSignedBy(id view.Identity, verifier Verifier) error
}

type Validator struct {
	backend tokenapi.Validator
}

func (c *Validator) Verify(ledger Ledger, sp SignatureProvider, binding string, tr *Request) ([]interface{}, error) {
	actions, err := c.backend.VerifyTokenRequest(ledger, &signatureProvider{sp: sp}, binding, tr.Actions)
	if err != nil {
		return nil, err
	}

	var res []interface{}
	for _, action := range actions {
		res = append(res, action)
	}
	return res, nil
}

func (c *Validator) UnmarshallAndVerify(ledger Ledger, binding string, raw []byte) ([]interface{}, error) {
	actions, err := c.backend.VerifyTokenRequestFromRaw(func(key string) ([]byte, error) {
		return ledger.GetState(key)
	}, binding, raw)
	if err != nil {
		return nil, err
	}

	var res []interface{}
	for _, action := range actions {
		res = append(res, action)
	}
	return res, nil
}

type signatureProvider struct {
	sp SignatureProvider
}

func (s *signatureProvider) HasBeenSignedBy(id view.Identity, v tokenapi.Verifier) error {
	return s.sp.HasBeenSignedBy(id, &verifier{v: v})
}

func (s *signatureProvider) Signatures() [][]byte {
	return s.Signatures()
}

type verifier struct {
	v Verifier
}

func (v *verifier) Verify(message, sigma []byte) error {
	return v.v.Verify(message, sigma)
}
