/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	vault "github.com/hashicorp/vault/api"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/services/logging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/pkg/errors"
)

const DATA = "data"
const VALUE = "value"

var (
	logger = logging.MustGetLogger("view-sdk.kvs")
)

type KVS struct {
	client *vault.Client
	path   string
}

// NewWithClient returns a new KVS instance for the passed hashicorp vault API client
func NewWithClient(client *vault.Client, path string) (*KVS, error) {
	return &KVS{
		client: client,
		path:   path,
	}, nil
}

func (v *KVS) NormalizeID(id string, isShort bool) string {
	// Replace all occurrences of \x00 with /
	replaced := strings.ReplaceAll(id, "\x00", "/")
	// Remove the leading slash if it exists
	replaced = strings.TrimPrefix(replaced, "/")
	// Remove the trailing slash if it exists
	id = strings.TrimSuffix(replaced, "/")
	// Append the id to the path
	if isShort {
		return id
	}
	return v.path + id
}

func (v *KVS) GetExisting(ids ...string) []string {
	results := make([]string, 0)

	for _, id := range ids {
		if v.Exists(id) {
			results = append(results, id)
		}
	}

	return results
}

func (v *KVS) Exists(id string) bool {
	id = v.NormalizeID(id, false)
	secret, err := v.client.Logical().Read(id)
	if err != nil {
		logger.Debugf("failed to check existence of id [%s]: %v", id, err)
		return false
	}

	if secret == nil || secret.Data == nil {
		logger.Debugf("state of id [%s] does not exist", id)
		return false
	}

	data, ok := secret.Data[DATA].(map[string]interface{})
	if !ok || len(data) == 0 {
		logger.Debugf("state of id [%s] does not exist", id)
		return false
	}

	return true
}

func (v *KVS) Delete(id string) error {
	id = v.NormalizeID(id, false)

	// Delete the secret from Vault
	_, err := v.client.Logical().Delete(id)
	if err != nil {
		logger.Errorf("failed to delete state of id [%s]: %v", id, err)
		return errors.Wrapf(err, "failed to delete state of id [%s]", id)
	}

	logger.Debugf("deleted state of id [%s] successfully", id)
	return nil
}

func (v *KVS) Put(id string, state interface{}) error {
	id = v.NormalizeID(id, false)

	raw, err := json.Marshal(state)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal state with id [%s]", id)
	}

	value := map[string]interface{}{VALUE: base64.StdEncoding.EncodeToString(raw)}
	_, err = v.client.Logical().Write(id, map[string]interface{}{DATA: value})
	if err == nil {
		logger.Debugf("put state of id [%s] successfully", id)
		return nil
	}

	return errors.Wrapf(err, "cannot Put state with id [%s]", id)
}

func (v *KVS) Get(id string, state interface{}) error {
	id = v.NormalizeID(id, false)
	secret, err := v.client.Logical().Read(id)
	if err != nil || secret == nil || secret.Data == nil {
		logger.Errorf("failed retrieving state of id [%s]", id)
		return errors.Errorf("failed retrieving state of id [%s]", id)
	}

	data, _ := secret.Data[DATA].(map[string]interface{})
	if len(data) == 0 {
		return errors.Errorf("state of id [%s] does not exist", id)
	}

	value, ok := data[VALUE]
	if !ok {
		return errors.Errorf("missing 'value' key in data")
	}
	raw, err := base64.StdEncoding.DecodeString(value.(string))
	if err != nil {
		logger.Debugf("Failed to decode base64 string: %v, error: %v", value, err)
		return errors.Wrapf(err, "failed to decode base64 string: %v", value)
	}

	if err := json.Unmarshal(raw, state); err != nil {
		logger.Debugf("failed retrieving state of id [%s], cannot unmarshal state, error [%s]", id, err)
		return errors.Wrapf(err, "failed retrieving state of id [%s], cannot unmarshal state", id)
	}
	logger.Debugf("got state of id [%s] successfully", id)
	return nil
}

func (v *KVS) GetByPartialCompositeID(prefix string, attrs []string) (kvs.Iterator, error) {

	partialCompositeKey, err := kvs.CreateCompositeKey(prefix, attrs)
	shortNormalizePartialCompositeKey := v.NormalizeID(partialCompositeKey, true)
	partialCompositeKey = v.NormalizeID(partialCompositeKey, false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed building composite key")
	}
	secret, err := v.client.Logical().List(partialCompositeKey)
	if err != nil {
		errors.Wrapf(err, "failed")
	}

	// Check if the secret contains any keys
	if secret == nil || secret.Data == nil {
		errors.Errorf("secret contains no keys")
	}

	// Extract the keys from the response
	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		errors.Errorf("unable to extract the keys from the response")
	}
	// Convert keys to []*string
	stringKeys := make([]*string, len(keys))
	for i, key := range keys {
		keyStr := shortNormalizePartialCompositeKey + "/" + key.(string) // Cast to string
		stringKeys[i] = &keyStr                                          // Store pointer to the string
	}
	// Create and return a sliceIterator for the keys
	keys_iterator := collections.NewSliceIterator(stringKeys)
	return &vaultIterator{ri: keys_iterator, client: v}, nil
}

type vaultIterator struct {
	ri     collections.Iterator[*string]
	next   *string
	client *KVS
}

func (i *vaultIterator) HasNext() bool {
	var err error
	i.next, err = i.ri.Next()
	if err != nil || i.next == nil {
		return false
	}
	return true
}

func (i *vaultIterator) Close() error {
	i.ri.Close()
	return nil
}

func (i *vaultIterator) Next(state interface{}) (string, error) {
	if i.next == nil {
		return "", errors.Errorf("nil i.next in the vault terator")
	}
	err := i.client.Get(*i.next, state)
	return *i.next, err
}
