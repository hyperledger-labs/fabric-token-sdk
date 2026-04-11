/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nfttx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCollectEndorsementsView(t *testing.T) {
	assert.Panics(t, func() {
		NewCollectEndorsementsView(nil)
	})
}

func TestNewOrderingAndFinalityView(t *testing.T) {
	assert.Panics(t, func() {
		NewOrderingAndFinalityView(nil)
	})
}

func TestNewOrderingAndFinalityWithTimeoutView(t *testing.T) {
	assert.Panics(t, func() {
		NewOrderingAndFinalityWithTimeoutView(nil, time.Second)
	})
}

func TestNewFinalityView(t *testing.T) {
	assert.Panics(t, func() {
		NewFinalityView(nil)
	})
}

func TestNewAcceptView(t *testing.T) {
	assert.Panics(t, func() {
		NewAcceptView(nil)
	})
}
