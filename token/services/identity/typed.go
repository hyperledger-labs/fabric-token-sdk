/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/marshal"
)

type (
	Type     = driver.IdentityType
	Identity = driver.Identity
)

// TypedIdentity encodes an identity with a type.
type TypedIdentity struct {
	// Type encodes the type of the identity
	Type Type `json:"type,omitempty" protobuf:"bytes,1,opt,name=type,json=type,proto3"`
	// Identity encodes the identity itself
	Identity Identity `json:"identity,omitempty" protobuf:"bytes,2,opt,name=identity,proto3"`
}

func (i TypedIdentity) Bytes() ([]byte, error) {
	return marshal.EncodeIdentity(i.Type, i.Identity), nil
}

func UnmarshalTypedIdentity(id Identity) (*TypedIdentity, error) {
	res, err := marshal.DecodeIdentity(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedIdentity")
	}

	if res.IsInt {
		si := &TypedIdentity{
			Type:     res.Int32,
			Identity: res.Data,
		}

		return si, nil
	}

	return nil, errors.New("invalid identity, type not recognized")
}

func WrapWithType(idType Type, id Identity) (Identity, error) {
	raw, err := (&TypedIdentity{Type: idType, Identity: id}).Bytes()
	if err != nil {
		return nil, err
	}

	return raw, nil
}
