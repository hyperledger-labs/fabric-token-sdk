/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package query

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/pkg/errors"

	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var logger = flogging.MustGetLogger("token-sdk.tms.zkat.query")

type cache interface {
	Get(key string) (interface{}, bool)
	Add(key string, value interface{})
}

type Engine struct {
	Vault     driver.Vault
	namespace string

	unspentTokensCache cache
}

func NewEngine(vault driver.Vault, namespace string, cache cache) *Engine {
	return &Engine{
		Vault:              vault,
		namespace:          namespace,
		unspentTokensCache: cache,
	}
}

func (e *Engine) IsMine(id *token.ID) (bool, error) {
	qe, err := e.Vault.NewQueryExecutor()
	if err != nil {
		return false, err
	}
	defer qe.Done()

	key, err := keys.CreateTokenMineKey(id.TxId, id.Index)
	if err != nil {
		return false, err
	}

	val, err := qe.GetState(e.namespace, key)
	if err != nil {
		return false, err
	}

	return len(val) == 1 && val[0] == 1, nil
}

func (e *Engine) UnspentTokensIteratorBy(id, typ string) (driver2.UnspentTokensIterator, error) {
	logger.Debugf("List token iterator [%s,%s]...", id, typ)
	startKey, err := keys.CreateCompositeKey(keys.FabTokenExtendedKeyPrefix, []string{id, typ})
	if err != nil {
		return nil, err
	}
	endKey := startKey + string(keys.MaxUnicodeRuneValue)

	logger.Debugf("New query executor")
	qe, err := e.Vault.NewQueryExecutor()
	if err != nil {
		return nil, err
	}
	defer qe.Done()

	logger.Debugf("Get range query scan iterator... [%s,%s]", startKey, endKey)
	iterator, err := qe.GetCachedStateRangeScanIterator(e.namespace, startKey, endKey)
	if err != nil {
		return nil, err
	}
	return &UnspentTokensIterator{it: iterator, e: e, extended: true}, nil
}

func (e *Engine) UnspentTokensIterator() (driver2.UnspentTokensIterator, error) {
	logger.Debugf("List token iterator...")
	startKey, err := keys.CreateCompositeKey(keys.FabTokenKeyPrefix, nil)
	if err != nil {
		return nil, err
	}
	endKey := startKey + string(keys.MaxUnicodeRuneValue)

	logger.Debugf("New query executor")
	qe, err := e.Vault.NewQueryExecutor()
	if err != nil {
		return nil, err
	}
	defer qe.Done()

	logger.Debugf("Get range query scan iterator... [%s,%s]", startKey, endKey)
	iterator, err := qe.GetCachedStateRangeScanIterator(e.namespace, startKey, endKey)
	if err != nil {
		return nil, err
	}
	return &UnspentTokensIterator{it: iterator, e: e}, nil
}

func (e *Engine) ListUnspentTokens() (*token.UnspentTokens, error) {
	logger.Debugf("List token...")
	startKey, err := keys.CreateCompositeKey(keys.FabTokenKeyPrefix, nil)
	if err != nil {
		return nil, err
	}
	endKey := startKey + string(keys.MaxUnicodeRuneValue)

	logger.Debugf("New query executor")
	qe, err := e.Vault.NewQueryExecutor()
	if err != nil {
		return nil, err
	}
	defer qe.Done()

	logger.Debugf("Get range query scan iterator... [%s,%s]", startKey, endKey)
	iterator, err := qe.GetCachedStateRangeScanIterator(e.namespace, startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer iterator.Close()

	logger.Debugf("scan range")
	tokens := make([]*token.UnspentToken, 0)
	for {
		next, err := iterator.Next()
		switch {
		case err != nil:
			logger.Errorf("scan failed [%s]", err)
			return nil, err

		case next == nil:
			logger.Debugf("done")
			// nil response from iterator indicates end of query results
			return &token.UnspentTokens{Tokens: tokens}, nil

		case len(next.V()) == 0:
			// logger.Debugf("nil content for key [%s]", next.K())
			continue

		default:
			logger.Debugf("parse token for key [%s]", next.K())

			output, err := UnmarshallFabtoken(next.V())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to retrieve unspent tokens for [%s]", next.K())
			}

			// show only tokens which are owned by transactor
			logger.Debugf("adding token with ID [%s] to list of unspent tokens", next.K())
			id, err := keys.GetTokenIdFromKey(next.K())
			if err != nil {
				return nil, err
			}
			// Convert quantity to decimal
			q, err := token.ToQuantity(output.Quantity, keys.Precision)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens,
				&token.UnspentToken{
					Owner:           output.Owner,
					Type:            output.Type,
					DecimalQuantity: q.Decimal(),
					Quantity:        q,
					Id:              id,
				})
		}
	}
}

