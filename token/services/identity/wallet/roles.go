/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wallet

import (
	"fmt"
	"strconv"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	db2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/wallet/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

// Roles is a map of Role, one for each identity role
type Roles map[identity.RoleType]identity.Role

// NewRoles returns a new Roles maps
func NewRoles() Roles {
	return make(Roles)
}

// Register associates an instance of Role to a given identifier
func (m Roles) Register(usage identity.RoleType, role identity.Role) {
	m[usage] = role
}

func (m Roles) ToWalletRegistries(logger logging.Logger, db driver.WalletStore) map[identity.RoleType]Registry {
	res := make(map[identity.RoleType]Registry, len(m))
	for roleType, role := range m {
		roleAsString, ok := identity.RoleTypeStrings[roleType]
		if !ok {
			roleAsString = strconv.Itoa(int(roleType))
		}
		res[roleType] = db2.NewWalletRegistry(logger.Named(fmt.Sprintf("identity.%s-wallet-registry", roleAsString)), role, db)
	}
	return res
}
