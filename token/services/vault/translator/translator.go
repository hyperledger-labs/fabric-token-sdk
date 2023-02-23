/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package translator

import (
	"crypto/sha256"
	"encoding/base64"
	"strconv"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.vault.translator")

// Translator validates token requests and generates the corresponding RWSets
type Translator struct {
	RWSet RWSet
	TxID  string
	// SpentIDs the spent IDs added so far
	SpentIDs [][]byte

	counter   uint64
	namespace string
}

func New(txID string, rwSet RWSet, namespace string) *Translator {
	w := &Translator{
		RWSet:     rwSet,
		TxID:      txID,
		counter:   0,
		namespace: namespace,
	}

	return w
}

// Write checks that transactions are correct wrt. the most recent rwset state.
// Write checks are ones that shall be done sequentially, since transactions within a block may introduce dependencies.
func (w *Translator) Write(action interface{}) error {
	logger.Debugf("checking transaction with txID '%s'", w.TxID)

	err := w.checkProcess(action)
	if err != nil {
		return err
	}

	logger.Debugf("committing transaction with txID '%s'", w.TxID)
	err = w.commitProcess(action)
	if err != nil {
		logger.Errorf("error committing transaction with txID '%s': %s", w.TxID, err)
		return err
	}
	logger.Debugf("successfully processed transaction with txID '%s'", w.TxID)
	return nil
}

func (w *Translator) CommitTokenRequest(raw []byte, storeHash bool) error {
	key, err := keys.CreateTokenRequestKey(w.TxID)
	if err != nil {
		return errors.Errorf("can't create for token request '%s'", w.TxID)
	}
	tr, err := w.RWSet.GetState(w.namespace, key)
	if err != nil {
		return errors.Wrapf(err, "failed to read token request'%s'", w.TxID)
	}
	if len(tr) != 0 {
		return errors.Wrapf(errors.New("token request with same ID already exists"), "failed to write token request'%s'", w.TxID)
	}
	if storeHash {
		hash := sha256.New()
		n, err := hash.Write(raw)
		if n != len(raw) {
			return errors.Errorf("failed to write token request, hash failure '%s'", w.TxID)
		}
		if err != nil {
			return errors.Wrapf(err, "failed to write token request, hash failure '%s'", w.TxID)
		}
		raw = hash.Sum(nil)
	}
	err = w.RWSet.SetState(w.namespace, key, raw)
	if err != nil {
		return errors.Wrapf(err, "failed to write token request'%s'", w.TxID)
	}
	return nil
}

func (w *Translator) ReadTokenRequest() ([]byte, error) {
	key, err := keys.CreateTokenRequestKey(w.TxID)
	if err != nil {
		return nil, errors.Errorf("can't create for token request '%s'", w.TxID)
	}
	tr, err := w.RWSet.GetState(w.namespace, key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read token request'%s'", w.TxID)
	}
	return tr, nil
}

func (w *Translator) ReadSetupParameters() ([]byte, error) {
	setupKey, err := keys.CreateSetupKey()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create setup key")
	}
	raw, err := w.RWSet.GetState(w.namespace, setupKey)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setup parameters")
	}
	return raw, nil
}

func (w *Translator) QueryTokens(ids []*token.ID) ([][]byte, error) {
	var res [][]byte
	var errs []error
	for _, id := range ids {
		outputID, err := keys.CreateTokenKey(id.TxId, id.Index)
		if err != nil {
			errs = append(errs, errors.Errorf("error creating output ID: %s", err))
			continue
			// return nil, errors.Errorf("error creating output ID: %s", err)
		}
		logger.Debugf("query state [%s:%s]", id, outputID)
		bytes, err := w.RWSet.GetState(w.namespace, outputID)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "failed getting output for [%s]", outputID))
			// return nil, errors.Wrapf(err, "failed getting output for [%s]", outputID)
			continue
		}
		if len(bytes) == 0 {
			errs = append(errs, errors.Errorf("output for key [%s] does not exist", outputID))
			// return nil, errors.Errorf("output for key [%s] does not exist", outputID)
			continue
		}
		res = append(res, bytes)
	}
	if len(errs) != 0 {
		return nil, errors.Errorf("failed quering tokens [%v] with errs [%d][%v]", ids, len(errs), errs)
	}
	return res, nil
}

