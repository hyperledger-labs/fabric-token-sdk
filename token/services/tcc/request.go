/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tcc

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/pkg/errors"
)

type ExtractTokenRequestFunc func(stub shim.ChaincodeStubInterface) ([]byte, error)

func ExtractTokenRequestFromTransient(stub shim.ChaincodeStubInterface) ([]byte, error) {
	args := stub.GetArgs()
	if len(args) != 1 {
		return nil, errors.New("empty token request")
	}
	// extract token request from transient
	t, err := stub.GetTransient()
	if err != nil {
		return nil, errors.New("failed getting transient")
	}
	tokenRequest, ok := t["token_request"]
	if !ok {
		return nil, errors.New("failed getting token request, entry not found")
	}

	return tokenRequest, nil
}
