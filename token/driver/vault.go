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
	CertificationStorage() CertificationStorage
}

type CertificationStorage interface {
	Exists(id *token.ID) bool
	Store(certifications map[*token.ID][]byte) error
}

type QueryEngine interface {
	// IsPending returns true if the transaction the passed id refers to is still pending, false otherwise
	IsPending(id *token.ID) (bool, error)
	// IsMine returns true if the passed id is owned by any known wallet
	IsMine(id *token.ID) (bool, error)
	// UnspentTokensIterator returns an iterator over all unspent tokens
	UnspentTokensIterator() (UnspentTokensIterator, error)
	// UnspentTokensIteratorBy returns an iterator of unspent tokens owned by the passed id and whose type is the passed on.
	// The token type can be empty. In that case, tokens of any type are returned.
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
	GetTokenInfos(ids []*token.ID) ([][]byte, error)
	// GetTokenOutputs retrieves the token output as stored on the ledger for the passed ids.
	// For each id, the callback is invoked to unmarshal the output
	GetTokenOutputs(ids []*token.ID, callback QueryCallbackFunc) error
	// GetTokenInfoAndOutputs retrieves both the token output and information for the passed ids.
	GetTokenInfoAndOutputs(ids []*token.ID) ([]string, [][]byte, [][]byte, error)
	// GetTokens returns the list of tokens with their respective vault keys
	GetTokens(inputs ...*token.ID) ([]string, []*token.Token, error)
	// WhoDeletedTokens returns info about who deleted the passed tokens.
	// The bool array is an indicator used to tell if the token at a given position has been deleted or not
	WhoDeletedTokens(inputs ...*token.ID) ([]string, []bool, error)
}
