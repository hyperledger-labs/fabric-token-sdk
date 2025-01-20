/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package topology

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509/msp"
)

type Identity struct {
	ID      string
	Type    string
	Path    string
	Default bool
	Opts    *msp.BCCSP
	Raw     []byte
}

type Wallets struct {
	Certifiers []Identity
	Issuers    []Identity
	Owners     []Identity
	Auditors   []Identity
}
