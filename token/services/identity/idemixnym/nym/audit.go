/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nym

import (
	"context"

	"github.com/IBM/idemix/bccsp/types"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
)

type AuditInfo struct {
	*crypto.AuditInfo
	IdemixSignature []byte
}

func (a *AuditInfo) Match(ctx context.Context, id []byte) error {
	valid, err := a.Csp.Verify(
		a.IssuerPublicKey,
		id,
		nil,
		&types.EidNymAuditOpts{
			AuditVerificationType: types.AuditExpectEidNym,
			EidIndex:              crypto.EIDIndex,
			EnrollmentID:          string(a.Attributes[crypto.EIDIndex]),
			RNymEid:               a.RhNymAuditData.Rand,
		},
	)
	if err != nil {
		return errors.Wrap(err, "error while verifying the nym eid")
	}
	if !valid {
		return errors.New("invalid nym eid")
	}

	return a.AuditInfo.Match(ctx, a.IdemixSignature)
}

// DeserializeAuditInfo deserializes the audit information from JSON.
func DeserializeAuditInfo(raw []byte) (*AuditInfo, error) {
	auditInfo := &AuditInfo{}
	err := auditInfo.FromBytes(raw)
	if err != nil {
		return nil, err
	}
	if auditInfo.AuditInfo == nil {
		return nil, errors.Errorf("failed to unmarshal, no audit info found")
	}
	if len(auditInfo.Attributes) == 0 {
		return nil, errors.Errorf("failed to unmarshal, no attributes found")
	}
	if len(auditInfo.IdemixSignature) == 0 {
		return nil, errors.Errorf("failed to unmarshal, no idemix signature found")
	}

	return auditInfo, nil
}
