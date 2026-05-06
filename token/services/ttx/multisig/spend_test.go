/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRequestSpendView_NilToken(t *testing.T) {
	v := NewRequestSpendView(nil)
	assert.NotNil(t, v)
	assert.Error(t, v.err)
}

func TestNewRequestSpendView_DefaultTimeout(t *testing.T) {
	v := &RequestSpendView{}
	v = v.WithTimeout(5 * time.Second)
	assert.Equal(t, 5*time.Second, v.timeout)
}

func TestRequestSpendView_WithTimeout(t *testing.T) {
	v := &RequestSpendView{timeout: defaultSpendRequestTimeout}
	assert.Equal(t, defaultSpendRequestTimeout, v.timeout)

	v.WithTimeout(10 * time.Second)
	assert.Equal(t, 10*time.Second, v.timeout)
}

func TestRequestSpendView_TimeoutApplied(t *testing.T) {
	answerCh := make(chan *answer)
	v := &RequestSpendView{timeout: 50 * time.Millisecond}

	timer := time.NewTimer(v.timeout)
	defer timer.Stop()

	var timedOut bool
	select {
	case <-answerCh:
		timedOut = false
	case <-timer.C:
		timedOut = true
	}

	assert.True(t, timedOut, "select should have taken the timer branch when no answer arrives")
}
