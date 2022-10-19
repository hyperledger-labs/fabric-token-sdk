/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package processor

import (
	"encoding/json"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = flogging.MustGetLogger("token-sdk.network.processor")

type GetStateOpt int

type RWSet interface {
	SetState(namespace string, key string, value []byte) error
	GetState(namespace string, key string) ([]byte, error)
	GetStateMetadata(namespace, key string) (map[string][]byte, error)
	DeleteState(namespace string, key string) error
	SetStateMetadata(namespace, key string, metadata map[string][]byte) error
}

type TokenStore interface {
	// DeleteFabToken adds to the passed rws the deletion of the passed token
	// TODO: we should delete also the extra tokens for the ids
	DeleteFabToken(ns string, txID string, index uint64, rws RWSet) error
	StoreFabToken(ns string, txID string, index uint64, tok *token2.Token, rws RWSet, infoRaw []byte, ids []string) error
	StoreIssuedHistoryToken(ns string, txID string, index uint64, tok *token2.Token, rws RWSet, infoRaw []byte, issuer view.Identity, precision uint64) error
	StoreAuditToken(ns string, txID string, index uint64, tok *token2.Token, rws RWSet, infoRaw []byte) error
}

type CommonTokenStore struct {
	notifier events.Publisher
}

func NewCommonTokenStore(sp view2.ServiceProvider) *CommonTokenStore {
	notifier, err := events.GetPublisher(sp)
	if err != nil {
		// TODO how to handle error here?
		logger.Warnf("cannot get notifier instance")
		// just return nil?
	}

	return &CommonTokenStore{notifier: notifier}
}

func (cts *CommonTokenStore) DeleteFabToken(ns string, txID string, index uint64, rws RWSet) error {
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
	idsRaw, ok := meta[keys.IDs]
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
		UnmarshalOrPanic(tokenRaw, &token)
		for _, id := range ids {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("delete extended key [%s]", id)
			}

			logger.Debugf("post new delete-token event")
			cts.Notify(DeleteToken, id, token.Type, txID, index)

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

	return nil
}

func (cts *CommonTokenStore) StoreFabToken(ns string, txID string, index uint64, tok *token2.Token, rws RWSet, infoRaw []byte, ids []string) error {
	outputID, err := keys.CreateFabTokenKey(txID, index)
	if err != nil {
		return errors.Wrapf(err, "error creating output ID: %s", err)
	}
	raw := MarshalOrPanic(tok)

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("transaction [%s], append fabtoken output [%s,%s,%v]", txID, outputID, view.Identity(tok.Owner.Raw), string(raw))
	}
	if err := rws.SetState(ns, outputID, raw); err != nil {
		return err
	}

	meta := map[string][]byte{}
	meta[keys.Info] = infoRaw
	if len(ids) > 0 {
		meta[keys.IDs] = MarshalOrPanic(ids)
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
		if err := rws.SetStateMetadata(ns, outputID, map[string][]byte{keys.Info: infoRaw}); err != nil {
			return err
		}

		// notify others
		logger.Debugf("post new event!")
		cts.Notify(AddToken, id, tok.Type, txID, index)
	}

	return nil
}

func (cts *CommonTokenStore) StoreIssuedHistoryToken(ns string, txID string, index uint64, tok *token2.Token, rws RWSet, infoRaw []byte, issuer view.Identity, precision uint64) error {
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
	raw := MarshalOrPanic(issuedToken)

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
	if err := rws.SetStateMetadata(ns, outputID, map[string][]byte{keys.Info: infoRaw}); err != nil {
		return err
	}
	return nil
}

func (cts *CommonTokenStore) StoreAuditToken(ns string, txID string, index uint64, tok *token2.Token, rws RWSet, infoRaw []byte) error {
	outputID, err := keys.CreateAuditTokenKey(txID, index)
	if err != nil {
		return errors.Wrapf(err, "error creating output ID: %s", err)
	}
	raw := MarshalOrPanic(tok)

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("transaction [%s], append audit token output [%s,%v]", txID, outputID, string(raw))
	}

	if err := rws.SetState(ns, outputID, raw); err != nil {
		return err
	}
	if err := rws.SetStateMetadata(ns, outputID, map[string][]byte{keys.Info: infoRaw}); err != nil {
		return err
	}
	return nil
}

func MarshalOrPanic(o interface{}) []byte {
	data, err := json.Marshal(o)
	if err != nil {
		panic(err)
	}
	return data
}

func UnmarshalOrPanic(raw []byte, o interface{}) {
	err := json.Unmarshal(raw, o)
	if err != nil {
		panic(err)
	}
}
