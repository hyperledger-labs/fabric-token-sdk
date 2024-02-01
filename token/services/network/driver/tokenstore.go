/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type RWSetProcessor interface {
	SetState(namespace string, key string, value []byte) error
	GetState(namespace string, key string) ([]byte, error)
	GetStateMetadata(namespace, key string) (map[string][]byte, error)
	DeleteState(namespace string, key string) error
	SetStateMetadata(namespace, key string, metadata map[string][]byte) error
}

type TokenStore interface {
	// DeleteFabToken adds to the passed rws the deletion of the passed token
	DeleteFabToken(ns string, txID string, index uint64, rws RWSetProcessor, deletedBy string) error
	StoreFabToken(ns string, txID string, index uint64, tok *token.Token, rws RWSetProcessor, infoRaw []byte, ids []string) error
	StoreIssuedHistoryToken(ns string, txID string, index uint64, tok *token.Token, rws RWSetProcessor, infoRaw []byte, issuer view.Identity, precision uint64) error
	StoreAuditToken(ns string, txID string, index uint64, tok *token.Token, rws RWSetProcessor, infoRaw []byte) error
}
