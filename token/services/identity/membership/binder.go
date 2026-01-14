/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package membership

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

// NoBinder implements a no-op binder
type NoBinder struct{}

func (n *NoBinder) Bind(ctx context.Context, longTerm token.Identity, ephemeral ...token.Identity) error {
	return nil
}
