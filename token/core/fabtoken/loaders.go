/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenVault interface {
	PublicParams() ([]byte, error)
}

type VaultTokenLoader struct {
	TokenVault driver.QueryEngine
}

// GetTokens takes an array of token identifiers (txID, index) and returns the keys of the identified tokens
// in the vault and the content of the tokens
func (s *VaultTokenLoader) GetTokens(ids []*token.ID) ([]string, []*token.Token, error) {
	return s.TokenVault.GetTokens(ids...)
}
