/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package elgamal_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEnc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Elgamal Encryption Suite")
}
