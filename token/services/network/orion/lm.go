/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type lm struct {
	lm *orion.IdentityManager
	ip IdentityProvider
}

func (n *lm) DefaultIdentity() view.Identity {
	return n.ip.DefaultIdentity()
}

func (n *lm) AnonymousIdentity() view.Identity {
	return view.Identity(n.lm.Me())
}
