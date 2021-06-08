/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package tcc

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

type rwsWrapper struct {
	stub shim.ChaincodeStubInterface
}

func (rwset *rwsWrapper) SetState(namespace string, key string, value []byte) error {
	return rwset.stub.PutState(key, value)
}
func (rwset *rwsWrapper) GetState(namespace string, key string, opts ...fabric.GetStateOpt) ([]byte, error) {
	return rwset.stub.GetState(key)
}
func (rwset *rwsWrapper) DeleteState(namespace string, key string) error {
	return rwset.stub.DelState(key)
}
func (rwset *rwsWrapper) Bytes() ([]byte, error) {
	return nil, nil
}
func (rwset *rwsWrapper) Done() {
	return
}
func (rwset *rwsWrapper) GetStateMetadata(namespace, key string, opts ...fabric.GetStateOpt) (map[string][]byte, error) {
	return nil, nil
}
func (rwset *rwsWrapper) SetStateMetadata(namespace, key string, metadata map[string][]byte) error {
	return nil
}
func (rwset *rwsWrapper) AppendRWSet(raw []byte, nss ...string) error {
	return nil
}
func (rwset *rwsWrapper) GetReadAt(ns string, i int) (string, []byte, error) {
	return "", nil, nil
}
func (rwset *rwsWrapper) GetWriteAt(ns string, i int) (string, []byte, error) {
	return "", nil, nil
}
func (rwset *rwsWrapper) NumReads(ns string) int {
	return 0
}
func (rwset *rwsWrapper) NumWrites(ns string) int {
	return 0
}
func (rwset *rwsWrapper) Namespaces() []string {
	return nil
}
