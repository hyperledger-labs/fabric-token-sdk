/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

//go:generate counterfeiter -o ../mock/signing_identity.go -fake-name SigningIdentity . SigningIdentity

type SigningIdentity interface {
	driver.SigningIdentity
}

type WrappedSigningIdentity struct {
	Identity driver.Identity
	Signer   driver.Signer
}

func (w *WrappedSigningIdentity) Serialize() ([]byte, error) {
	return w.Identity, nil
}

func (w *WrappedSigningIdentity) Sign(raw []byte) ([]byte, error) {
	if w.Signer == nil {
		return nil, errors.New("please initialize signing identity in WrappedSigningIdentity")
	}

	return w.Signer.Sign(raw)
}
