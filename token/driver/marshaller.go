/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	protosv1 "github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// ToTokenID converts *driver.TokenID to *token.ID
func ToTokenID(id *protosv1.TokenID) *token.ID {
	if id == nil {
		return nil
	}

	return &token.ID{
		TxId:  id.TxId,
		Index: id.Index,
	}
}

// ToProtoIdentitySlice converts []Identity to []*driver.Identity
func ToProtoIdentitySlice(identities []Identity) []*protosv1.Identity {
	res := make([]*protosv1.Identity, len(identities))
	for i, id := range identities {
		res[i] = &protosv1.Identity{
			Raw: id,
		}
	}

	return res
}

// FromProtoIdentitySlice converts []*driver.Identity to []Identity
func FromProtoIdentitySlice(identities []*protosv1.Identity) []Identity {
	res := make([]Identity, len(identities))
	for i, id := range identities {
		if id != nil {
			res[i] = id.Raw
		}
	}

	return res
}

// ToIdentity converts *driver.Identity to Identity
func ToIdentity(id *protosv1.Identity) Identity {
	if id == nil {
		return nil
	}

	return id.Raw
}
