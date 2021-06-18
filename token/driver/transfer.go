/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package driver

import (
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TransferService interface {
	Transfer(txID string, wallet OwnerWallet, ids []*token2.Id, Outputs ...*token2.Token) (TransferAction, *TransferMetadata, error)

	VerifyTransfer(tr TransferAction, tokenInfos [][]byte) error

	DeserializeTransferAction(raw []byte) (TransferAction, error)
}
