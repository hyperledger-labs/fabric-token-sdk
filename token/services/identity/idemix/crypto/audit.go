/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	csp "github.com/IBM/idemix/bccsp/types"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/pkg/errors"
)

type AuditInfo struct {
	EidNymAuditData *csp.AttrNymAuditData
	RhNymAuditData  *csp.AttrNymAuditData
	Attributes      [][]byte
	Csp             csp.BCCSP `json:"-"`
	IssuerPublicKey csp.Key   `json:"-"`
}

func (a *AuditInfo) Bytes() ([]byte, error) {
	return json.Marshal(a)
}

func (a *AuditInfo) FromBytes(raw []byte) error {
	return json.Unmarshal(raw, a)
}

func (a *AuditInfo) EnrollmentID() string {
	return string(a.Attributes[2])
}

func (a *AuditInfo) RevocationHandle() string {
	return string(a.Attributes[3])
}

func (a *AuditInfo) Match(id []byte) error {
	serialized := new(SerializedIdemixIdentity)
	err := proto.Unmarshal(id, serialized)
	if err != nil {
		return errors.Wrap(err, "could not deserialize a SerializedIdemixIdentity")
	}

	// Audit EID
	valid, err := a.Csp.Verify(
		a.IssuerPublicKey,
		serialized.Proof,
		nil,
		&csp.EidNymAuditOpts{
			EidIndex:     EIDIndex,
			EnrollmentID: string(a.Attributes[EIDIndex]),
			RNymEid:      a.EidNymAuditData.Rand,
		},
	)
	if err != nil {
		return errors.Wrap(err, "error while verifying the nym eid")
	}
	if !valid {
		return errors.New("invalid nym rh")
	}

	// Audit RH
	valid, err = a.Csp.Verify(
		a.IssuerPublicKey,
		serialized.Proof,
		nil,
		&csp.RhNymAuditOpts{
			RhIndex:          RHIndex,
			RevocationHandle: string(a.Attributes[RHIndex]),
			RNymRh:           a.RhNymAuditData.Rand,
		},
	)
	if err != nil {
		return errors.Wrap(err, "error while verifying the nym rh")
	}
	if !valid {
		return errors.New("invalid nym eid")
	}

	return nil
}

func DeserializeAuditInfo(raw []byte) (*AuditInfo, error) {
	auditInfo := &AuditInfo{}
	err := auditInfo.FromBytes(raw)
	if err != nil {
		return nil, err
	}
	if len(auditInfo.Attributes) == 0 {
		return nil, errors.Errorf("failed to unmarshal, no attributes found [%s]", string(raw))
	}
	return auditInfo, nil
}
