/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package o2omp

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test3OMP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "3OMP Suite")
}
