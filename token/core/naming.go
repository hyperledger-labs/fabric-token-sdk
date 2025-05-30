/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// TokenDriverNameType is the token driver name type
type TokenDriverNameType string

// TokenDriverNameFromPP returns the token driver name for the passed public parameters.
// The token driver name has the form <pp identifier>.v<pp version>.
func TokenDriverNameFromPP(pp driver.PublicParameters) TokenDriverNameType {
	return TokenDriverName(pp.Identifier(), pp.Version())
}

// TokenDriverName returns the token driver name for the passed identifier and version.
// The token driver name has the form <id>.v<ver>.
func TokenDriverName(id string, ver uint64) TokenDriverNameType {
	return TokenDriverNameType(fmt.Sprintf("%s.v%d", id, ver))
}
