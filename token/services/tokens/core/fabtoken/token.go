/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

const (
	Type driver.Type = 1
)

// Token carries the output of an action
type Token = token2.Token

// Metadata contains a serialization of the issuer of the token..
// Type, value and owner of token can be derived from the token itself.
type Metadata struct {
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
