/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storage

import "github.com/hyperledger-labs/fabric-token-sdk/token"

type DBEntry struct {
	TMSID    token.TMSID
	WalletID string
}

type DBEntriesStorage interface {
	Put(tmsID token.TMSID, walletID string) error
	Iterator() (Iterator[*DBEntry], error)
}
