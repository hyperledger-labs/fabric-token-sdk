/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/request"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func ToActionSlice(actionType request.ActionType, actions [][]byte) []*request.Action {
	res := make([]*request.Action, len(actions))
	for i, action := range actions {
		res[i] = &request.Action{
			Type: actionType,
			Raw:  action,
		}
	}
	return res
}

func ToSignatureSlice(signatures [][]byte) []*request.Signature {
	res := make([]*request.Signature, len(signatures))
	for i, signature := range signatures {
		res[i] = &request.Signature{
			Raw: signature,
		}
	}
	return res
}

func ToTokenID(id *token.ID) (*request.TokenID, error) {
	if id == nil {
		return nil, nil
	}
	return &request.TokenID{
		TxId:  id.TxId,
		Index: id.Index,
	}, nil
}