func (e *Engine) ListAuditTokens(ids ...*token.ID) ([]*token.Token, error) {
	logger.Debugf("retrieve inputs for auditing...")
	qe, err := e.Vault.NewQueryExecutor()
	if err != nil {
		return nil, err
	}
	defer qe.Done()

	var res []*token.Token
	for _, id := range ids {
		idKey, err := keys.CreateAuditTokenKey(id.TxId, id.Index)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating id key [%v]", id)
		}
		tokRaw, err := qe.GetState(e.namespace, idKey)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting token for key [%v]", idKey)
		}
		if len(tokRaw) == 0 {
			return nil, errors.Errorf("token not found for key [%v]", idKey)
		}
		tok := &token.Token{}
		if err := json.Unmarshal(tokRaw, tok); err != nil {
			return nil, errors.Wrapf(err, "failed unmarshalling token for key [%v]", idKey)
		}
		res = append(res, tok)
	}
	logger.Debugf("retrieve inputs for auditing done")
	return res, nil
}

func (e *Engine) ListHistoryIssuedTokens() (*token.IssuedTokens, error) {
	logger.Debugf("History issued tokens...")
	startKey, err := keys.CreateCompositeKey(keys.IssuedHistoryTokenKeyPrefix, nil)
	if err != nil {
		return nil, err
	}
	endKey := startKey + string(keys.MaxUnicodeRuneValue)

	logger.Debugf("New query executor")
	qe, err := e.Vault.NewQueryExecutor()
	if err != nil {
		return nil, err
	}
	defer qe.Done()

	logger.Debugf("Get range query scan iterator... [%s,%s]", startKey, endKey)
	iterator, err := qe.GetStateRangeScanIterator(e.namespace, startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer iterator.Close()

	logger.Debugf("scan range")
	tokens := make([]*token.IssuedToken, 0)
	for {
		next, err := iterator.Next()
		switch {
		case err != nil:
			logger.Errorf("scan failed [%s]", err)
			return nil, err

		case next == nil:
			logger.Debugf("done")
			// nil response from iterator indicates end of query results
			return &token.IssuedTokens{Tokens: tokens}, nil

		case len(next.V()) == 0:
			logger.Debugf("nil content for key [%s]", next.K())
			continue

		default:
			logger.Debugf("parse token for key [%s]", next.K())

			output, err := UnmarshallIssuedToken(next.V())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to retrieve unspent tokens for [%s]", next.K())
			}

			// show only tokens which are owned by transactor
			logger.Debugf("adding token with ID '%s' to list of history issued tokens", next.K())
			id, err := keys.GetTokenIdFromKey(next.K())
			if err != nil {
				return nil, err
			}
			// Convert quantity to decimal
			q, err := token.ToQuantity(output.Quantity, keys.Precision)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens,
				&token.IssuedToken{
					Id:       id,
					Owner:    output.Owner,
					Type:     output.Type,
					Quantity: q.Decimal(),
					Issuer:   output.Issuer,
				})
		}
	}
}

