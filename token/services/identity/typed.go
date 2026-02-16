/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"encoding/asn1"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
)

type (
	Type     = driver2.IdentityType
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
	return asn1.Marshal(i)
}

func UnmarshalTypedIdentity(id driver.Identity) (*TypedIdentity, error) {
	si := &TypedIdentity{}
	_, err := asn1.Unmarshal(id, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedIdentity")
	}

	return si, nil
}

func WrapWithType(idType Type, id driver.Identity) (driver.Identity, error) {
	raw, err := (&TypedIdentity{Type: idType, Identity: id}).Bytes()
	if err != nil {
		return nil, err
	}

	return raw, nil
}
