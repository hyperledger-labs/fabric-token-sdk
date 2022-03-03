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

type TransferService interface {
	// Transfer generates a TransferAction whose tokens are transferred by the passed wallet.
	// The tokens to be transferred are passed as token IDs.
	// In addition, a set of options can be specified to further customize the transfer command
	Transfer(txID string, wallet OwnerWallet, ids []*token2.ID, Outputs []*token2.Token, Opts ...*TransferOptions) (TransferAction, *TransferMetadata, error)

	VerifyTransfer(tr TransferAction, tokenInfos [][]byte) error

	DeserializeTransferAction(raw []byte) (TransferAction, error)
}
