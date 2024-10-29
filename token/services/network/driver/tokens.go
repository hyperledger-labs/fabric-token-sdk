/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenQueryExecutorProvider interface {
	GetExecutor(network, channel string) (TokenQueryExecutor, error)
}

// TokenQueryExecutor queries the global state/ledger for tokens
type TokenQueryExecutor interface {
	QueryTokens(context view.Context, namespace string, IDs []*token2.ID) ([][]byte, error)
}

type SpentTokenQueryExecutorProvider interface {
	GetSpentExecutor(network, channel string) (SpentTokenQueryExecutor, error)
}

// SpentTokenQueryExecutor queries the global state/ledger for tokens
type SpentTokenQueryExecutor interface {
	QuerySpentTokens(context view.Context, namespace string, IDs []*token2.ID, meta []string) ([]bool, error)
}
