/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type QueryCallbackFunc func(*token.Id, []byte) error

type Vault interface {
	QueryEngine() QueryEngine
}

type CertificationStorage interface {
	Exists(id *token.Id) bool
	Store(certifications map[*token.Id][]byte) error
}

type QueryEngine interface {
	IsMine(id *token.Id) (bool, error)
	ListUnspentTokens() (*token.UnspentTokens, error)
	ListAuditTokens(ids ...*token.Id) ([]*token.Token, error)
	ListHistoryIssuedTokens() (*token.IssuedTokens, error)
	PublicParams() ([]byte, error)
	GetTokenInfos(ids []*token.Id, callback QueryCallbackFunc) error
	GetTokenCommitments(ids []*token.Id, callback QueryCallbackFunc) error
	GetTokens(inputs ...*token.Id) ([]*token.Token, error)
}
