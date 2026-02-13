/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

// PublicParameters models the public parameters of a token management system.
type PublicParameters struct {
	// TMSID is the ID of the token management system.
	TMSID token.TMSID
	// Path is the path to the public parameters.
	Path string
	// Raw is the raw public parameters.
	Raw []byte
}
