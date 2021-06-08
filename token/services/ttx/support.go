/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttx

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type allIssuersValid struct{}

func (i *allIssuersValid) Validate(creator view.Identity, tokenType string) error {
	return nil
}
