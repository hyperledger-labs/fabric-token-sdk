/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deserializer

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/pkg/errors"
)

// EIDRHDeserializer returns enrollment IDs behind the owners of token
type EIDRHDeserializer struct {
	deserializers map[identity.Type]driver2.AuditInfoDeserializer
}

// NewEIDRHDeserializer returns an enrollmentService
func NewEIDRHDeserializer() *EIDRHDeserializer {
	return &EIDRHDeserializer{
		deserializers: map[string]driver2.AuditInfoDeserializer{},
	}
}

func (e *EIDRHDeserializer) AddDeserializer(typ string, d driver2.AuditInfoDeserializer) {
	e.deserializers[typ] = d
}

// GetEnrollmentID returns the enrollmentID associated with the identity matched to the passed auditInfo
func (e *EIDRHDeserializer) GetEnrollmentID(identity driver.Identity, auditInfo []byte) (string, error) {
	ai, err := e.getAuditInfo(identity, auditInfo)
	if err != nil {
		return "", err
	}
	return ai.EnrollmentID(), nil
}

// GetRevocationHandler returns the revocation handle associated with the identity matched to the passed auditInfo
func (e *EIDRHDeserializer) GetRevocationHandler(identity driver.Identity, auditInfo []byte) (string, error) {
	ai, err := e.getAuditInfo(identity, auditInfo)
	if err != nil {
		return "", err
	}
	return ai.RevocationHandle(), nil
}

func (e *EIDRHDeserializer) GetEIDAndRH(identity driver.Identity, auditInfo []byte) (string, string, error) {
	ai, err := e.getAuditInfo(identity, auditInfo)
	if err != nil {
		return "", "", err
	}
	return ai.EnrollmentID(), ai.RevocationHandle(), nil
}

func (e *EIDRHDeserializer) getAuditInfo(id driver.Identity, auditInfo []byte) (driver2.AuditInfo, error) {
	if len(auditInfo) == 0 {
		return nil, errors.Errorf("nil audit info")
	}

	si, err := identity.UnmarshalTypedIdentity(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedIdentity")
	}
	d, ok := e.deserializers[si.Type]
	if !ok {
		return nil, errors.Errorf("no deserializer found for [%s]", si.Type)
	}
	res, err := d.DeserializeAuditInfo(auditInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to deserialize audit info for identity type [%s]", si.Type)
	}
	return res, nil
}
