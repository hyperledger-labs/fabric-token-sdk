/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"context"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalBidirectionalChannel(t *testing.T) {
	ctx := context.Background()
	caller := "alice"
	contextID := "cid-1"
	endpoint := "endpoint-1"
	pkid := []byte("pkid-1")

	channel, err := NewLocalBidirectionalChannel(ctx, caller, contextID, endpoint, pkid)
	require.NoError(t, err)
	require.NotNil(t, channel)

	left := channel.LeftSession()
	require.NotNil(t, left)
	right := channel.RightSession()
	require.NotNil(t, right)

	// Test Info
	info := left.Info()
	assert.NotEmpty(t, info.ID)
	assert.Equal(t, endpoint, info.Endpoint)
	assert.Equal(t, pkid, info.EndpointPKID)
	assert.False(t, info.Closed)

	// Test Send and Receive (Left to Right)
	payload := []byte("hello from left")
	err = left.Send(payload)
	assert.NoError(t, err)

	msg := <-right.Receive()
	assert.NotNil(t, msg)
	assert.Equal(t, payload, msg.Payload)
	assert.Equal(t, int32(view.OK), msg.Status)
	assert.Equal(t, info.ID, msg.SessionID)
	assert.Equal(t, contextID, msg.ContextID)
	assert.Equal(t, caller, msg.Caller)

	// Test Send and Receive (Right to Left)
	payloadRL := []byte("hello from right")
	err = right.Send(payloadRL)
	assert.NoError(t, err)

	msgRL := <-left.Receive()
	assert.NotNil(t, msgRL)
	assert.Equal(t, payloadRL, msgRL.Payload)
	assert.Equal(t, int32(view.OK), msgRL.Status)

	// Test SendError
	errPayload := []byte("error from left")
	err = left.SendError(errPayload)
	assert.NoError(t, err)

	errMsg := <-right.Receive()
	assert.NotNil(t, errMsg)
	assert.Equal(t, errPayload, errMsg.Payload)
	assert.Equal(t, int32(view.ERROR), errMsg.Status)

	// Test SendWithContext
	ctx2 := context.WithValue(ctx, "key", "value")
	payloadCtx := []byte("hello with context")
	err = left.SendWithContext(ctx2, payloadCtx)
	assert.NoError(t, err)

	msgCtx := <-right.Receive()
	assert.NotNil(t, msgCtx)
	assert.Equal(t, payloadCtx, msgCtx.Payload)
	assert.Equal(t, ctx2, msgCtx.Ctx)

	// Test SendErrorWithContext
	errPayloadCtx := []byte("error with context")
	err = left.SendErrorWithContext(ctx2, errPayloadCtx)
	assert.NoError(t, err)

	errMsgCtx := <-right.Receive()
	assert.NotNil(t, errMsgCtx)
	assert.Equal(t, errPayloadCtx, errMsgCtx.Payload)
	assert.Equal(t, int32(view.ERROR), errMsgCtx.Status)
	assert.Equal(t, ctx2, errMsgCtx.Ctx)

	// Test multiple messages to check buffer
	for i := 0; i < 5; i++ {
		err = left.Send([]byte{byte(i)})
		assert.NoError(t, err)
	}
	for i := 0; i < 5; i++ {
		msg := <-right.Receive()
		assert.Equal(t, []byte{byte(i)}, msg.Payload)
	}

	// Test Close
	left.Close()
	assert.True(t, left.Info().Closed)
	assert.Nil(t, left.Receive())

	err = left.Send([]byte("after close"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session is closed")

	err = left.SendError([]byte("after close"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session is closed")

	// Right should still be open
	assert.False(t, right.Info().Closed)
	err = right.Send([]byte("to closed left"))
	assert.NoError(t, err)

	right.Close()
	assert.True(t, right.Info().Closed)
	assert.Nil(t, right.Receive())
}
