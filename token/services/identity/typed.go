/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"encoding/asn1"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

// TypedIdentity encodes an identity with a type.
type TypedIdentity struct {
	// Type encodes the type of the identity
	Type string `protobuf:"bytes,1,opt,name=type,json=type,proto3" json:"type,omitempty"`
	// Identity encodes the identity itself
	Identity []byte `protobuf:"bytes,2,opt,name=identity,proto3" json:"identity,omitempty"`
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

func WrapWithType(idType string, id driver.Identity) (driver.Identity, error) {
	raw, err := (&TypedIdentity{Type: idType, Identity: id}).Bytes()
	if err != nil {
		return nil, err
	}
	return raw, nil
}
