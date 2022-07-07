/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"bytes"
	"encoding/json"
	"sync"

	idemix2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/exchange"
	"github.com/pkg/errors"
)

type VerifierDES interface {
	DeserializeVerifier(id view.Identity) (driver.Verifier, error)
}

type AuditDES interface {
	DeserializeAuditInfo(raw []byte) (driver.Matcher, error)
}

type deserializer struct {
	auditorDeserializer VerifierDES
	ownerDeserializer   VerifierDES
	issuerDeserializer  VerifierDES
	auditDeserializer   AuditDES
}

func NewDeserializer(pp *crypto.PublicParams) (*deserializer, error) {
	idemixDes, err := idemix.NewDeserializer(pp.IdemixPK, pp.IdemixCurve)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting idemix deserializer for passed public params")
	}

	return &deserializer{
		auditorDeserializer: &x509.MSPIdentityDeserializer{},
		issuerDeserializer:  &x509.MSPIdentityDeserializer{},
		ownerDeserializer:   identity.NewRawOwnerIdentityDeserializer(idemixDes),
		auditDeserializer:   idemixDes,
	}, nil
}

func (d *deserializer) GetOwnerVerifier(id view.Identity) (driver.Verifier, error) {
	si, err := identity.UnmarshallRawOwner(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal RawOwner")
	}
	if si.Type == identity.SerializedIdentityType {
		return d.ownerDeserializer.DeserializeVerifier(id)
	}
	if si.Type == exchange.ScriptTypeExchange {
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

func (d *deserializer) GetIssuerVerifier(id view.Identity) (driver.Verifier, error) {
	return d.issuerDeserializer.DeserializeVerifier(id)
}

func (d *deserializer) GetAuditorVerifier(id view.Identity) (driver.Verifier, error) {
	return d.auditorDeserializer.DeserializeVerifier(id)
}

func (d *deserializer) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	return d.auditDeserializer.DeserializeAuditInfo(raw)
}

type DeserializerProvider struct {
	oldHash []byte
	des     *deserializer
	mux     sync.Mutex
}

func NewDeserializerProvider() *DeserializerProvider {
	return &DeserializerProvider{}
}

func (d *DeserializerProvider) Deserialize(params *crypto.PublicParams) (driver.Deserializer, error) {
	d.mux.Lock()
	defer d.mux.Unlock()

	logger.Infof("Deserialize: [%s][%s]", params.Hash, d.oldHash)
	if bytes.Equal(d.oldHash, params.Hash) {
		return d.des, nil
	}

	des, err := NewDeserializer(params)
	if err != nil {
		return nil, err
	}
	d.des = des
	d.oldHash = params.Hash
	return des, nil
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
	si := &ScriptInfo{}
	err := json.Unmarshal(auditInfo, si)
	if err == nil && (len(si.Sender) != 0 || len(si.Recipient) != 0) {
		return "", nil
	}

	ai := &idemix2.AuditInfo{}
	if err := ai.FromBytes(auditInfo); err != nil {
		return "", errors.Wrapf(err, "failed unamrshalling audit info [%s]", auditInfo)
	}
	return ai.EnrollmentID(), nil
}
