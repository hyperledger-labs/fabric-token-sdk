/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package logging

import (
	"fmt"
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/hash"
)

func WalletID(id string) fmt.Stringer {
	return walletID(id)
}

type walletID string

func (w walletID) String() string {
	s := string(w)
	if len(s) <= 20 {
		return strings.ToValidUTF8(s, "X")
	}
	return fmt.Sprintf("%s~%s", strings.ToValidUTF8(s[:20], "X"), hash.Hashable(s).String())
}
