/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	vdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenQueryEngine struct {
	Vault     vdriver.Vault
	namespace string
	db        *ttxdb.DB
}

func NewEngine(namespace string, db *ttxdb.DB) *TokenQueryEngine {
	return &TokenQueryEngine{
		// Vault:     vault,
		namespace: namespace,
		db:        db,
	}
}

// IsPending returns true if the transaction the passed id refers to is still pending, false otherwise
func (tqe *TokenQueryEngine) IsPending(id *token.ID) (bool, error) {
	// TODO
	vc, err := tqe.Vault.TransactionStatus(id.TxId)
	if err != nil {
		return false, err
	}
	return vc == vdriver.Busy, nil
}

// IsMine returns true if the passed id is owned by any known wallet
func (tqe *TokenQueryEngine) IsMine(id *token.ID) (bool, error) {
	return tqe.db.IsMine(tqe.namespace, id.TxId, id.Index)
}

// UnspentTokensIterator returns an iterator over all unspent tokens
func (tqe *TokenQueryEngine) UnspentTokensIterator() (driver.UnspentTokensIterator, error) {
	return tqe.db.UnspentTokensIterator(tqe.namespace)
}

// UnspentTokensIteratorBy returns an iterator of unspent tokens owned by the passed id and whose type is the passed on.
// The token type can be empty. In that case, tokens of any type are returned.
func (tqe *TokenQueryEngine) UnspentTokensIteratorBy(id, typ string) (driver.UnspentTokensIterator, error) {
	return tqe.db.UnspentTokensIteratorBy(tqe.namespace, id, typ)
}

// ListUnspentTokens returns the list of unspent tokens
func (tqe *TokenQueryEngine) ListUnspentTokens() (*token.UnspentTokens, error) {
	return tqe.db.ListUnspentTokens(tqe.namespace)
}

// ListUnspentTokensBy returns the list of unspent tokens filtered by enrollment id and token type
func (tqe *TokenQueryEngine) ListUnspentTokensBy(id, typ string) (*token.UnspentTokens, error) {
	return tqe.db.ListUnspentTokensBy(tqe.namespace, id, typ)
}

// ListAuditTokens returns the audited tokens associated to the passed ids
func (tqe *TokenQueryEngine) ListAuditTokens(ids ...*token.ID) ([]*token.Token, error) {
	return tqe.db.ListAuditTokens(tqe.namespace, ids...)
}

// ListHistoryIssuedTokens returns the list of issues tokens
func (tqe *TokenQueryEngine) ListHistoryIssuedTokens() (*token.IssuedTokens, error) {
	return tqe.db.ListHistoryIssuedTokens(tqe.namespace)
}

// PublicParams returns the public parameters
func (tqe *TokenQueryEngine) PublicParams() ([]byte, error) {
	return tqe.db.GetRawPublicParams()
}

// GetTokenInfos retrieves the token information for the passed ids.
// For each id, the callback is invoked to unmarshal the token information
func (tqe *TokenQueryEngine) GetTokenInfos(ids []*token.ID, callback driver.QueryCallbackFunc) error {
	return tqe.db.GetTokenInfos(tqe.namespace, ids, callback)
}

// GetTokenOutputs retrieves the token output as stored on the ledger for the passed ids.
func (tqe *TokenQueryEngine) GetTokenOutputs(ids []*token.ID, callback driver.QueryCallbackFunc) error {
	// TODO
	qe, err := tqe.Vault.NewQueryExecutor()
	if err != nil {
		return err
	}
	defer qe.Done()

	for _, id := range ids {
		outputID, err := keys.CreateTokenKey(id.TxId, id.Index)
		if err != nil {
			return errors.Wrapf(err, "error creating output ID: %v", id)
		}
		val, err := qe.GetState(tqe.namespace, outputID)
		if err != nil {
			return errors.Wrapf(err, "failed getting state for id [%v]", id)
		}
		if err := callback(id, val); err != nil {
			return err
		}
	}
	return nil
}

// GetTokenInfoAndOutputs retrieves both the token output and information for the passed ids.
func (tqe *TokenQueryEngine) GetTokenInfoAndOutputs(ids []*token.ID, callback driver.QueryCallback2Func) error {
	// TODO
	qe, err := tqe.Vault.NewQueryExecutor()
	if err != nil {
		return err
	}
	defer qe.Done()

	// get info from database
	infos, err := tqe.db.GetAllTokenInfos(tqe.namespace, ids)
	if err != nil {
		return err
	}

	// The actual token, as stored by the tokenchaincode, is in the vault
	for i, id := range ids {
		outputID, err := keys.CreateTokenKey(id.TxId, id.Index)
		if err != nil {
			return errors.Wrapf(err, "error creating output ID: %v", id)
		}
		val, err := qe.GetState(tqe.namespace, outputID)
		if err != nil {
			return errors.Wrapf(err, "failed getting state for id [%v]", id)
		}

		if err := callback(id, outputID, val, infos[i]); err != nil {
			return err
		}
	}
	return nil
}

// GetTokens returns the list of tokens with their respective vault keys
func (tqe *TokenQueryEngine) GetTokens(inputs ...*token.ID) ([]string, []*token.Token, error) {
	return tqe.db.GetTokens(tqe.namespace, inputs...)
}

// WhoDeletedTokens returns info about who deleted the passed tokens.
// The bool array is an indicator used to tell if the token at a given position has been deleted or not
func (tqe *TokenQueryEngine) WhoDeletedTokens(inputs ...*token.ID) ([]string, []bool, error) {
	return tqe.db.WhoDeletedTokens(tqe.namespace, inputs...)
}
