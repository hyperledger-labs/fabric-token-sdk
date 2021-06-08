/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package pssign_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPSSignature(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PS signature Suite")
}
