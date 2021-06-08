/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package anonym_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAnonymousIssuer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Anonymous Issuer Suite")
}
