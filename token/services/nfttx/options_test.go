/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nfttx

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/stretchr/testify/assert"
)

func TestWithUniqueID(t *testing.T) {
	opt := WithUniqueID("my-id")

	opts := &token.IssueOptions{
		Attributes: make(map[interface{}]interface{}),
	}
	err := opt(opts)
	assert.NoError(t, err)
	assert.Equal(t, "my-id", opts.Attributes["github.com/hyperledger-labs/fabric-token-sdk/token/services/nfttx/UniqueID"])
}
