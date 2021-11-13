/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Vault interface {
	GetLastTxID() (string, error)
	ListUnspentTokens() (*token.UnspentTokens, error)
	Exists(id *token.ID) bool
	Store(certifications map[*token.ID][]byte) error
	TokenVault() *vault.Vault
}
