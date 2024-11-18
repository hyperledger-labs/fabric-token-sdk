/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package math

import (
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/stretchr/testify/assert"
)

func TestCheckElement(t *testing.T) {
	var g1 *math.G1
	assert.Error(t, CheckElement(g1, 0))

	g1 = &math.G1{}
	assert.Error(t, CheckElement(g1, 0))
}
