/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/meta"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/stretchr/testify/assert"
)

func TestTransferApplicationDataValidate(t *testing.T) {
	tests := []struct {
		name    string
		err     bool
		errMsg  string
		context func() (*TestContext, TestCheck)
	}{
		{
			name: "no metadata",
			err:  false,
			context: func() (*TestContext, TestCheck) {
				pp := &mock.PublicParameters{}
				pp.AuditorsReturns(nil)
				action := &mock.TransferAction{}
				action.GetMetadataReturns(nil)
				ctx := &TestContext{
					PP:             pp,
					TransferAction: action,
				}
				return ctx, func() bool {
					return len(ctx.MetadataCounter) == 0
				}
			},
		},
		{
			name: "one metadata without the selected prefix",
			err:  false,
			context: func() (*TestContext, TestCheck) {
				pp := &mock.PublicParameters{}
				pp.AuditorsReturns(nil)
				action := &mock.TransferAction{}
				action.GetMetadataReturns(map[string][]byte{
					"key1": []byte("value1"),
				})
				ctx := &TestContext{
					PP:             pp,
					TransferAction: action,
				}
				return ctx, func() bool {
					return len(ctx.MetadataCounter) == 0
				}
			},
		},
		{
			name: "one metadata with the selected prefix",
			err:  false,
			context: func() (*TestContext, TestCheck) {
				pp := &mock.PublicParameters{}
				pp.AuditorsReturns(nil)
				action := &mock.TransferAction{}
				k := meta.PublicMetadataPrefix + "key1"
				action.GetMetadataReturns(map[string][]byte{
					k:             []byte("value1"),
					"another key": []byte("value2"),
				})
				ctx := &TestContext{
					PP:              pp,
					TransferAction:  action,
					MetadataCounter: map[MetadataCounterID]int{},
				}
				return ctx, func() bool {
					return len(ctx.MetadataCounter) == 1 && ctx.MetadataCounter[k] == 1
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, check := tt.context()
			err := TransferApplicationDataValidate(context.Background(), ctx)
			if tt.err {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
			if check != nil {
				assert.True(t, check())
			}
		})
	}
}
