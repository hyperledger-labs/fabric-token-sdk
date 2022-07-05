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
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/exchange"
	"github.com/pkg/errors"
)

const ScriptTypeExchange = "exchange" // exchange script

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
	si, err := identity.UnmarshallRawOwner(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal RawOwner")
	}
	if si.Type == identity.SerializedIdentityType {
		return d.ownerDeserializer.DeserializeVerifier(id)
	}
	if si.Type == ScriptTypeExchange {
		return d.getExchangeVerifier(si.Identity)
	}
	return nil, errors.Errorf("failed to deserialize RawOwner: Unknown owner type %s", si.Type)
}

func (d *deserializer) getExchangeVerifier(raw []byte) (driver.Verifier, error) {
	script := &exchange.Script{}
	err := json.Unmarshal(raw, script)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal RawOwner as a pledge script")
	}
	v := &ExchangeVerifier{}
	v.Sender, err = d.ownerDeserializer.DeserializeVerifier(script.Sender)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal the identity of the sender")
	}
	v.Recipient, err = d.ownerDeserializer.DeserializeVerifier(script.Recipient)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal the identity of the issuer")
	}
	v.Deadline = script.Deadline
	v.HashInfo.Hash = script.HashInfo.Hash
	v.HashInfo.HashFunc = script.HashInfo.HashFunc
	v.HashInfo.HashEncoding = script.HashInfo.HashEncoding
	return v, nil
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
	return string(auditInfo), nil
}
