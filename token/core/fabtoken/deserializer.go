/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/interop"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type VerifierDES interface {
	DeserializeVerifier(id view.Identity) (driver.Verifier, error)
}

type deserializer struct {
	auditorDeserializer VerifierDES
	ownerDeserializer   VerifierDES
	issuerDeserializer  VerifierDES
}

func NewDeserializer() *deserializer {
	return &deserializer{
		auditorDeserializer: &x509.MSPIdentityDeserializer{},
		issuerDeserializer:  &x509.MSPIdentityDeserializer{},
		ownerDeserializer:   identity.NewRawOwnerIdentityDeserializer(&x509.MSPIdentityDeserializer{}),
	}
}

func (d *deserializer) GetOwnerVerifier(id view.Identity) (driver.Verifier, error) {
	return interop.NewDeserializer(d.ownerDeserializer).GetOwnerVerifier(id)
}

func (d *deserializer) GetIssuerVerifier(id view.Identity) (driver.Verifier, error) {
	return d.issuerDeserializer.DeserializeVerifier(id)
}

func (d *deserializer) GetAuditorVerifier(id view.Identity) (driver.Verifier, error) {
	return d.auditorDeserializer.DeserializeVerifier(id)
}

func (d *deserializer) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	panic("not supported")
}

type enrollmentService struct {
}

func NewEnrollmentIDDeserializer() *enrollmentService {
	return &enrollmentService{}
}

func (e *enrollmentService) GetEnrollmentID(auditInfo []byte) (string, error) {
	if len(auditInfo) == 0 {
		return "", nil
	}

	// Try to unmarshal it as ScriptInfo
	si := &interop.ScriptInfo{}
	err := json.Unmarshal(auditInfo, si)
	if err == nil && (len(si.Sender) != 0 || len(si.Recipient) != 0) {
		if len(si.Recipient) != 0 {
			return string(si.Recipient), nil
		}
		return "", nil
	}
	return string(auditInfo), nil
}
