/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

func DatasourceName(tmsID token.TMSID) string {
	return fmt.Sprintf("%s-%s-%s", tmsID.Network, tmsID.Channel, tmsID.Namespace)
}
