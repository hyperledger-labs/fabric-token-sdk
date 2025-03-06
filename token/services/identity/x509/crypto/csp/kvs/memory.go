/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type MemoryKVS struct {
	kvs map[string][]byte
}

func NewMemoryKVS() *MemoryKVS {
	return &MemoryKVS{
		kvs: make(map[string][]byte),
	}
}

func (f *MemoryKVS) Put(id string, entry interface{}) error {
	bytes, err := json.Marshal(entry)
	if err != nil {
		return errors.Wrapf(err, "marshalling key [%s] failed", id)
	}

	f.kvs[id] = bytes
	return nil
}

func (f *MemoryKVS) Get(id string, entry interface{}) error {
	bytes, ok := f.kvs[id]
	if !ok {
		return errors.Errorf("key [%s] not found", id)
	}
	err := json.Unmarshal(bytes, entry)
	if err != nil {
		return errors.Wrapf(err, "could not unmarshal bytes")
	}
	return nil
}
