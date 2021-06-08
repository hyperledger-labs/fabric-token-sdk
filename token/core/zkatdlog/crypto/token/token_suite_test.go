/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestToken(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Token Suite")
}
