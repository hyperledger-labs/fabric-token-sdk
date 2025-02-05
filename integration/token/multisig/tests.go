/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
)

type OnRestartFunc = func(*integration.Infrastructure, string)

func TestAll(network *integration.Infrastructure, sel *token3.ReplicaSelector) {
	auditor := sel.Get("auditor")
	RegisterAuditor(network, auditor)
	issuer := sel.Get("issuer")
	alice := sel.Get("alice")
	bob := sel.Get("bob")
	charlie := sel.Get("charlie")

	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	IssueCash(network, "", "USD", 110, alice, auditor, true, issuer)
	CheckBalance(network, alice, "", "USD", 110, 0)
	CheckHolding(network, alice, "", "USD", 110, auditor)

	LockCash(network, alice, "", "USD", 50, bob, charlie, auditor)
	CheckBalance(network, alice, "", "USD", 60, 0)
	CheckBalance(network, bob, "", "USD", 0, 50)
	CheckBalance(network, charlie, "", "USD", 0, 50)
}
