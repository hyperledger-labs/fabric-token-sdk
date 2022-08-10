/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

// AssertTokensInVault checks that the tokens are or are not in the vault
func AssertTokensInVault(vault *network.Vault, tx *ttx.Transaction, outputs *token.OutputStream, id view.Identity) {
	qe := vault.TokenVault().QueryEngine()
	for _, output := range outputs.Outputs() {
		tokenID := output.ID(tx.ID())
		if output.Owner.Equal(id) || tx.TokenService().WalletManager().IsMe(output.Owner) {
			// check it exists
			_, toks, err := qe.GetTokens(tokenID)
			assert.NoError(err, "failed to retrieve token [%s]", tokenID)
			assert.Equal(1, len(toks), "expected one token")
			assert.Equal(output.Quantity.Hex(), toks[0].Quantity, "token quantity mismatch")
			assert.Equal(output.Type, toks[0].Type, "token type mismatch")
		} else {
			// check it does not exist
			_, _, err := qe.GetTokens(tokenID)
			assert.Error(err, "token [%s] should not exist", tokenID)
			assert.True(strings.Contains(err.Error(), "token not found"))
		}
	}

}
