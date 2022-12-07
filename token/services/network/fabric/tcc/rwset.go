/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tcc

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

type rwsWrapper struct {
	stub shim.ChaincodeStubInterface
}

func (rwset *rwsWrapper) SetState(namespace string, key string, value []byte) error {
	return rwset.stub.PutState(key, value)
}

func (rwset *rwsWrapper) GetState(namespace string, key string) ([]byte, error) {
	return rwset.stub.GetState(key)
}

func (rwset *rwsWrapper) DeleteState(namespace string, key string) error {
	return rwset.stub.DelState(key)
}
