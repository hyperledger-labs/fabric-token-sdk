/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type GetFunc func() (view.Identity, []byte, error)

type LocalMembership interface {
	DefaultIdentity() view.Identity
	AnonymousIdentity() view.Identity
	IsMe(id view.Identity) bool
	GetAnonymousIdentifier(label string) (string, error)
	GetAnonymousIdentity(label string, auditInfo []byte) (string, string, GetFunc, error)
	GetLongTermIdentifier(id view.Identity) (string, error)
	GetLongTermIdentity(label string) (string, string, view.Identity, error)
}
