/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
)

var logger = flogging.MustGetLogger("token-sdk.tokens")

type NetworkProvider interface {
	GetNetwork(network string, channel string) (*network.Network, error)
}

// Transaction models a token transaction
type Transaction interface {
	ID() string
	Network() string
	Channel() string
	Request() *token.Request
}

// Tokens is the interface for the owner service
type Tokens struct {
	networkProvider NetworkProvider
	db              *tokendb.DB
}

func (a *Tokens) Append(tx Transaction) error {
	panic("not implemented yet")
}
