/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	Type driver.Type = 1
)

// Token carries the output of an action
type Token struct {
	Output *token.Token
}
