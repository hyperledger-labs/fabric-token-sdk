/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/api"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

//go:generate counterfeiter -o ../mock/identity.go -fake-name Identity . Identity

// identity
type Identity interface {
	api.Identity
}

//go:generate counterfeiter -o ../mock/signing_identity.go -fake-name SigningIdentity . SigningIdentity

// signing identity
type SigningIdentity interface {
	api.SigningIdentity
}

type WrappedSigningIdentity struct {
	Identity view.Identity
	Signer   api.Signer
}

func (S *WrappedSigningIdentity) Serialize() ([]byte, error) {
	return S.Identity, nil
}

func (S *WrappedSigningIdentity) Verify(message []byte, signature []byte) error {
	panic("implement me")
}

func (S *WrappedSigningIdentity) Sign(raw []byte) ([]byte, error) {
	return S.Signer.Sign(raw)
}

func (S *WrappedSigningIdentity) GetPublicVersion() api.Identity {
	return S
}
