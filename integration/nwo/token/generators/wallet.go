/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package generators

import "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"

type Identity struct {
	ID      string
	Type    string
	Path    string
	Default bool
	Opts    *topology.BCCSP
}

type Wallets struct {
	Certifiers []Identity
	Issuers    []Identity
	Owners     []Identity
	Auditors   []Identity
}
