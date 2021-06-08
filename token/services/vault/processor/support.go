/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package processor

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func (r *RWSetProcessor) deleteFabToken(ns string, txID string, index int, rws *fabric.RWSet) error {
	outputID, err := keys.CreateFabtokenKey(txID, index)
	if err != nil {
		return errors.Wrapf(err, "error creating output ID: %s", err)
	}
	logger.Debugf("delete key [%s]", outputID)
	err = rws.DeleteState(ns, outputID)
	if err != nil {
		return err
	}
	err = rws.SetStateMetadata(ns, outputID, nil)
	if err != nil {
		return err
	}
	return nil
}

func (r *RWSetProcessor) storeFabToken(ns string, txID string, index int, tok *token2.Token, rws *fabric.RWSet, infoRaw []byte) error {
	outputID, err := keys.CreateFabtokenKey(txID, index)
	if err != nil {
		return errors.Wrapf(err, "error creating output ID: %s", err)
	}
	raw := MarshalOrPanic(tok)

	logger.Debugf("transaction [%s], append fabtoken output [%s,%v]", txID, outputID, string(raw))

	if err := rws.SetState(ns, outputID, raw); err != nil {
		return err
	}
	if err := rws.SetStateMetadata(ns, outputID, map[string][]byte{keys.Info: infoRaw}); err != nil {
		return err
	}
	return nil
}

func (r *RWSetProcessor) storeIssuedHistoryToken(ns string, txID string, index int, tok *token2.Token, rws *fabric.RWSet, infoRaw []byte, issuer view.Identity) error {
	outputID, err := keys.CreateIssuedHistoryTokenKey(txID, index)
	if err != nil {
		return errors.Wrapf(err, "error creating output ID: [%s,%d]", txID, index)
	}
	issuedToken := &token2.IssuedToken{
		Id: &token2.Id{
			TxId:  txID,
			Index: uint32(index),
		},
		Owner:    tok.Owner,
		Type:     tok.Type,
		Quantity: tok.Quantity,
		Issuer: &token2.Owner{
			Raw: issuer,
		},
	}
	raw := MarshalOrPanic(issuedToken)

	q, err := token2.ToQuantity(tok.Quantity, 64)
	if err != nil {
		return errors.Wrapf(err, "invalid quantity [%s]", tok.Quantity)
	}

	logger.Debugf("transaction [%s], append issued history token [%s,%s][%s,%v]",
		txID,
		tok.Type, q.Decimal(),
		outputID, string(raw),
	)

	if err := rws.SetState(ns, outputID, raw); err != nil {
		return err
	}
	if err := rws.SetStateMetadata(ns, outputID, map[string][]byte{keys.Info: infoRaw}); err != nil {
		return err
	}
	return nil
}

func (r *RWSetProcessor) storeAuditToken(ns string, txID string, index int, tok *token2.Token, rws *fabric.RWSet, infoRaw []byte) error {
	outputID, err := keys.CreateAuditTokenKey(txID, index)
	if err != nil {
		return errors.Wrapf(err, "error creating output ID: %s", err)
	}
	raw := MarshalOrPanic(tok)

	logger.Debugf("transaction [%s], append audit token output [%s,%v]", txID, outputID, string(raw))

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
