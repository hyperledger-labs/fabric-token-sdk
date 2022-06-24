/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type Mapper interface {
	GetIdentityInfo(id string) driver.IdentityInfo
	MapToID(v interface{}) (view.Identity, string)
	RegisterIdentity(id string, typ string, path string) error
}

type Mappers map[driver.IdentityUsage]Mapper

func New() Mappers {
	return make(Mappers)
}

func (m Mappers) Put(usage driver.IdentityUsage, mapper Mapper) {
	m[usage] = mapper
}

func (m Mappers) SetIssuerRole(mapper Mapper) {
	m[driver.IssuerRole] = mapper
}

func (m Mappers) SetAuditorRole(mapper Mapper) {
	m[driver.AuditorRole] = mapper
}

func (m Mappers) SetOwnerRole(mapper Mapper) {
	m[driver.OwnerRole] = mapper
}
