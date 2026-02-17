/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/request"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// ToTokenID converts *request.TokenID to *token.ID
func ToTokenID(id *request.TokenID) *token.ID {
	if id == nil {
		return nil
	}

	return &token.ID{
		TxId:  id.TxId,
		Index: id.Index,
	}
}

// ToProtoIdentitySlice converts []Identity to []*request.Identity
func ToProtoIdentitySlice(identities []Identity) []*request.Identity {
	res := make([]*request.Identity, len(identities))
	for i, id := range identities {
		res[i] = &request.Identity{
			Raw: id,
		}
	}

	return res
}

// FromProtoIdentitySlice converts []*request.Identity to []Identity
func FromProtoIdentitySlice(identities []*request.Identity) []Identity {
	res := make([]Identity, len(identities))
	for i, id := range identities {
		if id != nil {
			res[i] = id.Raw
		}
	}

	return res
}

// ToIdentity converts *request.Identity to Identity
func ToIdentity(id *request.Identity) Identity {
	if id == nil {
		return nil
	}

	return id.Raw
}
