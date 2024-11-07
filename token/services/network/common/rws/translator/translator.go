/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package translator

import (
	"crypto/sha256"
	"strconv"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

var logger = logging.MustGetLogger("token-sdk.vault.translator")

// Translator validates token requests and generates the corresponding RWSets
type Translator struct {
	RWSet         ExRWSet
	KeyTranslator KeyTranslator
	TxID          string
	// SpentIDs the spent IDs added so far
	SpentIDs []string
	counter  uint64
}

func New(txID string, rws ExRWSet) *Translator {
	w := &Translator{
		RWSet:         rws,
		TxID:          txID,
		counter:       0,
		KeyTranslator: &keys.Translator{},
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

func (w *Translator) CommitTokenRequest(raw []byte, storeHash bool) ([]byte, error) {
	key, err := w.KeyTranslator.CreateTokenRequestKey(w.TxID)
	if err != nil {
		return nil, errors.Errorf("can't create for token request '%s'", w.TxID)
	}
	if err := w.RWSet.StateMustNotExist(key); err != nil {
		return nil, errors.Wrapf(err, "failed to read token request")
	}
	var h []byte
	if storeHash {
		hash := sha256.New()
		n, err := hash.Write(raw)
		if n != len(raw) {
			return nil, errors.Errorf("failed to write token request, hash failure '%s'", w.TxID)
		}
		if err != nil {
			return nil, errors.Wrapf(err, "failed to write token request, hash failure '%s'", w.TxID)
		}
		raw = hash.Sum(nil)
		h = raw
	}
	err = w.RWSet.SetState(key, raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to write token request'%s'", w.TxID)
	}
	return h, nil
}

func (w *Translator) ReadTokenRequest() ([]byte, error) {
	key, err := w.KeyTranslator.CreateTokenRequestKey(w.TxID)
	if err != nil {
		return nil, errors.Errorf("can't create for token request '%s'", w.TxID)
	}
	tr, err := w.RWSet.GetState(key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read token request'%s'", w.TxID)
	}
	return tr, nil
}

func (w *Translator) ReadSetupParameters() ([]byte, error) {
	setupKey, err := w.KeyTranslator.CreateSetupKey()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create setup key")
	}
	raw, err := w.RWSet.GetState(setupKey)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setup parameters")
	}
	return raw, nil
}

func (w *Translator) AddPublicParamsDependency() error {
	setupKey, err := w.KeyTranslator.CreateSetupHashKey()
	if err != nil {
		return errors.Wrapf(err, "failed to create setup key")
	}
	if err := w.RWSet.StateMustExist(setupKey, Any); err != nil {
		return errors.Wrapf(err, "failed to add public params dependency")
	}
	return nil
}

func (w *Translator) QueryTokens(ids []*token.ID) ([][]byte, error) {
	var res [][]byte
	var errs []error
	for _, id := range ids {
		outputID, err := w.KeyTranslator.CreateTokenKey(id.TxId, id.Index)
		if err != nil {
			errs = append(errs, errors.Errorf("error creating output ID: %s", err))
			continue
			// return nil, errors.Errorf("error creating output ID: %s", err)
		}
		logger.Debugf("query state [%s:%s]", id, outputID)
		bytes, err := w.RWSet.GetState(outputID)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "failed getting output for [%s]", outputID))
			continue
		}
		if len(bytes) == 0 {
			errs = append(errs, errors.Errorf("output for key [%s] does not exist", outputID))
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
	return w.KeyTranslator.GetTransferMetadataSubKey(k)
}

