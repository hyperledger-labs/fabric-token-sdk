/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

import (
	"encoding/json"

	vault "github.com/hashicorp/vault/api"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/services/logging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/pkg/errors"
)

var (
	logger         = logging.MustGetLogger("view-sdk.kvs")
	token_sdk_path = "secret/data/"
)

// type Iterator interface {
// 	HasNext() bool
// 	Close() error
// 	Next(state interface{}) (string, error)
// }

type KVS struct {
	client *vault.Client
}

// NewWithClient returns a new KVS instance for the passed hashicorp vault API client
func NewWithClient(client *vault.Client) (*KVS, error) {
	return &KVS{
		client: client,
	}, nil
}

func (v *KVS) GetExisting(ids ...string) []string {
	results := make([]string, 0)
	results = append(results, "Not implemented")
	logger.Errorf("error GetExisting not implemented")
	return results
}

func (v *KVS) Exists(id string) bool {
	logger.Errorf("error Exists not implemented")
	return false
}

func (v *KVS) Put(id string, state interface{}) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal state with id [%s]", id)
	}

	id = token_sdk_path + id

	value := map[string]interface{}{"value": string(raw)}
	_, err = v.client.Logical().Write(id, map[string]interface{}{"data": value})
	if err == nil {
		logger.Debugf("put state of id %s successfully", id)
	}
	return err
}

func (v *KVS) Get(id string, state interface{}) error {
	id = token_sdk_path + id
	secret, err := v.client.Logical().Read(id)
	if err != nil {
		logger.Debugf("failed retrieving state of id %s", id)
		return errors.Wrapf(err, "failed retrieving state of id %s", id)
	}

	data := secret.Data
	if len(data) == 0 {
		return errors.Errorf("state of id %s does not exist", id)
	}

	value, ok := data["value"].(string)
	if !ok {
		return errors.Wrapf(err, "invalid value type: expected string")
	}

	if err := json.Unmarshal([]byte(value), state); err != nil {
		logger.Debugf("failed retrieving state of id %s, cannot unmarshal state, error [%s]", id, err)
		return errors.Wrapf(err, "failed retrieving state of id %s], cannot unmarshal state", id)
	}

	logger.Debugf("got state of id %s successfully", id)
	return nil
}

func (v *KVS) GetByPartialCompositeID(prefix string, attrs []string) (kvs.Iterator, error) {
	logger.Errorf("error GetByPartialCompositeID not implemented")
	return &vaultIterator{ri: nil}, nil
}

type vaultIterator struct {
	ri collections.Iterator[*driver.UnversionedRead]
}

func (i *vaultIterator) HasNext() bool {
	logger.Errorf("error HasNext not implemented")
	return false
}

func (i *vaultIterator) Close() error {
	logger.Errorf("error Close not implemented")
	return nil
}

func (i *vaultIterator) Next(state interface{}) (string, error) {
	logger.Errorf("error Next not implemented")
	return "Not implemented ", nil
}
