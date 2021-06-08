/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package impl_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestOffchainTx(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Offchain Tx Suite")
}
