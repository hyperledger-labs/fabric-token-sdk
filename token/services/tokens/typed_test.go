/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/driver"
	"github.com/stretchr/testify/assert"
	"github.com/test-go/testify/require"
)

func TestSerialization(t *testing.T) {
	raw := []byte("pineapple")
	wrappedToken, err := WrapWithType(0, raw)
	require.NoError(t, err)
	tok, err := UnmarshalTypedToken(wrappedToken)
	require.NoError(t, err)
	assert.Equal(t, driver.Type(0), tok.Type)
	assert.Equal(t, driver.Token(raw), tok.Token)
}
