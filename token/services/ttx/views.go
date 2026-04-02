/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

// Context is an alias of view.Context
//
//go:generate counterfeiter -o deps/mock/context.go -fake-name Context . Context
type Context = view.Context

// Session is an alias of view.Session
//
//go:generate counterfeiter -o deps/mock/session.go -fake-name Session . Session
type Session = view.Session
