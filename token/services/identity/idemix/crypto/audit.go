/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"context"

	csp "github.com/IBM/idemix/bccsp/types"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
)

type Schema = string

// SchemaManager handles the various credential schemas. A credential schema
// contains information about the number of attributes, which attributes
// must be disclosed when creating proofs, the format of the attributes etc.
type SchemaManager interface {
	// EidNymAuditOpts returns the options that `sid` must use to audit an EIDNym
	EidNymAuditOpts(schema string, attrs [][]byte) (*csp.EidNymAuditOpts, error)
	// RhNymAuditOpts returns the options that `sid` must use to audit an RhNym
	RhNymAuditOpts(schema string, attrs [][]byte) (*csp.RhNymAuditOpts, error)
}

type AuditInfo struct {
	EidNymAuditData *csp.AttrNymAuditData
	RhNymAuditData  *csp.AttrNymAuditData
	Attributes      [][]byte

	Csp             csp.BCCSP     `json:"-"`
	IssuerPublicKey csp.Key       `json:"-"`
	SchemaManager   SchemaManager `json:"-"`
	Schema          string
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

func (a *AuditInfo) Match(_ context.Context, id []byte) error {
	serialized := new(SerializedIdemixIdentity)
	err := proto.Unmarshal(id, serialized)
	if err != nil {
		return errors.Wrap(err, "could not deserialize a SerializedIdemixIdentity")
	}

	eidAuditOpts, err := a.SchemaManager.EidNymAuditOpts(a.Schema, a.Attributes)
	if err != nil {
		return errors.Wrap(err, "error while getting a RhNymAuditOpts")
	}
	eidAuditOpts.RNymEid = a.EidNymAuditData.Rand

	// Audit EID
	valid, err := a.Csp.Verify(
		a.IssuerPublicKey,
		serialized.Proof,
		nil,
		eidAuditOpts,
	)
	if err != nil {
		return errors.Wrap(err, "error while verifying the nym eid")
	}
	if !valid {
		return errors.New("invalid nym rh")
	}

	rhAuditOpts, err := a.SchemaManager.RhNymAuditOpts(a.Schema, a.Attributes)
	if err != nil {
		return errors.Wrap(err, "error while getting a RhNymAuditOpts")
	}
	rhAuditOpts.RNymRh = a.RhNymAuditData.Rand

	// Audit RH
	valid, err = a.Csp.Verify(
		a.IssuerPublicKey,
		serialized.Proof,
		nil,
		rhAuditOpts,
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
