/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"encoding/json"

	csp "github.com/IBM/idemix/bccsp/schemes"
	"github.com/golang/protobuf/proto"
	m "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
)

const (
	EIDIndex = 2
	RHIndex  = 3
)

type AuditInfo struct {
	*csp.NymEIDAuditData
	Attributes      [][]byte
	Csp             csp.BCCSP `json:"-"`
	IssuerPublicKey csp.Key   `json:"-"`
}

func DeserializeAuditInfo(raw []byte) (*AuditInfo, error) {
	auditInfo := &AuditInfo{}
	err := auditInfo.FromBytes(raw)
	if err != nil {
		return nil, err
	}
	return auditInfo, nil
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

func (a *AuditInfo) Match(id []byte) error {
	si := &m.SerializedIdentity{}
	err := proto.Unmarshal(id, si)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal to msp.SerializedIdentity{}")
	}

	serialized := new(m.SerializedIdemixIdentity)
	err = proto.Unmarshal(si.IdBytes, serialized)
	if err != nil {
		return errors.Wrap(err, "could not deserialize a SerializedIdemixIdentity")
	}

	valid, err := a.Csp.Verify(
		a.IssuerPublicKey,
		serialized.Proof,
		nil,
		&csp.EidNymAuditOpts{
			EidIndex:     EIDIndex,
			EnrollmentID: string(a.Attributes[EIDIndex]),
			RNymEid:      a.RNymEid,
		},
	)
	if err != nil {
		return errors.Wrapf(err, "error while verifying the nym eid for [%s]", a.EnrollmentID())
	}

	if !valid {
		return errors.New("invalid nym eid")
	}

	return nil
}
