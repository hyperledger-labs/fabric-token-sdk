/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1_test

import (
	"context"
	"testing"

	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/stretchr/testify/assert"
)

// TestAuditorService verifies the functionality of the AuditorService.
func TestAuditorService(t *testing.T) {
	service := v1.NewAuditorService()
	assert.NotNil(t, service)

	t.Run("AuditorCheck", func(t *testing.T) {
		// Test that AuditorCheck returns nil as expected for fabtoken
		err := service.AuditorCheck(context.Background(), &driver.TokenRequest{}, &driver.TokenRequestMetadata{}, "")
		assert.NoError(t, err)

		err = service.AuditorCheck(context.Background(), nil, nil, "")
		assert.NoError(t, err)
	})
}
