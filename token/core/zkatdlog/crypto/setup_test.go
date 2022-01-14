/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package crypto

import (
	"fmt"
	"testing"
	"time"

	math3 "github.com/IBM/mathlib"
	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	s := time.Now()
	_, err := Setup(100, 2, nil, math3.FP256BN_AMCL)
	e := time.Now()
	fmt.Printf("elapsed %d", e.Sub(s).Milliseconds())
	assert.NoError(t, err)
}
