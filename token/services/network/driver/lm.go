/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type GetFunc func() (view.Identity, []byte, error)

// LocalMembership models the local membership service
type LocalMembership interface {
	// DefaultIdentity returns the default FSC node identity
	DefaultIdentity() view.Identity

	// AnonymousIdentity returns a fresh anonymous identity
	AnonymousIdentity() view.Identity

	// IsMe returns true if the given identity belongs to the caller
	IsMe(id view.Identity) bool
}
