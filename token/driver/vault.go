/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type QueryCallbackFunc func(*token.ID, []byte) error

type QueryCallback2Func func(*token.ID, string, []byte, []byte) error

type UnspentTokensIterator interface {
	Close()
	Next() (*token.UnspentToken, error)
}

type Vault interface {
	QueryEngine() QueryEngine
}

type CertificationStorage interface {
	Exists(id *token.ID) bool
	Store(certifications map[*token.ID][]byte) error
}

type QueryEngine interface {
	// IsMine returns true if the passed id is owned by any known wallet
	IsMine(id *token.ID) (bool, error)
	// UnspentTokensIterator returns an iterator over all unspent tokens
	UnspentTokensIterator() (UnspentTokensIterator, error)
	// UnspentTokensIteratorBy returns an iterator over all unspent tokens by type and id
	UnspentTokensIteratorBy(id, typ string) (UnspentTokensIterator, error)
	// ListUnspentTokens returns the list of unspent tokens
	ListUnspentTokens() (*token.UnspentTokens, error)
	// ListAuditTokens returns the audited tokens associated to the passed ids
	ListAuditTokens(ids ...*token.ID) ([]*token.Token, error)
	// ListHistoryIssuedTokens returns the list of issues tokens
	ListHistoryIssuedTokens() (*token.IssuedTokens, error)
	// PublicParams returns the public parameters
	PublicParams() ([]byte, error)
	// GetTokenInfos retrieves the token information for the passed ids.
	// For each id, the callback is invoked to unmarshal the token information
	GetTokenInfos(ids []*token.ID, callback QueryCallbackFunc) error
	// GetTokenCommitments retrieves the token commitments for the passed ids.
	// For each id, the callback is invoked to unmarshal the token commitment
	GetTokenCommitments(ids []*token.ID, callback QueryCallbackFunc) error

	GetTokenInfoAndCommitments(ids []*token.ID, callback QueryCallback2Func) error
	// GetTokens returns the list of tokens with their respective vault keys
	GetTokens(inputs ...*token.ID) ([]string, []*token.Token, error)
}
