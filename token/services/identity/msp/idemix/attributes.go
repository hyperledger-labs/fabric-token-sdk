/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	m "github.com/hyperledger/fabric-protos-go/msp"
)

// RoleAttribute : Represents a IdemixRole
type RoleAttribute int32

// The expected roles are 4; We can combine them using a bitmask
const (
	MEMBER RoleAttribute = 1
	ADMIN  RoleAttribute = 2
	CLIENT RoleAttribute = 4
	PEER   RoleAttribute = 8
	// Next role values: 16, 32, 64 ...
)

func (role RoleAttribute) getValue() int {
	return int(role)
}

// CheckRole Prove that the desired role is contained or not in the bitmask
func CheckRole(bitmask int, role RoleAttribute) bool {
	return (bitmask & role.getValue()) == role.getValue()
}

// GetIdemixRoleFromMSPRole gets a MSP RoleAttribute type and returns the integer value
func GetIdemixRoleFromMSPRole(role *m.MSPRole) int {
	return GetIdemixRoleFromMSPRoleType(role.GetRole())
}

// GetIdemixRoleFromMSPRoleType gets a MSP role type and returns the integer value
func GetIdemixRoleFromMSPRoleType(rtype m.MSPRole_MSPRoleType) int {
	return GetIdemixRoleFromMSPRoleValue(int(rtype))
}

// GetIdemixRoleFromMSPRoleValue Receives a MSP role value and returns the idemix equivalent
func GetIdemixRoleFromMSPRoleValue(role int) int {
	switch role {
	case int(m.MSPRole_ADMIN):
		return ADMIN.getValue()
	case int(m.MSPRole_CLIENT):
		return CLIENT.getValue()
	case int(m.MSPRole_MEMBER):
		return MEMBER.getValue()
	case int(m.MSPRole_PEER):
		return PEER.getValue()
	default:
		return MEMBER.getValue()
	}
}
