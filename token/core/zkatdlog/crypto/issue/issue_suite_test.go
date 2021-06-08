/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package issue_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestIssue(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Issue Suite")
}
