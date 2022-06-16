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

// VerifierDES is the interface for verifiers' deserializer
// A verifier checks the validity of a signature against the identity
// associated with the verifier
type VerifierDES interface {
	DeserializeVerifier(id view.Identity) (driver.Verifier, error)
}

// deserializer deserializes verifiers associated with issuers, owners, and auditors
type deserializer struct {
	auditorDeserializer VerifierDES
	ownerDeserializer   VerifierDES
	issuerDeserializer  VerifierDES
}

// NewDeserializer returns a deserializer
func NewDeserializer() *deserializer {
	return &deserializer{
		auditorDeserializer: &x509.MSPIdentityDeserializer{},
		issuerDeserializer:  &x509.MSPIdentityDeserializer{},
		ownerDeserializer:   identity.NewRawOwnerIdentityDeserializer(&x509.MSPIdentityDeserializer{}),
	}
}

// GetOwnerVerifier deserializes the verifier for the passed owner identity
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
		return nil, errors.Errorf("failed to unmarshal RawOwner as a exchange script")
	}
	v := &exchange.ExchangeVerifier{}
	v.Sender, err = d.ownerDeserializer.DeserializeVerifier(script.Sender)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal the identity of the sender in the exchange script")
	}
	v.Recipient, err = d.ownerDeserializer.DeserializeVerifier(script.Recipient)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal the identity of the recipient in the exchange script")
	}
	v.Deadline = script.Deadline
	v.HashInfo.Hash = script.HashInfo.Hash
	v.HashInfo.HashFunc = script.HashInfo.HashFunc
	v.HashInfo.HashEncoding = script.HashInfo.HashEncoding
	return v, nil
}

// GetIssuerVerifier deserializes the verifier for the passed issuer identity
func (d *deserializer) GetIssuerVerifier(id view.Identity) (driver.Verifier, error) {
	return d.issuerDeserializer.DeserializeVerifier(id)
}

// GetAuditorVerifier deserializes the verifier for the passed auditor identity
func (d *deserializer) GetAuditorVerifier(id view.Identity) (driver.Verifier, error) {
	return d.auditorDeserializer.DeserializeVerifier(id)
}

// GetOwnerMatcher is not needed in fabtoken, as identities are in the clear
func (d *deserializer) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	panic("not supported")
}

// enrollmentService returns enrollment IDs behind the owners of token
type enrollmentService struct {
}

// NewEnrollmentIDDeserializer returns an enrollmentService
func NewEnrollmentIDDeserializer() *enrollmentService {
	return &enrollmentService{}
}

type ScriptInfo struct {
	Sender    []byte
	Recipient []byte
}

// GetEnrollmentID returns the enrollmentID associated with the identity matched to the passed auditInfo
func (e *enrollmentService) GetEnrollmentID(auditInfo []byte) (string, error) {
	if len(auditInfo) == 0 {
		return "", nil
	}

	// Try to unmarshal it as ScriptInfo
	si := &ScriptInfo{}
	err := json.Unmarshal(auditInfo, si)
	if err == nil && (len(si.Sender) != 0 || len(si.Recipient) != 0) {
		if len(si.Recipient) != 0 {
			return string(si.Recipient), nil
		}
		return "", nil
	}
	return string(auditInfo), nil
}
