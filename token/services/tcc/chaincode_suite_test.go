/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package tcc_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestZKToken(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Token Chaincode Suite")
}
