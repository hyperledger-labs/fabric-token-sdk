/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rws

import (
	"encoding/json"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/processor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/keys"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = flogging.MustGetLogger("token-sdk.network.processor")

const (
	IDs = "ids"
)

type TokenStore struct {
	notifier events.Publisher
	tmsID    token.TMSID
}

func NewTokenStore(sp view2.ServiceProvider, tmsID token.TMSID) (*TokenStore, error) {
	notifier, err := events.GetPublisher(sp)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get event publisher")
	}

	return &TokenStore{notifier: notifier, tmsID: tmsID}, nil
}

func (cts *TokenStore) DeleteToken(ns string, txID string, index uint64, rws processor.RWSet, deletedBy string) error {
	outputID, err := keys.CreateFabTokenKey(txID, index)
	if err != nil {
		return errors.Wrapf(err, "error creating output ID: %s", err)
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("delete key [%s]", outputID)
	}

	meta, err := rws.GetStateMetadata(ns, outputID)
	if err != nil {
		return errors.Wrapf(err, "error getting metadata for key [%s]", outputID)
	}
	idsRaw, ok := meta[IDs]
	if ok && len(idsRaw) > 0 {
		// unmarshall ids
		ids := make([]string, 0)
		if err := json.Unmarshal(idsRaw, &ids); err != nil {
			return errors.Wrapf(err, "error unmarshalling IDs for key [%s]", outputID)
		}
		// delete extended tokens as well
		tokenRaw, err := rws.GetState(ns, outputID)
		if err != nil {
			return errors.Wrapf(err, "error getting token for key [%s]", outputID)
		}
		token := token2.Token{}
		if err := processor.Unmarshal(tokenRaw, &token); err != nil {
			return errors.Wrapf(err, "failed to unmarshal token")
		}
		for _, id := range ids {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("delete extended key [%s]", id)
			}

			logger.Debugf("post new delete-token event")
			cts.Notify(processor.DeleteToken, cts.tmsID, id, token.Type, txID, index)

			outputID, err := keys.CreateExtendedFabTokenKey(id, token.Type, txID, index)
			if err != nil {
				return errors.Wrapf(err, "error creating extendend output ID: %s", err)
			}
			err = rws.DeleteState(ns, outputID)
			if err != nil {
				return errors.Wrapf(err, "error deleting extended key [%s]", outputID)
			}
		}
	}

	err = rws.DeleteState(ns, outputID)
	if err != nil {
		return errors.Wrapf(err, "error deleting key [%s]", outputID)
	}
	err = rws.SetStateMetadata(ns, outputID, nil)
	if err != nil {
		return errors.Wrapf(err, "error deleting metadata for key [%s]", outputID)
	}

	// append a key reporting which transaction deleted this
	deletedTokenKey, err := keys.CreateDeletedTokenKey(txID, index)
	if err != nil {
		return errors.Wrapf(err, "error creating deleted key output ID: %s", err)
	}
	err = rws.SetState(ns, deletedTokenKey, []byte(deletedBy))
	if err != nil {
		return errors.Wrapf(err, "failed to aadd deleted token key for key [%s]", outputID)
	}

	return nil
}

