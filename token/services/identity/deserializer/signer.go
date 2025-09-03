/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deserializer

import (
	"context"
	errors2 "errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
)

type TypedSignerDeserializer = driver2.TypedSignerDeserializer

type TypedSignerDeserializerMultiplex struct {
	deserializers map[string][]TypedSignerDeserializer
}

func NewTypedSignerDeserializerMultiplex() *TypedSignerDeserializerMultiplex {
	return &TypedSignerDeserializerMultiplex{deserializers: map[string][]TypedSignerDeserializer{}}
}

func (v *TypedSignerDeserializerMultiplex) AddTypedSignerDeserializer(typ driver2.IdentityType, d driver2.TypedSignerDeserializer) {
	_, ok := v.deserializers[typ]
	if !ok {
		v.deserializers[typ] = []TypedSignerDeserializer{d}
		return
	}
	v.deserializers[typ] = append(v.deserializers[typ], d)
}

func (v *TypedSignerDeserializerMultiplex) DeserializeSigner(ctx context.Context, id []byte) (driver.Signer, error) {
	si, err := identity.UnmarshalTypedIdentity(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedIdentity")
	}
	dess, ok := v.deserializers[si.Type]
	if !ok {
		return nil, errors.Errorf("no deserializer found for [%s]", si.Type)
	}
	logger.DebugfContext(ctx, "deserializing [%s] with type [%s]", id, si.Type)
	var errs []error
	for _, deserializer := range dess {
		signer, err := deserializer.DeserializeSigner(ctx, si.Type, si.Identity)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return signer, nil
	}
	return nil, errors.Wrapf(errors2.Join(errs...), "failed to deserialize verifier for [%s]", si.Type)
}
