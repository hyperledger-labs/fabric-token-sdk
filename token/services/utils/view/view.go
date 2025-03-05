/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package view

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

type viewContext = view.Context

type contextWrapper struct {
	viewContext
	ctx context.Context
}

func (c *contextWrapper) Context() context.Context {
	return c.ctx
}

func RunViewWithTimeout(ctx view.Context, v view.View, timeout time.Duration, opts ...view.RunViewOption) (interface{}, error) {
	if timeout == 0 {
		return ctx.RunView(v, opts...)
	}

	timeoutContext, cancel := context.WithTimeout(ctx.Context(), timeout)
	defer cancel()
	newContext := &contextWrapper{
		viewContext: ctx,
		ctx:         timeoutContext,
	}
	return newContext.RunView(v, opts...)
}

// RunView runs passed view within the passed Context and using the passed options in a separate goroutine
func RunView(logger logging.Logger, context view.Context, view view.View, opts ...view.RunViewOption) {
	defer func() {
		if r := recover(); r != nil {
			logger.Debugf("panic in RunView: %v", r)
		}
	}()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Debugf("panic in RunView: %v", r)
			}
		}()
		_, err := context.RunView(view, opts...)
		if err != nil {
			logger.Errorf("failed to run view: %s", err)
		}
	}()
}
