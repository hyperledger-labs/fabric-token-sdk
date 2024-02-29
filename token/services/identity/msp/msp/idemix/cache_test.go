/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"testing"

	api2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/stretchr/testify/assert"
)

func TestIdentityCache(t *testing.T) {
	c := NewIdentityCache(func(opts *api2.IdentityOptions) (view.Identity, []byte, error) {
		return []byte("hello world"), []byte("audit"), nil
	}, 100, nil)
	id, audit, err := c.Identity(&api2.IdentityOptions{
		EIDExtension: true,
		AuditInfo:    nil,
	})
	assert.NoError(t, err)
	assert.Equal(t, view.Identity([]byte("hello world")), id)
	assert.Equal(t, []byte("audit"), audit)

	id, audit, err = c.Identity(nil)
	assert.NoError(t, err)
	assert.Equal(t, view.Identity([]byte("hello world")), id)
	assert.Equal(t, []byte("audit"), audit)
}
