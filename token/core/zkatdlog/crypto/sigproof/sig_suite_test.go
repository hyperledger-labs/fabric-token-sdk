/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package sigproof_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMembershipProof(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Membership MembershipProof Suite")
}
