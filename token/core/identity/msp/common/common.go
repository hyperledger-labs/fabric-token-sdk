package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/sig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type GetIdentityFunc func(opts *driver.IdentityOptions) (view.Identity, []byte, error)

type Resolver struct {
	Name         string `yaml:"name,omitempty"`
	Type         string `yaml:"type,omitempty"`
	EnrollmentID string
	GetIdentity  GetIdentityFunc
	Default      bool
}

type Info struct {
	id          string
	eid         string
	getIdentity func() (view.Identity, []byte, error)
}

func NewInfo(id string, eid string, getIdentity func() (view.Identity, []byte, error)) *Info {
	return &Info{id: id, eid: eid, getIdentity: getIdentity}
}

func (i *Info) ID() string {
	return i.id
}

func (i *Info) EnrollmentID() string {
	return i.eid
}

func (i *Info) Get() (view.Identity, []byte, error) {
	return i.getIdentity()
}

type SignerService interface {
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
