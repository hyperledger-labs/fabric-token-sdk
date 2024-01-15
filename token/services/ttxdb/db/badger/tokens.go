/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package badger

import (
	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func (db *Persistence) StoreOwnerToken(tr driver.TokenRecord, owners []string) error {
	panic("not implemented")
	// outputID, err := keys.CreateFabTokenKey(txID, index)
	// if err != nil {
	// 	return errors.Wrapf(err, "error creating output ID: %s", err)
	// }
	// raw, err := Marshal(tok)
	// if err != nil {
	// 	return errors.Wrapf(err, "failed to marshal token")
	// }

	// if logger.IsEnabledFor(zapcore.DebugLevel) {
	// 	logger.Debugf("transaction [%s], append fabtoken output [%s,%s,%v]", txID, outputID, view.Identity(tok.Owner.Raw), string(raw))
	// }
	// if err := rws.SetState(ns, outputID, raw); err != nil {
	// 	return err
	// }

	// meta := map[string][]byte{}
	// meta[keys.Info] = infoRaw
	// if len(ids) > 0 {
	// 	meta[IDs], err = Marshal(ids)
	// 	if err != nil {
	// 		return errors.Wrapf(err, "failed to marshal token ids")
	// 	}

	// }
	// if err := rws.SetStateMetadata(ns, outputID, meta); err != nil {
	// 	return err
	// }

	// // store extended fabtoken, if needed
	// if logger.IsEnabledFor(zapcore.DebugLevel) {
	// 	logger.Debugf("transaction [%s], append extended fabtoken output [%s,%v]", txID, outputID, ids)
	// }
	// for _, id := range ids {
	// 	if len(id) == 0 {
	// 		continue
	// 	}

	// 	outputID, err := keys.CreateExtendedFabTokenKey(id, tok.Type, txID, index)
	// 	if err != nil {
	// 		return errors.Wrapf(err, "error creating output ID: %s", err)
	// 	}
	// 	if logger.IsEnabledFor(zapcore.DebugLevel) {
	// 		logger.Debugf("transaction [%s], append extended fabtoken output [%s, %s,%s,%v]", txID, outputID, view.Identity(tok.Owner.Raw), id, string(raw))
	// 	}
	// 	if err := rws.SetState(ns, outputID, raw); err != nil {
	// 		return err
	// 	}
	// 	if err := rws.SetStateMetadata(ns, outputID, map[string][]byte{keys.Info: infoRaw}); err != nil {
	// 		return err
	// 	}

	// 	// notify others
	// 	logger.Debugf("post new event!")
	// 	cts.Notify(AddToken, cts.tmsID, id, tok.Type, txID, index)
	// }
	// return nil
}

func (db *Persistence) StoreIssuedToken(tr driver.TokenRecord) error {
	panic("not implemented")
	// outputID, err := keys.CreateIssuedHistoryTokenKey(txID, index)
	// if err != nil {
	// 	return errors.Wrapf(err, "error creating output ID: [%s,%d]", txID, index)
	// }
	// issuedToken := &token2.IssuedToken{
	// 	Id: &token2.ID{
	// 		TxId:  txID,
	// 		Index: index,
	// 	},
	// 	Owner:    tok.Owner,
	// 	Type:     tok.Type,
	// 	Quantity: tok.Quantity,
	// 	Issuer: &token2.Owner{
	// 		Raw: issuer,
	// 	},
	// }
	// raw, err := Marshal(issuedToken)
	// if err != nil {
	// 	return errors.Wrapf(err, "failed to marshal issued token")
	// }

	// q, err := token2.ToQuantity(tok.Quantity, precision)
	// if err != nil {
	// 	return errors.Wrapf(err, "invalid quantity [%s]", tok.Quantity)
	// }

	// if logger.IsEnabledFor(zapcore.DebugLevel) {
	// 	logger.Debugf("transaction [%s], append issued history token [%s,%s][%s,%v]",
	// 		txID,
	// 		tok.Type, q.Decimal(),
	// 		outputID, string(raw),
	// 	)
	// }

	// if err := rws.SetState(ns, outputID, raw); err != nil {
	// 	return err
	// }
	// if err := rws.SetStateMetadata(ns, outputID, map[string][]byte{keys.Info: infoRaw}); err != nil {
	// 	return err
	// }
	// return nil
}

func (db *Persistence) StoreAuditToken(tr driver.TokenRecord) error {
	panic("not implemented")
	// outputID, err := keys.CreateAuditTokenKey(txID, index)
	// if err != nil {
	// 	return errors.Wrapf(err, "error creating output ID: %s", err)
	// }
	// raw, err := Marshal(tok)
	// if err != nil {
	// 	return errors.Wrapf(err, "failed to marshal token")
	// }

	// if logger.IsEnabledFor(zapcore.DebugLevel) {
	// 	logger.Debugf("transaction [%s], append audit token output [%s,%v]", txID, outputID, string(raw))
	// }

	// if err := rws.SetState(ns, outputID, raw); err != nil {
	// 	return err
	// }
	// if err := rws.SetStateMetadata(ns, outputID, map[string][]byte{keys.Info: infoRaw}); err != nil {
	// 	return err
	// }
	// return nil
}

func (db *Persistence) Delete(namespace, txID string, index uint64, deletedBy string) error {
	panic("not implemented")
	// outputID, err := keys.CreateFabTokenKey(txID, index)
	// if err != nil {
	// 	return errors.Wrapf(err, "error creating output ID: %s", err)
	// }
	// if logger.IsEnabledFor(zapcore.DebugLevel) {
	// 	logger.Debugf("delete key [%s]", outputID)
	// }

	// meta, err := rws.GetStateMetadata(ns, outputID)
	// if err != nil {
	// 	return errors.Wrapf(err, "error getting metadata for key [%s]", outputID)
	// }
	// idsRaw, ok := meta[IDs]
	// if ok && len(idsRaw) > 0 {
	// 	// unmarshall ids
	// 	ids := make([]string, 0)
	// 	if err := json.Unmarshal(idsRaw, &ids); err != nil {
	// 		return errors.Wrapf(err, "error unmarshalling IDs for key [%s]", outputID)
	// 	}
	// 	// delete extended tokens as well
	// 	tokenRaw, err := rws.GetState(ns, outputID)
	// 	if err != nil {
	// 		return errors.Wrapf(err, "error getting token for key [%s]", outputID)
	// 	}
	// 	token := token2.Token{}
	// 	if err := Unmarshal(tokenRaw, &token); err != nil {
	// 		return errors.Wrapf(err, "failed to unmarshal token")
	// 	}
	// 	for _, id := range ids {
	// 		if logger.IsEnabledFor(zapcore.DebugLevel) {
	// 			logger.Debugf("delete extended key [%s]", id)
	// 		}

	// 		logger.Debugf("post new delete-token event")
	// 		cts.Notify(DeleteToken, cts.tmsID, id, token.Type, txID, index)

	// 		outputID, err := keys.CreateExtendedFabTokenKey(id, token.Type, txID, index)
	// 		if err != nil {
	// 			return errors.Wrapf(err, "error creating extendend output ID: %s", err)
	// 		}
	// 		err = rws.DeleteState(ns, outputID)
	// 		if err != nil {
	// 			return errors.Wrapf(err, "error deleting extended key [%s]", outputID)
	// 		}
	// 	}
	// }

	// err = rws.DeleteState(ns, outputID)
	// if err != nil {
	// 	return errors.Wrapf(err, "error deleting key [%s]", outputID)
	// }
	// err = rws.SetStateMetadata(ns, outputID, nil)
	// if err != nil {
	// 	return errors.Wrapf(err, "error deleting metadata for key [%s]", outputID)
	// }

	// // append a key reporting which transaction deleted this
	// deletedTokenKey, err := keys.CreateDeletedTokenKey(txID, index)
	// if err != nil {
	// 	return errors.Wrapf(err, "error creating deleted key output ID: %s", err)
	// }
	// err = rws.SetState(ns, deletedTokenKey, []byte(deletedBy))
	// if err != nil {
	// 	return errors.Wrapf(err, "failed to aadd deleted token key for key [%s]", outputID)
	// }

	// return nil
}
func (db *Persistence) IsMine(namespace, txID string, index uint64) (bool, error) {
	panic("not implemented")
}

// UnspentTokensIterator returns an iterator over all unspent tokens
func (db *Persistence) UnspentTokensIterator(namespace string) (tdriver.UnspentTokensIterator, error) {
	panic("not implemented")
}

// UnspentTokensIteratorBy returns an iterator of unspent tokens owned by the passed id and whose type is the passed on.
// The token type can be empty. In that case, tokens of any type are returned.
func (db *Persistence) UnspentTokensIteratorBy(namespace, id, typ string) (tdriver.UnspentTokensIterator, error) {
	panic("not implemented")
}

// ListUnspentTokens returns the list of unspent tokens
func (db *Persistence) ListUnspentTokens(namespace string) (*token.UnspentTokens, error) {
	panic("not implemented")
}

// ListUnspentTokensBy returns the list of unspent tokens, filtered by owner and token type
func (db *Persistence) ListUnspentTokensBy(ns, ownerEID, typ string) (*token.UnspentTokens, error) {
	panic("not implemented")
}

// ListAuditTokens returns the audited tokens associated to the passed ids
func (db *Persistence) ListAuditTokens(namespace string, ids ...*token.ID) ([]*token.Token, error) {
	panic("not implemented")
}

// ListHistoryIssuedTokens returns the list of issues tokens
func (db *Persistence) ListHistoryIssuedTokens(namespace string) (*token.IssuedTokens, error) {
	panic("not implemented")
}

// GetTokenInfos retrieves the token information for the passed ids.
// For each id, the callback is invoked to unmarshal the token information
func (db *Persistence) GetTokenInfos(namespace string, ids []*token.ID, callback tdriver.QueryCallbackFunc) error {
	panic("not implemented")
}

// GetAllTokenInfos retrieves the token information for the passed ids.
func (db *Persistence) GetAllTokenInfos(namespace string, ids []*token.ID) ([][]byte, error) {
	panic("not implemented")
}

// GetTokens returns the list of tokens with their respective vault keys
func (db *Persistence) GetTokens(namespace string, inputs ...*token.ID) ([]string, []*token.Token, error) {
	panic("not implemented")
}

// WhoDeletedTokens returns info about who deleted the passed tokens.
// The bool array is an indicator used to tell if the token at a given position has been deleted or not
func (db *Persistence) WhoDeletedTokens(namespace string, inputs ...*token.ID) ([]string, []bool, error) {
	panic("not implemented")
}
