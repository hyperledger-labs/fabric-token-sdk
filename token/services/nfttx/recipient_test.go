/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nfttx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestRecipientIdentity(t *testing.T) {
	// RequestRecipientIdentity panic/error check when nil context is passed
	assert.Panics(t, func() {
		_, _ = RequestRecipientIdentity(nil, nil)
	})
}

func TestRespondRequestRecipientIdentity(t *testing.T) {
	assert.Panics(t, func() {
		_, _ = RespondRequestRecipientIdentity(nil)
	})
}
