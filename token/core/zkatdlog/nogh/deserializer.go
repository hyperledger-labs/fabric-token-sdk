/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"sync"

	idemix2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
	"github.com/pkg/errors"
)

// VerifierDES deserializes verifiers
// A verifier checks the validity of a signature against the identity
// associated with the verifier
type VerifierDES interface {
	DeserializeVerifier(id view.Identity) (driver.Verifier, error)
}

// AuditDES deserializes raw bytes into a matcher, which allows an auditor to match an identity to an enrollment ID
type AuditDES interface {
	GetOwnerMatcher(raw []byte) (driver.Matcher, error)
}

// Deserializer deserializes verifiers associated with issuers, owners, and auditors
type Deserializer struct {
	auditorDeserializer VerifierDES
	ownerDeserializer   VerifierDES
	issuerDeserializer  VerifierDES
	auditDeserializer   AuditDES
}

// NewDeserializer returns a deserializer
func NewDeserializer(pp *crypto.PublicParams) (*Deserializer, error) {
	if pp == nil {
		return nil, errors.New("failed to get deserializer: nil public parameters")
	}
	idemixDes, err := idemix.NewDeserializer(pp.IdemixIssuerPK, pp.IdemixCurveID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting idemix deserializer for passed public params [%d]", pp.IdemixCurveID)
	}

	return &Deserializer{
		auditorDeserializer: &x509.MSPIdentityDeserializer{},
		issuerDeserializer:  &x509.MSPIdentityDeserializer{},
		ownerDeserializer:   interop.NewDeserializer(owner.NewTypedIdentityDeserializer(idemixDes)),
		auditDeserializer:   idemixDes,
	}, nil
}

// GetOwnerVerifier deserializes the verifier for the passed owner identity
func (d *Deserializer) GetOwnerVerifier(id view.Identity) (driver.Verifier, error) {
	return d.ownerDeserializer.DeserializeVerifier(id)
}

// GetIssuerVerifier deserializes the verifier for the passed issuer identity
func (d *Deserializer) GetIssuerVerifier(id view.Identity) (driver.Verifier, error) {
	return d.issuerDeserializer.DeserializeVerifier(id)
}

// GetAuditorVerifier deserializes the verifier for the passed auditor identity
func (d *Deserializer) GetAuditorVerifier(id view.Identity) (driver.Verifier, error) {
	return d.auditorDeserializer.DeserializeVerifier(id)
}

// GetOwnerMatcher returns a matcher that allows auditors to match an identity to an enrollment ID
func (d *Deserializer) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	return d.auditDeserializer.GetOwnerMatcher(raw)
}

// DeserializerProvider provides the deserializer matching zkatdlog public parameters
type DeserializerProvider struct {
	oldHash []byte
	des     *Deserializer
	mux     sync.Mutex
}

// NewDeserializerProvider returns a DeserializerProvider
func NewDeserializerProvider() *DeserializerProvider {
	return &DeserializerProvider{}
}

// Deserialize returns the deserializer matching the passed public parameters
func (d *DeserializerProvider) Deserialize(params *crypto.PublicParams) (driver.Deserializer, error) {
	d.mux.Lock()
	defer d.mux.Unlock()

	newHash, err := params.ComputeHash()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to compute the hash of the public params")
	}
	logger.Debugf("Deserialize: [%s][%s]", base64.StdEncoding.EncodeToString(newHash), base64.StdEncoding.EncodeToString(d.oldHash))
	if bytes.Equal(d.oldHash, newHash) {
		return d.des, nil
	}

	des, err := NewDeserializer(params)
	if err != nil {
		return nil, err
	}
	d.des = des
	d.oldHash = newHash
	return des, nil
}

// EnrollmentService returns enrollment IDs behind the owners of token
type EnrollmentService struct {
}

// NewEnrollmentIDDeserializer returns an enrollmentService
func NewEnrollmentIDDeserializer() *EnrollmentService {
	return &EnrollmentService{}
}

// GetEnrollmentID returns the enrollmentID associated with the identity matched to the passed auditInfo
func (e *EnrollmentService) GetEnrollmentID(auditInfo []byte) (string, error) {
	ai, err := e.getAuditInfo(auditInfo)
	if err != nil {
		return "", err
	}
	if ai == nil {
		return "", nil
	}
	return ai.EnrollmentID(), nil
}

// GetRevocationHandler returns the recoatopn handle associated with the identity matched to the passed auditInfo
func (e *EnrollmentService) GetRevocationHandler(auditInfo []byte) (string, error) {
	ai, err := e.getAuditInfo(auditInfo)
	if err != nil {
		return "", err
	}
	if ai == nil {
		return "", nil
	}
	return ai.RevocationHandle(), nil
}

func (e *EnrollmentService) getAuditInfo(auditInfo []byte) (*idemix2.AuditInfo, error) {
	if len(auditInfo) == 0 {
		return nil, nil
	}

	// Try to unmarshal it as ScriptInfo
	si := &interop.ScriptInfo{}
	err := json.Unmarshal(auditInfo, si)
	if err == nil && (len(si.Sender) != 0 || len(si.Recipient) != 0) {
		if len(si.Recipient) != 0 {
			ai := &idemix2.AuditInfo{}
			if err := ai.FromBytes(si.Recipient); err != nil {
				return nil, errors.Wrapf(err, "failed unamrshalling audit info [%s]", auditInfo)
			}
			return ai, nil
		}

		return nil, nil
	}

	ai := &idemix2.AuditInfo{}
	if err := ai.FromBytes(auditInfo); err != nil {
		return nil, errors.Wrapf(err, "failed unamrshalling audit info [%s]", auditInfo)
	}
	return ai, nil
}