func (e *Engine) PublicParams() ([]byte, error) {
	qe, err := e.Vault.NewQueryExecutor()
	if err != nil {
		return nil, err
	}
	defer qe.Done()

	setupKey, err := keys.CreateSetupKey()
	if err != nil {
		return nil, err
	}
	logger.Debugf("get public parameters with key [%s]", setupKey)
	raw, err := qe.GetState(e.namespace, setupKey)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (e *Engine) GetTokenInfos(ids []*token.ID, callback driver2.QueryCallbackFunc) error {
	qe, err := e.Vault.NewQueryExecutor()
	if err != nil {
		return err
	}
	defer qe.Done()
	for _, id := range ids {
		outputID, err := keys.CreateFabTokenKey(id.TxId, id.Index)
		if err != nil {
			return errors.Wrapf(err, "error creating output ID: %v", id)
		}
		meta, err := qe.GetStateMetadata(e.namespace, outputID)
		if err != nil {
			return errors.Wrapf(err, "failed getting metadata for id [%v]", id)
		}

		if err := callback(id, meta[keys.Info]); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) GetTokenCommitments(ids []*token.ID, callback driver2.QueryCallbackFunc) error {
	qe, err := e.Vault.NewQueryExecutor()
	if err != nil {
		return err
	}
	defer qe.Done()
	for _, id := range ids {
		outputID, err := keys.CreateTokenKey(id.TxId, id.Index)
		if err != nil {
			return errors.Wrapf(err, "error creating output ID: %v", id)
		}
		val, err := qe.GetState(e.namespace, outputID)
		if err != nil {
			return errors.Wrapf(err, "failed getting state for id [%v]", id)
		}

		if err := callback(id, val); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) GetTokenInfoAndCommitments(ids []*token.ID, callback driver2.QueryCallback2Func) error {
	qe, err := e.Vault.NewQueryExecutor()
	if err != nil {
		return err
	}
	defer qe.Done()
	for _, id := range ids {
		outputID, err := keys.CreateFabTokenKey(id.TxId, id.Index)
		if err != nil {
			return errors.Wrapf(err, "error creating output ID: %v", id)
		}
		meta, err := qe.GetStateMetadata(e.namespace, outputID)
		if err != nil {
			return errors.Wrapf(err, "failed getting metadata for id [%v]", id)
		}

		outputID, err = keys.CreateTokenKey(id.TxId, id.Index)
		if err != nil {
			return errors.Wrapf(err, "error creating output ID: %v", id)
		}
		val, err := qe.GetState(e.namespace, outputID)
		if err != nil {
			return errors.Wrapf(err, "failed getting state for id [%v]", id)
		}

		if err := callback(id, outputID, val, meta[keys.Info]); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) GetTokens(inputs ...*token.ID) ([]string, []*token.Token, error) {
	logger.Debugf("retrieve tokens from ids...")
	qe, err := e.Vault.NewQueryExecutor()
	if err != nil {
		return nil, nil, err
	}
	defer qe.Done()

	var res []*token.Token
	var resKeys []string
	for _, id := range inputs {
		idKey, err := keys.CreateFabTokenKey(id.TxId, id.Index)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed generating id key [%v]", id)
		}
		tokRaw, err := qe.GetState(e.namespace, idKey)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting token for key [%v]", idKey)
		}
		if len(tokRaw) == 0 {
			return nil, nil, errors.Errorf("token not found for key [%v]", idKey)
		}
		tok := &token.Token{}
		if err := json.Unmarshal(tokRaw, tok); err != nil {
			return nil, nil, errors.Wrapf(err, "failed unmarshalling token for key [%v]", idKey)
		}

		idKey, err = keys.CreateTokenKey(id.TxId, id.Index)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed generating id key [%v]", id)
		}
		resKeys = append(resKeys, idKey)
		res = append(res, tok)
	}
	logger.Debugf("retrieve tokens from ids done")
	return resKeys, res, nil
}

func (e *Engine) unmarshalUnspentToken(key string, raw []byte, extended bool) (*token.UnspentToken, error) {
	// lookup cache first
	if tok, ok := e.unspentTokensCache.Get(key); ok {
		return tok.(*token.UnspentToken), nil
	}

	// unmarshal
	output := &token.Token{}
	if err := json.Unmarshal(raw, output); err != nil {
		logger.Errorf("failed unmarshalling token for key [%v]", key)
		return nil, err
	}
	// show only tokens which are owned by transactor
	logger.Debugf("adding token with ID [%s] to list of unspent tokens", key)
	var id *token.ID
	var err error
	if extended {
		id, err = keys.GetTokenIdFromExtendedKey(key)
	} else {
		id, err = keys.GetTokenIdFromKey(key)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting token ID from key [%v]", key)
	}
	// Convert quantity to decimal
	q, err := token.ToQuantity(output.Quantity, keys.Precision)
	if err != nil {
		return nil, err
	}
	ut := &token.UnspentToken{
		Owner:           output.Owner,
		Type:            output.Type,
		DecimalQuantity: q.Decimal(),
		Quantity:        q,
		Id:              id,
	}

	// store in cache and return
	e.unspentTokensCache.Add(key, ut)
	return ut, nil
}

type UnspentTokensIterator struct {
	e        *Engine
	it       driver.Iterator
	extended bool
}

func (u *UnspentTokensIterator) Close() {
	u.it.Close()
}

func (u *UnspentTokensIterator) Next() (*token.UnspentToken, error) {
	for {
		next, err := u.it.Next()
		if err != nil {
			return nil, err
		}
		if next == nil {
			return nil, nil
		}
		if len(next.V()) == 0 {
			// TODO: remove this keys from the vault
			// logger.Debugf("nil content for key [%s]", next.K())
			continue
		}

		output, err := u.e.unmarshalUnspentToken(next.K(), next.V(), u.extended)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to retrieve unspent tokens for [%s][%s", next.K(), string(next.V()))
		}
		return output, nil
	}
}
