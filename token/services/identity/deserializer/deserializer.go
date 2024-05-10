/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deserializer

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/interop/htlc"
	"github.com/pkg/errors"
)

type AuditInfo interface {
	EnrollmentID() string
	RevocationHandle() string
}

type AuditInfoDeserializer[T AuditInfo] interface {
	DeserializeAuditInfo([]byte) (T, error)
}

// EIDRHDeserializer returns enrollment IDs behind the owners of token
type EIDRHDeserializer[T AuditInfo] struct {
	AuditInfoDeserializer AuditInfoDeserializer[T]
}

// NewEIDRHDeserializer returns an enrollmentService
func NewEIDRHDeserializer[T AuditInfo](AuditInfoDeserializer AuditInfoDeserializer[T]) *EIDRHDeserializer[T] {
	return &EIDRHDeserializer[T]{
		AuditInfoDeserializer: AuditInfoDeserializer,
	}
}

// GetEnrollmentID returns the enrollmentID associated with the identity matched to the passed auditInfo
func (e *EIDRHDeserializer[T]) GetEnrollmentID(identity view.Identity, auditInfo []byte) (string, error) {
	ai, err := e.getAuditInfo(auditInfo)
	if err != nil {
		return "", err
	}
	return ai.EnrollmentID(), nil
}

// GetRevocationHandler returns the recoatopn handle associated with the identity matched to the passed auditInfo
func (e *EIDRHDeserializer[T]) GetRevocationHandler(identity view.Identity, auditInfo []byte) (string, error) {
	ai, err := e.getAuditInfo(auditInfo)
	if err != nil {
		return "", err
	}
	return ai.RevocationHandle(), nil
}

func (e *EIDRHDeserializer[T]) getAuditInfo(auditInfo []byte) (T, error) {
	var zeroAuditInfo T
	if len(auditInfo) == 0 {
		return zeroAuditInfo, nil
	}

	// Try to unmarshal it as ScriptInfo
	si := &htlc.ScriptInfo{}
	err := json.Unmarshal(auditInfo, si)
	if err == nil && (len(si.Sender) != 0 || len(si.Recipient) != 0) {
		if len(si.Recipient) != 0 {
			ai, err := e.AuditInfoDeserializer.DeserializeAuditInfo(si.Recipient)
			if err != nil {
				return zeroAuditInfo, errors.Wrapf(err, "failed unamrshalling audit info [%s]", auditInfo)
			}
			return ai, nil
		}

		return zeroAuditInfo, nil
	}

	ai, err := e.AuditInfoDeserializer.DeserializeAuditInfo(auditInfo)
	if err != nil {
		return zeroAuditInfo, errors.Wrapf(err, "failed unamrshalling audit info [%s]", auditInfo)
	}
	return ai, nil
}
