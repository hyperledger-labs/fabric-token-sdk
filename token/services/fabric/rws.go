/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/pkg/errors"
)

type RWSWrapper struct {
	r *fabric.RWSet
}

func NewRWSWrapper(r *fabric.RWSet) *RWSWrapper {
	return &RWSWrapper{r: r}
}

func (rwset *RWSWrapper) SetState(namespace string, key string, value []byte) error {
	return rwset.r.SetState(namespace, key, value)
}

func (rwset *RWSWrapper) GetState(namespace string, key string) ([]byte, error) {
	return rwset.r.GetState(namespace, key)
}

func (rwset *RWSWrapper) DeleteState(namespace string, key string) error {
	return rwset.r.DeleteState(namespace, key)
}

func (rwset *RWSWrapper) Bytes() ([]byte, error) {
	return rwset.r.Bytes()
}

func (rwset *RWSWrapper) Done() {
	rwset.r.Done()
}

func (rwset *RWSWrapper) SetStateMetadata(namespace, key string, metadata map[string][]byte) error {
	return rwset.r.SetStateMetadata(namespace, key, metadata)
}

func (rwset *RWSWrapper) Equals(r interface{}, namespace string) error {
	switch t := r.(type) {
	case *RWSWrapper:
		return rwset.r.Equals(t.r, namespace)
	case *fabric.RWSet:
		return rwset.r.Equals(t, namespace)
	default:
		return errors.Errorf("invalid type, got [%T]", t)
	}
}
