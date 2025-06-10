/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// TokenDriverIdentifier is the token driver identifier
type TokenDriverIdentifier string

// DriverIdentifierFromPP returns the token driver identifier for the passed public parameters.
// The token driver identifier has the form <token driver nam>.v<token driver version>.
func DriverIdentifierFromPP(pp driver.PublicParameters) TokenDriverIdentifier {
	return DriverIdentifier(pp.TokenDriverName(), pp.TokenDriverVersion())
}

// DriverIdentifier returns the token driver identifier for the passed name and version.
// The token driver name has the form <name>.v<ver>.
func DriverIdentifier(name driver.TokenDriverName, ver driver.TokenDriverVersion) TokenDriverIdentifier {
	return TokenDriverIdentifier(fmt.Sprintf("%s.v%d", name, ver))
}
