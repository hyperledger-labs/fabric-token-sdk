/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

// PublicParameters is a data structure to wrap public parameters and other host related information
type PublicParameters struct {
	TMSID token.TMSID
	Path  string
	Raw   []byte
}
