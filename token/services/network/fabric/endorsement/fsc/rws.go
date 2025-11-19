/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fsc

import (
	fdriver "github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
)

type RWSWrapper struct {
	Stub *fdriver.RWSet
}

func (rwset *RWSWrapper) SetState(namespace string, key string, value []byte) error {
	return rwset.Stub.SetState(namespace, key, value)
}

func (rwset *RWSWrapper) GetState(namespace string, key string) ([]byte, error) {
	return rwset.Stub.GetState(namespace, key)
}

func (rwset *RWSWrapper) DeleteState(namespace string, key string) error {
	return rwset.Stub.DeleteState(namespace, key)
}
