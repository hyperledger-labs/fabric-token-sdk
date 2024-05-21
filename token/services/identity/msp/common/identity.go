/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import "github.com/hyperledger-labs/fabric-token-sdk/token/driver"

type IdentityOptions struct {
	EIDExtension bool
	AuditInfo    []byte
}

// GetIdentityFunc is a function that returns an Identity and its associated audit info for the given options
type GetIdentityFunc func(opts *IdentityOptions) (driver.Identity, []byte, error)

// IdentityInfo implements the driver.IdentityInfo interface.
type IdentityInfo struct {
	id          string
	eid         string
	getIdentity func() (driver.Identity, []byte, error)
	remote      bool
}

func NewIdentityInfo(id string, eid string, remote bool, getIdentity func() (driver.Identity, []byte, error)) *IdentityInfo {
	return &IdentityInfo{id: id, eid: eid, remote: remote, getIdentity: getIdentity}
}

func (i *IdentityInfo) ID() string {
	return i.id
}

func (i *IdentityInfo) EnrollmentID() string {
	return i.eid
}

func (i *IdentityInfo) Get() (driver.Identity, []byte, error) {
	return i.getIdentity()
}

func (i *IdentityInfo) Remote() bool {
	return i.remote
}
