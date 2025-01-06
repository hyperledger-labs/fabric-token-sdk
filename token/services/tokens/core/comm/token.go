/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package comm

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	Type driver.Type = 2
)

// Token encodes Type, Value, Owner
type Token struct {
	// Owner is the owner of the token
	Owner []byte
	// Data is the Pedersen commitment to type and value
	Data *math.G1
}

// Metadata contains the metadata of a token
type Metadata struct {
	// Type is the type of the token
	Type token2.Type
	// Value is the quantity of the token
	Value *math.Zr
	// BlindingFactor is the blinding factor used to commit type and value
	BlindingFactor *math.Zr
	// Owner is the owner of the token
	Owner []byte
	// Issuer is the issuer of the token, if defined
	Issuer []byte
}

func WrapTokenWithType(token driver.Token) (driver.Token, error) {
	return tokens.WrapWithType(Type, token)
}

func UnmarshalTypedToken(token driver.Token) (*tokens.TypedToken, error) {
	ttoken, err := tokens.UnmarshalTypedToken(token)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling token")
	}
	if ttoken.Type != Type {
		return nil, errors.Errorf("invalid token type [%v]", ttoken.Type)
	}
	return ttoken, nil
}

func WrapMetadataWithType(metadata driver.Metadata) (driver.Metadata, error) {
	return tokens.WrapMetadataWithType(Type, metadata)
}

func UnmarshalTypedMetadata(metadata driver.Metadata) (*tokens.TypedMetadata, error) {
	tmetadata, err := tokens.UnmarshalTypedMetadata(metadata)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling metadata")
	}
	if tmetadata.Type != Type {
		return nil, errors.Errorf("invalid metadata type [%v]", tmetadata.Type)
	}
	return tmetadata, nil
}
