/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/meta"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// IssueApplicationDataValidate accepts any metadata in the "pub" namespace.
// This gives the user of the Token SDK the option to attach public data to the token transaction.
func IssueApplicationDataValidate[P driver.PublicParameters, T driver.Input, TA driver.TransferAction, IA driver.IssueAction, DS driver.Deserializer](c context.Context, ctx *Context[P, T, TA, IA, DS]) error {
	for key := range ctx.IssueAction.GetMetadata() {
		if strings.HasPrefix(key, meta.PublicMetadataPrefix) {
			ctx.CountMetadataKey(key)
		}
	}
	return nil
}
