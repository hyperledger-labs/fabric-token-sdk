/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package exchange

import (
	"crypto"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
)

type Script struct {
	Sender       view.Identity
	Recipient    view.Identity
	Deadline     time.Time
	Hash         []byte
	HashFunc     crypto.Hash
	HashEncoding encoding.Encoding
}