func (w *Translator) AreTokensSpent(ids []string, graphHiding bool) ([]bool, error) {
	res := make([]bool, len(ids))
	if graphHiding {
		for i, id := range ids {
			logger.Debugf("check serial number %s\n", id)
			k, err := w.KeyTranslator.CreateSNKey(id)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to generate key for id [%s]", id)
			}
			v, err := w.RWSet.GetState(k)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get serial number %s", id)
			}
			res[i] = len(v) != 0
		}
	} else {
		for i, id := range ids {
			logger.Debugf("check state %s\n", id)
			v, err := w.RWSet.GetState(id)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get output %s", id)
			}
			res[i] = len(v) == 0
		}
	}

	return res, nil
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
	inputs := t.GetInputs()

	if !t.IsGraphHiding() {
		// in this case, the state must exist
		for _, input := range inputs {
			key, err := w.KeyTranslator.CreateTokenKey(input.TxId, input.Index)
			if err != nil {
				return errors.Wrapf(err, "invalid transfer: failed creating output ID [%v]", input)
			}
			if err := w.RWSet.StateMustExist(key, VersionZero); err != nil {
				return errors.Wrapf(err, "invalid transfer: input must exist")
			}
		}
	} else {
		// in this case, the state must not exist
		for _, key := range t.GetSerialNumbers() {
			if err := w.RWSet.StateMustNotExist(key); err != nil {
				return errors.Wrapf(err, "invalid transfer: serial number must not exist")
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
	tokenKey, err := w.KeyTranslator.CreateTokenKey(txID, index)
	if err != nil {
		return errors.Wrapf(err, "error creating output ID")
	}
	if err := w.RWSet.StateMustNotExist(tokenKey); err != nil {
		return errors.Errorf("token already exists")
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
	setupKey, err := w.KeyTranslator.CreateSetupKey()
	if err != nil {
		return err
	}
	err = w.RWSet.SetState(setupKey, raw)
	if err != nil {
		return err
	}

	setupHashKey, err := w.KeyTranslator.CreateSetupHashKey()
	if err != nil {
		return err
	}
	hash := sha256.New()
	n, err := hash.Write(raw)
	if n != len(raw) {
		panic("hash failure")
	}
	if err != nil {
		panic(err)
	}
	digest := hash.Sum(nil)

	err = w.RWSet.SetState(setupHashKey, digest)
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
		outputID, err := w.KeyTranslator.CreateTokenKey(w.TxID, base+uint64(i))
		if err != nil {
			return errors.Errorf("error creating output ID: %s", err)
		}
		if err := w.RWSet.SetState(outputID, output); err != nil {
			return err
		}
	}

	// store metadata
	metadata := issueAction.GetMetadata()
	for key, value := range metadata {
		k, err := w.KeyTranslator.CreateIssueActionMetadataKey(key)
		if err != nil {
			return errors.Wrapf(err, "failed constructing metadata key")
		}
		if err := w.RWSet.StateMustNotExist(k); err != nil {
			return errors.Errorf("entry with issue metadata key [%s] is already occupied", key)
		}
		if err := w.RWSet.SetState(k, value); err != nil {
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
			outputID, err := w.KeyTranslator.CreateTokenKey(w.TxID, base+uint64(i))
			if err != nil {
				return errors.Errorf("error creating output ID: %s", err)
			}

			bytes, err := transferAction.SerializeOutputAt(i)
			if err != nil {
				return err
			}
			err = w.RWSet.SetState(outputID, bytes)
			if err != nil {
				return err
			}
		}
	}

	// store inputs
	err := w.spendTokens(transferAction)
	if err != nil {
		return err
	}

	// store metadata
	metadata := transferAction.GetMetadata()
	for key, value := range metadata {
		k, err := w.KeyTranslator.CreateTransferActionMetadataKey(key)
		if err != nil {
			return errors.Wrapf(err, "failed constructing metadata key")
		}
		if err := w.RWSet.StateMustNotExist(k); err != nil {
			return errors.Errorf("entry with transfer metadata key [%s] is already occupied", key)
		}
		if err := w.RWSet.SetState(k, value); err != nil {
			return err
		}
	}

	w.counter = w.counter + uint64(transferAction.NumOutputs())
	return nil
}

func (w *Translator) spendTokens(transferAction TransferAction) error {
	if !transferAction.IsGraphHiding() {
		ids := transferAction.GetInputs()
		rwsetKeys := make([]string, len(ids))
		var err error
		for i, input := range ids {
			rwsetKeys[i], err = w.KeyTranslator.CreateTokenKey(input.TxId, input.Index)
			if err != nil {
				return errors.Wrapf(err, "invalid transfer: failed creating output ID [%v]", input)
			}
		}

		for _, id := range rwsetKeys {
			logger.Debugf("Delete state %s\n", id)
			err := w.RWSet.DeleteState(id)
			if err != nil {
				return errors.Wrapf(err, "failed to delete output %s", id)
			}
			if err := w.appendSpentID(id); err != nil {
				return errors.Wrapf(err, "failed to append spent id [%s]", id)
			}
		}
	} else {
		ids := transferAction.GetSerialNumbers()
		for _, id := range ids {
			logger.Debugf("add serial number %s\n", id)
			k, err := w.KeyTranslator.CreateSNKey(id)
			if err != nil {
				return errors.Wrapf(err, "failed to generate key for id [%s]", id)
			}
			if err := w.RWSet.SetState(k, []byte(strconv.FormatBool(true))); err != nil {
				return errors.Wrapf(err, "failed to add serial number %s", id)
			}
			if err := w.appendSpentID(id); err != nil {
				return errors.Wrapf(err, "failed to append spent id [%s]", id)
			}
		}
	}

	return nil
}

func (w *Translator) appendSpentID(id string) error {
	// check first it is already in the list
	for _, d := range w.SpentIDs {
		if d == id {
			return errors.Errorf("[%s] already spent", id)
		}
	}
	w.SpentIDs = append(w.SpentIDs, id)
	return nil
}
