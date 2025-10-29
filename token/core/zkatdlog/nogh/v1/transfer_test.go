/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/stretchr/testify/assert"
)

func TestTransferService_VerifyTransfer(t *testing.T) {
	tests := []struct {
		name     string
		TestCase func() (*TransferService, driver.TransferAction, []*driver.TransferOutputMetadata)
		wantErr  string
	}{
		{
			name: "nil action",
			TestCase: func() (*TransferService, driver.TransferAction, []*driver.TransferOutputMetadata) {
				service := &TransferService{}
				return service, nil, nil
			},
			wantErr: "nil action",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, action, meta := tt.TestCase()
			err := service.VerifyTransfer(t.Context(), action, meta)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}
