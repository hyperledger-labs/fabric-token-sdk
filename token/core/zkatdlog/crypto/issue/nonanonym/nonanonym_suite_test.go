/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package nonanonym_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNonAnonymousIssuer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cleartext Issuer Suite")
}
