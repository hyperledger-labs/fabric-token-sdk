/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/common"
	"github.com/stretchr/testify/assert"
)

func TestIdentityCache(t *testing.T) {
	c := NewIdentityCache(func(opts *common.IdentityOptions) (driver.Identity, []byte, error) {
		return []byte("hello world"), []byte("audit"), nil
	}, 100, nil)
	id, audit, err := c.Identity(&common.IdentityOptions{
		EIDExtension: true,
		AuditInfo:    nil,
	})
	assert.NoError(t, err)
	assert.Equal(t, driver.Identity([]byte("hello world")), id)
	assert.Equal(t, []byte("audit"), audit)

	id, audit, err = c.Identity(nil)
	assert.NoError(t, err)
	assert.Equal(t, driver.Identity([]byte("hello world")), id)
	assert.Equal(t, []byte("audit"), audit)
}