func (w *Translator) GetTransferMetadataSubKey(k string) (string, error) {
	return keys.GetTransferMetadataSubKey(k)
}

func (w *Translator) AreTokensSpent(id []string, graphHiding bool) ([]bool, error) {
	return w.areTokensSpent(id, graphHiding)
}

func (w *Translator) checkProcess(action interface{}) error {
	if err := w.checkAction(action); err != nil {
		return err
	}
	return nil
}

func (w *Translator) checkAction(tokenAction interface{}) error {
	switch action := tokenAction.(type) {
	case IssueAction:
		return w.checkIssue(action)
	case TransferAction:
		return w.checkTransfer(action)
	case SetupAction:
		return nil
	default:
		return errors.Errorf("unknown token action: %T", action)
	}
}

func (w *Translator) checkIssue(issue IssueAction) error {
	// check if the keys of issued tokens aren't already used.
	// check is assigned owners are valid
	for i := 0; i < issue.NumOutputs(); i++ {
		if err := w.checkTokenDoesNotExist(w.counter+uint64(i), w.TxID); err != nil {
			return err
		}
	}
	return nil
}

func (w *Translator) checkTransfer(t TransferAction) error {
	keys, err := t.GetInputs()
	if err != nil {
		return errors.Wrapf(err, "invalid transfer: failed getting input IDs")
	}
	if !t.IsGraphHiding() {
		for _, key := range keys {
			bytes, err := w.RWSet.GetState(w.namespace, key)
			if err != nil {
				return errors.Wrapf(err, "invalid transfer: failed getting state [%s]", key)
			}
			if len(bytes) == 0 {
				return errors.Errorf("invalid transfer: input is already spent [%s]", key)
			}
		}
	} else {
		for _, key := range keys {
			bytes, err := w.RWSet.GetState(w.namespace, key)
			if err != nil {
				return errors.Wrapf(err, "invalid transfer: failed getting state [%s]", key)
			}
			if len(bytes) != 0 {
				return errors.Errorf("invalid transfer: input is already spent [%s:%v]", key, bytes)
			}
		}
	}
	// check if the keys of the new tokens aren't already used.
	for i := 0; i < t.NumOutputs(); i++ {
		if !t.IsRedeemAt(i) {
			// this is not a redeemed output
			err := w.checkTokenDoesNotExist(w.counter+uint64(i), w.TxID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *Translator) checkTokenDoesNotExist(index uint64, txID string) error {
	tokenKey, err := keys.CreateTokenKey(txID, index)
	if err != nil {
		return errors.Wrapf(err, "error creating output ID")
	}

	outputBytes, err := w.RWSet.GetState(w.namespace, tokenKey)
	if err != nil {
		return err
	}
	if len(outputBytes) != 0 {
		return errors.Errorf("token already exists: %s", tokenKey)
	}
	return nil
}

func (w *Translator) commitProcess(action interface{}) error {
	logger.Debugf("committing action with txID '%s'", w.TxID)
	err := w.commitAction(action)
	if err != nil {
		logger.Errorf("error committing action with txID '%s': %s", w.TxID, err)
		return err
	}

	logger.Debugf("action with txID '%s' committed successfully", w.TxID)
	return nil
}

func (w *Translator) commitAction(tokenAction interface{}) (err error) {
	switch action := tokenAction.(type) {
	case IssueAction:
		err = w.commitIssueAction(action)
	case TransferAction:
		err = w.commitTransferAction(action)
	case SetupAction:
		err = w.commitSetupAction(action)
	}
	return
}

func (w *Translator) commitSetupAction(setup SetupAction) error {
	raw, err := setup.GetSetupParameters()
	if err != nil {
		return err
	}
	setupKey, err := keys.CreateSetupKey()
	if err != nil {
		return err
	}
	err = w.RWSet.SetState(w.namespace, setupKey, raw)
	if err != nil {
		return err
	}
	return nil
}

func (w *Translator) commitIssueAction(issueAction IssueAction) error {
	base := w.counter

	outputs, err := issueAction.GetSerializedOutputs()
	if err != nil {
		return err
	}
	for i, output := range outputs {
		outputID, err := keys.CreateTokenKey(w.TxID, base+uint64(i))
		if err != nil {
			return errors.Errorf("error creating output ID: %s", err)
		}
		if err := w.RWSet.SetState(w.namespace, outputID, output); err != nil {
			return err
		}
	}

	// store metadata
	metadata := issueAction.GetMetadata()
	for key, value := range metadata {
		k, err := keys.CreateIssueActionMetadataKey(key)
		if err != nil {
			return errors.Wrapf(err, "failed constructing metadata key")
		}
		raw, err := w.RWSet.GetState(w.namespace, k)
		if err != nil {
			return err
		}
		if len(raw) != 0 {
			return errors.Errorf("entry with issue metadata key [%s] is already occupied by [%s]", key, string(raw))
		}
		if err := w.RWSet.SetState(w.namespace, k, value); err != nil {
			return err
		}
	}

	w.counter = w.counter + uint64(len(outputs))
	return nil
}

// commitTransferAction is called for both transfer and redeem transactions
// Check the owner of each output to determine how to generate the key
func (w *Translator) commitTransferAction(transferAction TransferAction) error {
	base := w.counter

	// store outputs
	for i := 0; i < transferAction.NumOutputs(); i++ {
		if !transferAction.IsRedeemAt(i) {
			outputID, err := keys.CreateTokenKey(w.TxID, base+uint64(i))
			if err != nil {
				return errors.Errorf("error creating output ID: %s", err)
			}

			bytes, err := transferAction.SerializeOutputAt(i)
			if err != nil {
				return err
			}
			err = w.RWSet.SetState(w.namespace, outputID, bytes)
			if err != nil {
				return err
			}
		}
	}

	// store inputs
	ids, err := transferAction.GetInputs()
	if err != nil {
		return err
	}
	err = w.spendTokens(ids, transferAction.IsGraphHiding())
	if err != nil {
		return err
	}

	// store metadata
	metadata := transferAction.GetMetadata()
	for key, value := range metadata {
		k, err := keys.CreateTransferActionMetadataKey(key)
		if err != nil {
			return errors.Wrapf(err, "failed constructing metadata key")
		}
		raw, err := w.RWSet.GetState(w.namespace, k)
		if err != nil {
			return err
		}
		if len(raw) != 0 {
			return errors.Errorf("entry with transfer metadata key [%s] is already occupied by [%s]", key, base64.StdEncoding.EncodeToString(raw))
		}
		if err := w.RWSet.SetState(w.namespace, k, value); err != nil {
			return err
		}
	}

	w.counter = w.counter + uint64(transferAction.NumOutputs())
	return nil
}

func (w *Translator) spendTokens(ids []string, graphHiding bool) error {
	if !graphHiding {
		for _, id := range ids {
			logger.Debugf("Delete state %s\n", id)
			err := w.RWSet.DeleteState(w.namespace, id)
			if err != nil {
				return errors.Wrapf(err, "failed to delete output %s", id)
			}
			w.SpentIDs = append(w.SpentIDs, []byte(id))
		}
	} else {
		for _, id := range ids {
			logger.Debugf("add serial number %s\n", id)
			err := w.RWSet.SetState(w.namespace, id, []byte(strconv.FormatBool(true)))
			if err != nil {
				return errors.Wrapf(err, "failed to add serial number %s", id)
			}
			w.SpentIDs = append(w.SpentIDs, []byte(id))
		}
	}

	return nil
}

func (w *Translator) areTokensSpent(ids []string, graphHiding bool) ([]bool, error) {
	res := make([]bool, len(ids))
	if graphHiding {
		for i, id := range ids {
			logger.Debugf("check serial number %s\n", id)
			v, err := w.RWSet.GetState(w.namespace, id)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get serial number %s", id)
			}
			res[i] = len(v) != 0
		}
	} else {
		for i, id := range ids {
			logger.Debugf("check state %s\n", id)
			v, err := w.RWSet.GetState(w.namespace, id)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get output %s", id)
			}
			res[i] = len(v) == 0
		}
	}

	return res, nil
}