func (cts *TokenStore) StoreToken(ns string, txID string, index uint64, tok *token2.Token, rws processor.RWSet, tokenOnLedger []byte, tokenOnLedgerMetadata []byte, ids []string) error {
	// Add a lookup key to identify quickly that this token belongs to this instance
	mineTokenID, err := keys.CreateTokenMineKey(txID, index)
	if err != nil {
		return errors.Wrapf(err, "failed computing mine key for for tx [%s]", txID)
	}
	err = rws.SetState(ns, mineTokenID, []byte{1})
	if err != nil {
		return err
	}

	// Store token
	outputID, err := keys.CreateFabTokenKey(txID, index)
	if err != nil {
		return errors.Wrapf(err, "error creating output ID: %s", err)
	}
	raw, err := processor.Marshal(tok)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal token")
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("transaction [%s], append fabtoken output [%s,%s,%v]", txID, outputID, view.Identity(tok.Owner.Raw), string(raw))
	}
	if err := rws.SetState(ns, outputID, raw); err != nil {
		return err
	}

	meta := map[string][]byte{}
	meta[keys.Info] = tokenOnLedgerMetadata
	if len(ids) > 0 {
		meta[IDs], err = processor.Marshal(ids)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal token ids")
		}

	}
	if err := rws.SetStateMetadata(ns, outputID, meta); err != nil {
		return err
	}

	// store extended fabtoken, if needed
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("transaction [%s], append extended fabtoken output [%s,%v]", txID, outputID, ids)
	}
	for _, id := range ids {
		if len(id) == 0 {
			continue
		}

		outputID, err := keys.CreateExtendedFabTokenKey(id, tok.Type, txID, index)
		if err != nil {
			return errors.Wrapf(err, "error creating output ID: %s", err)
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s], append extended fabtoken output [%s, %s,%s,%v]", txID, outputID, view.Identity(tok.Owner.Raw), id, string(raw))
		}
		if err := rws.SetState(ns, outputID, raw); err != nil {
			return err
		}
		if err := rws.SetStateMetadata(ns, outputID, map[string][]byte{keys.Info: tokenOnLedgerMetadata}); err != nil {
			return err
		}

		// notify others
		logger.Debugf("post new event!")
		cts.Notify(processor.AddToken, cts.tmsID, id, tok.Type, txID, index)
	}

	return nil
}

func (cts *TokenStore) StoreIssuedHistoryToken(ns string, txID string, index uint64, tok *token2.Token, rws processor.RWSet, tokenOnLedger []byte, tokenOnLedgerMetadata []byte, issuer view.Identity, precision uint64) error {
	outputID, err := keys.CreateIssuedHistoryTokenKey(txID, index)
	if err != nil {
		return errors.Wrapf(err, "error creating output ID: [%s,%d]", txID, index)
	}
	issuedToken := &token2.IssuedToken{
		Id: &token2.ID{
			TxId:  txID,
			Index: index,
		},
		Owner:    tok.Owner,
		Type:     tok.Type,
		Quantity: tok.Quantity,
		Issuer: &token2.Owner{
			Raw: issuer,
		},
	}
	raw, err := processor.Marshal(issuedToken)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal issued token")
	}

	q, err := token2.ToQuantity(tok.Quantity, precision)
	if err != nil {
		return errors.Wrapf(err, "invalid quantity [%s]", tok.Quantity)
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("transaction [%s], append issued history token [%s,%s][%s,%v]",
			txID,
			tok.Type, q.Decimal(),
			outputID, string(raw),
		)
	}

	if err := rws.SetState(ns, outputID, raw); err != nil {
		return err
	}
	if err := rws.SetStateMetadata(ns, outputID, map[string][]byte{keys.Info: tokenOnLedgerMetadata}); err != nil {
		return err
	}
	return nil
}

func (cts *TokenStore) StoreAuditToken(ns string, txID string, index uint64, tok *token2.Token, rws processor.RWSet, tokenOnLedger []byte, tokenOnLedgerMetadata []byte) error {
	outputID, err := keys.CreateAuditTokenKey(txID, index)
	if err != nil {
		return errors.Wrapf(err, "error creating output ID: %s", err)
	}
	raw, err := processor.Marshal(tok)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal token")
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("transaction [%s], append audit token output [%s,%v]", txID, outputID, string(raw))
	}

	if err := rws.SetState(ns, outputID, raw); err != nil {
		return err
	}
	if err := rws.SetStateMetadata(ns, outputID, map[string][]byte{keys.Info: tokenOnLedgerMetadata}); err != nil {
		return err
	}
	return nil
}

func (cts *TokenStore) Notify(topic string, tmsID token.TMSID, walletID, tokenType, txID string, index uint64) {
	if cts.notifier == nil {
		logger.Warnf("cannot notify others!")
		return
	}

	e := processor.NewTokenProcessorEvent(topic, &processor.TokenMessage{
		TMSID:     tmsID,
		WalletID:  walletID,
		TokenType: tokenType,
		TxID:      txID,
		Index:     index,
	})

	logger.Debugf("Publish new event %v", e)
	cts.notifier.Publish(e)
}
