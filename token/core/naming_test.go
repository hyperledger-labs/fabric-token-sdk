/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"fmt"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/stretchr/testify/assert"
)

func TestTokenDriverName(t *testing.T) {
	id := TokenDriverName("test", 0)
	assert.Equal(t, fmt.Sprintf("%s.v%d", "test", 0), string(id))
}

func TestTokenDriverNameFromPP(t *testing.T) {
	pp := &mock.PublicParameters{}
	pp.IdentifierReturns("test")
	pp.VersionReturns(1)
	id := TokenDriverNameFromPP(pp)
	assert.Equal(t, fmt.Sprintf("%s.v%d", "test", 1), string(id))
}
