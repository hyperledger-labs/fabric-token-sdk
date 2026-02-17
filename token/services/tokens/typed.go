/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"encoding/asn1"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/driver"
)

// TypedToken encodes a token with a type.
type TypedToken struct {
	// Type encodes the token's type
	Type driver.Type
	// Token encodes the token itself
	Token driver.Token
}

func (i TypedToken) Bytes() ([]byte, error) {
	return asn1.Marshal(i)
}

func UnmarshalTypedToken(token driver.Token) (*TypedToken, error) {
	si := &TypedToken{}
	_, err := asn1.Unmarshal(token, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedToken")
	}

	return si, nil
}

func WrapWithType(typ driver.Type, id driver.Token) (driver.Token, error) {
	raw, err := (&TypedToken{Type: typ, Token: id}).Bytes()
	if err != nil {
		return nil, err
	}

	return raw, nil
}

// TypedMetadata encodes metadata with a type.
type TypedMetadata struct {
	// Type encodes the metadata type
	Type driver.Type
	// Metadata encodes the metadata itself
	Metadata driver.Metadata
}

func (i TypedMetadata) Bytes() ([]byte, error) {
	return asn1.Marshal(i)
}

func UnmarshalTypedMetadata(metadata driver.Metadata) (*TypedMetadata, error) {
	si := &TypedMetadata{}
	_, err := asn1.Unmarshal(metadata, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedMetadata")
	}

	return si, nil
}

func WrapMetadataWithType(typ driver.Type, id driver.Metadata) (driver.Metadata, error) {
	raw, err := (&TypedMetadata{Type: typ, Metadata: id}).Bytes()
	if err != nil {
		return nil, err
	}

	return raw, nil
}
