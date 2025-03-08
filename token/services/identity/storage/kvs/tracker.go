/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

// Backend interface for key-value storage
type Backend interface {
	Put(id string, value interface{}) error
	Get(id string, entry interface{}) error
}

// KeyValuePair stores tracking info
type KeyValuePair struct {
	Key   string
	Value interface{}
	Error string
}

// TrackedKVS wraps a Backend and tracks operations
type TrackedKVS struct {
	Backend    Backend
	PutCounter int
	GetCounter int
	PutHistory []KeyValuePair
	GetHistory []KeyValuePair
}

func NewTrackedMemory() *TrackedKVS {
	backend, err := NewInMemory()
	if err != nil {
		panic(err)
	}
	return &TrackedKVS{
		Backend:    backend,
		PutHistory: []KeyValuePair{},
		GetHistory: []KeyValuePair{},
	}
}

func NewTrackedMemoryFrom(backend Backend) *TrackedKVS {
	return &TrackedKVS{
		Backend:    backend,
		PutHistory: []KeyValuePair{},
		GetHistory: []KeyValuePair{},
	}
}

func (f *TrackedKVS) Put(id string, entry interface{}) error {
	err := f.Backend.Put(id, entry)
	f.PutCounter++
	f.PutHistory = append(f.PutHistory, KeyValuePair{Key: id, Value: entry, Error: ""})
	return err
}

func (f *TrackedKVS) Get(id string, entry interface{}) error {
	f.GetCounter++

	errorMsg := ""
	var e any

	err := f.Backend.Get(id, entry)
	if err != nil {
		errorMsg = err.Error()
	} else {
		e = entry
	}

	f.GetHistory = append(f.GetHistory, KeyValuePair{Key: id, Value: e, Error: errorMsg})
	return err
}
