/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package tcc

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

//go:generate counterfeiter -o mock/chaincode_stub_interface.go -fake-name ChaincodeStubInterface . ChaincodeStubInterface

// ChaincodeStubInterface is used by deployable chaincode apps to access and
// modify their ledgers
type ChaincodeStubInterface interface {
	shim.ChaincodeStubInterface
}
