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
	// Selector is the custom token selector to use. If nil, the default will be used.
	// tod add Selector Selector
	// TokenIDs to transfer. If empty, the tokens will be selected.
	TokenIDs []*token2.ID
}

type TransferService interface {
	Transfer(txID string, wallet OwnerWallet, ids []*token2.ID, Outputs []*token2.Token, opts *TransferOptions) (TransferAction, *TransferMetadata, error)

	VerifyTransfer(tr TransferAction, tokenInfos [][]byte) error

	DeserializeTransferAction(raw []byte) (TransferAction, error)
}
