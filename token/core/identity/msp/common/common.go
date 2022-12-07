/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"reflect"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/sig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"
)

type KVS interface {
	Exists(id string) bool
	Put(id string, state interface{}) error
	Get(id string, state interface{}) error
	GetByPartialCompositeID(prefix string, attrs []string) (kvs.Iterator, error)
}

// GetIdentityFunc is a function that returns an Identity and its associated audit info for the given options
type GetIdentityFunc func(opts *driver.IdentityOptions) (view.Identity, []byte, error)

// Resolver contains information about an identity and how to retrieve it.
type Resolver struct {
	Name         string `yaml:"name,omitempty"`
	EnrollmentID string
	Default      bool
	GetIdentity  GetIdentityFunc
}

// IdentityInfo implements the driver.IdentityInfo interface.
type IdentityInfo struct {
	id          string
	eid         string
	getIdentity func() (view.Identity, []byte, error)
}

func NewIdentityInfo(id string, eid string, getIdentity func() (view.Identity, []byte, error)) *IdentityInfo {
	return &IdentityInfo{id: id, eid: eid, getIdentity: getIdentity}
}

func (i *IdentityInfo) ID() string {
	return i.id
}

func (i *IdentityInfo) EnrollmentID() string {
	return i.eid
}

func (i *IdentityInfo) Get() (view.Identity, []byte, error) {
	return i.getIdentity()
}

type SignerService interface {
	IsMe(view.Identity) bool
	RegisterSigner(identity view.Identity, signer driver.Signer, verifier driver.Verifier) error
}

type BinderService interface {
	Bind(longTerm view.Identity, ephemeral view.Identity) error
}

type EnrollmentService interface {
	GetEnrollmentID(auditInfo []byte) (string, error)
}

type DeserializerManager interface {
	AddDeserializer(deserializer sig.Deserializer)
}

func GetDeserializerManager(sp view2.ServiceProvider) (DeserializerManager, error) {
	dm, err := sp.GetService(reflect.TypeOf((*DeserializerManager)(nil)))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed looking up deserializer manager")
	}
	return dm.(DeserializerManager), nil
}
