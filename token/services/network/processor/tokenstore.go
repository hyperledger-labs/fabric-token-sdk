/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package processor

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type RWSet interface {
	SetState(namespace string, key string, value []byte) error
	GetState(namespace string, key string) ([]byte, error)
	GetStateMetadata(namespace, key string) (map[string][]byte, error)
	DeleteState(namespace string, key string) error
	SetStateMetadata(namespace, key string, metadata map[string][]byte) error
}

type TokenStore interface {
	DeleteToken(ns string, txID string, index uint64, rws RWSet, deletedBy string) error
	StoreToken(ns string, txID string, index uint64, tok *token.Token, rws RWSet, tokenOnLedger []byte, tokenOnLedgerMetadata []byte, ids []string) error
	StoreIssuedHistoryToken(ns string, txID string, index uint64, tok *token.Token, rws RWSet, tokenOnLedger []byte, tokenOnLedgerMetadata []byte, issuer view.Identity, precision uint64) error
	StoreAuditToken(ns string, txID string, index uint64, tok *token.Token, rws RWSet, tokenOnLedger []byte, tokenOnLedgerMetadata []byte) error
}

const (
	AddToken    = "store-token"
	DeleteToken = "delete-token"
)

type TokenProcessorEvent struct {
	topic   string
	message TokenMessage
}

func NewTokenProcessorEvent(topic string, message *TokenMessage) *TokenProcessorEvent {
	return &TokenProcessorEvent{
		topic:   topic,
		message: *message,
	}
}

type TokenMessage struct {
	TMSID     token2.TMSID
	WalletID  string
	TokenType string
	TxID      string
	Index     uint64
}

func (t *TokenProcessorEvent) Topic() string {
	return t.topic
}

func (t *TokenProcessorEvent) Message() interface{} {
	return t.message
}
