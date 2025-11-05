/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnchorInContext checks that the anchor is set in the context
func TestAnchorInContext(t *testing.T) {
	anchor := driver.TokenRequestAnchor("hello world")
	anotherAnchor := driver.TokenRequestAnchor("another anchor")
	v := NewValidator[driver.PublicParameters, driver.Input, driver.TransferAction, driver.IssueAction, driver.Deserializer](
		nil,
		nil,
		nil,
		nil,
		[]ValidateTransferFunc[driver.PublicParameters, driver.Input, driver.TransferAction, driver.IssueAction, driver.Deserializer]{
			func(c context.Context, ctx *Context[driver.PublicParameters, driver.Input, driver.TransferAction, driver.IssueAction, driver.Deserializer]) error {
				if anchor != ctx.Anchor {
					return fmt.Errorf("transfer, anchor does not match, expected %s, got %s", anchor, ctx.Anchor)
				}
				return nil
			},
		},
		[]ValidateIssueFunc[driver.PublicParameters, driver.Input, driver.TransferAction, driver.IssueAction, driver.Deserializer]{
			func(c context.Context, ctx *Context[driver.PublicParameters, driver.Input, driver.TransferAction, driver.IssueAction, driver.Deserializer]) error {
				if anchor != ctx.Anchor {
					return fmt.Errorf("issue, anchor does not match, expected %s, got %s", anchor, ctx.Anchor)
				}
				return nil
			},
		},
		[]ValidateAuditingFunc[driver.PublicParameters, driver.Input, driver.TransferAction, driver.IssueAction, driver.Deserializer]{
			func(c context.Context, ctx *Context[driver.PublicParameters, driver.Input, driver.TransferAction, driver.IssueAction, driver.Deserializer]) error {
				if anchor != ctx.Anchor {
					return fmt.Errorf("audit, anchor does not match, expected %s, got %s", anchor, ctx.Anchor)
				}
				return nil
			},
		},
	)

	// check anchor in the context for an issue action
	ctx := t.Context()
	err := v.VerifyIssue(ctx, anchor, nil, &mock.IssueAction{}, nil, nil, nil)
	require.NoError(t, err)
	err = v.VerifyIssue(ctx, anotherAnchor, nil, &mock.IssueAction{}, nil, nil, nil)
	require.Error(t, err)
	assert.EqualError(t, err, "issue, anchor does not match, expected hello world, got another anchor")

	// check anchor in the context for a transfer action
	err = v.VerifyTransfer(ctx, anchor, nil, &mock.TransferAction{}, nil, nil, nil)
	require.NoError(t, err)
	err = v.VerifyTransfer(ctx, anotherAnchor, nil, &mock.TransferAction{}, nil, nil, nil)
	require.Error(t, err)
	assert.EqualError(t, err, "transfer, anchor does not match, expected hello world, got another anchor")

	// check anchor in the context for a transfer action
	err = v.VerifyAuditing(ctx, anchor, nil, nil, nil, nil)
	require.NoError(t, err)
	err = v.VerifyAuditing(ctx, anotherAnchor, nil, nil, nil, nil)
	require.Error(t, err)
	assert.EqualError(t, err, "audit, anchor does not match, expected hello world, got another anchor")
}
