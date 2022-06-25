/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
)

type lm struct {
	lm *orion.IdentityManager
	ip IdentityProvider
}

func (n *lm) FSCNodeIdentity() view.Identity {
	return n.ip.DefaultIdentity()
}

func (n *lm) AnonymousIdentity() view.Identity {
	return view.Identity(n.lm.Me())
}

func (n *lm) IsMe(id view.Identity) bool {
	panic("implement me")
}

func (n *lm) GetAnonymousIdentity(label string, auditInfo []byte) (string, string, driver.GetFunc, error) {
	panic("implement me")
}

func (n *lm) GetAnonymousIdentifier(label string) (string, error) {
	panic("implement me")
}

func (n *lm) GetLongTermIdentity(label string) (string, string, view.Identity, error) {
	panic("implement me")
}

func (n *lm) GetLongTermIdentifier(id view.Identity) (string, error) {
	panic("implement me")
}

func (n *lm) RegisterIdentity(id string, typ string, path string) error {
	panic("implement me")
}
