/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

//go:generate counterfeiter -o ../mock/signing_identity.go -fake-name SigningIdentity . SigningIdentity

// signing identity
type SigningIdentity interface {
	driver.SigningIdentity
}

type WrappedSigningIdentity struct {
	Identity view.Identity
	Signer   driver.Signer
}

func (S *WrappedSigningIdentity) Serialize() ([]byte, error) {
	return S.Identity, nil
}

func (S *WrappedSigningIdentity) Verify(message []byte, signature []byte) error {
	panic("implement me")
}

func (S *WrappedSigningIdentity) Sign(raw []byte) ([]byte, error) {
	if S.Signer == nil {
		return nil, errors.New("please initialize signing identity in WrappedSigningIdentity")
	}
	return S.Signer.Sign(raw)
}
