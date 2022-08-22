/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// TransferOptions models the options that can be passed to the transfer command
type TransferOptions struct {
	// Attributes is a container of generic options that might be driver specific
	Attributes map[interface{}]interface{}
}

// TransferService models the token transfer service
type TransferService interface {
	// Transfer generates a TransferAction that spend the passed token ids and created the passed outputs.
	// In addition, a set of options can be specified to further customize the transfer command.
	// The function returns an TransferAction and the associated metadata.
	Transfer(txID string, wallet OwnerWallet, ids []*token2.ID, Outputs []*token2.Token, opts *TransferOptions) (TransferAction, *TransferMetadata, error)

	// VerifyTransfer checks the well-formedness of the passed TransferAction with the respect to the passed output metadata
	VerifyTransfer(tr TransferAction, tokenInfos [][]byte) error

	// DeserializeTransferAction deserializes the passed bytes into an TransferAction
	DeserializeTransferAction(raw []byte) (TransferAction, error)
}
