/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package audit_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAudit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Audit Suite")
}
