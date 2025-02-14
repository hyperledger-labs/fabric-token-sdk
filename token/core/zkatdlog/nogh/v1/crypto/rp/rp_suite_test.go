/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package rp_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEnc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bullet Proof Suite")
}
