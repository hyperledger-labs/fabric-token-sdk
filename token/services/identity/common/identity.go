/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import "github.com/hyperledger-labs/fabric-token-sdk/token/driver"

// GetIdentityFunc is a function that returns an Identity and its associated audit info for the given options
type GetIdentityFunc func(auditInfo []byte) (driver.Identity, []byte, error)

// LocalIdentity contains information about an identity
type LocalIdentity struct {
	Name         string `yaml:"name,omitempty"`
	EnrollmentID string
	Default      bool
	GetIdentity  GetIdentityFunc
	Remote       bool
}

// IdentityInfo implements the driver.IdentityInfo interface on top LocalIdentity
type IdentityInfo struct {
	localIdentity *LocalIdentity
	getIdentity   func() (driver.Identity, []byte, error)
}

func NewIdentityInfo(localIdentity *LocalIdentity, getIdentity func() (driver.Identity, []byte, error)) *IdentityInfo {
	return &IdentityInfo{localIdentity: localIdentity, getIdentity: getIdentity}
}

func (i *IdentityInfo) ID() string {
	return i.localIdentity.Name
}

func (i *IdentityInfo) EnrollmentID() string {
	return i.localIdentity.EnrollmentID
}

func (i *IdentityInfo) Get() (driver.Identity, []byte, error) {
	return i.getIdentity()
}

func (i *IdentityInfo) Remote() bool {
	return i.localIdentity.Remote
}
