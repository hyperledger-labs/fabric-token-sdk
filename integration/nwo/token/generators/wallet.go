/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package generators

type Identity struct {
	ID      string
	Type    string
	Path    string
	Default bool
}

type Wallets struct {
	Certifiers []Identity
	Issuers    []Identity
	Owners     []Identity
	Auditors   []Identity
}
