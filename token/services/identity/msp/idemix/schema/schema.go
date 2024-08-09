/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package schema

import (
	"fmt"

	msp "github.com/IBM/idemix"
	bccsp "github.com/IBM/idemix/bccsp/types"
	msp2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix/msp"
	m "github.com/hyperledger/fabric-protos-go/msp"
)

const (
	eidIdx = 2
	rhIdx  = 3
	skIdx  = 0
)

// attributeNames are the attribute names for the `w3c` schema
var attributeNames = []string{
	"_:c14n0 <http://www.w3.",
	"_:c14n0 <https://w3id.o",
	"_:c14n0 <https://w3id.o",
	"<did:key:z6MknntgQWCT8Z",
	"<did:key:z6MknntgQWCT8Z",
	"<did:key:z6MknntgQWCT8Z",
	"<did:key:z6MknntgQWCT8Z",
	"<did:key:z6MknntgQWCT8Z",
	"<did:key:z6MknntgQWCT8Z",
	"<did:key:z6MknntgQWCT8Z",
	"<did:key:z6MknntgQWCT8Z",
	"<did:key:z6MknntgQWCT8Z",
	"<did:key:z6MknntgQWCT8Z",
	"<did:key:z6MknntgQWCT8Z",
	"<did:key:z6MknntgQWCT8Z",
	"<did:key:z6MknntgQWCT8Z",
	"<https://issuer.oidp.us",
	"<https://issuer.oidp.us",
	"<https://issuer.oidp.us",
	"<https://issuer.oidp.us",
	"<https://issuer.oidp.us",
	"<https://issuer.oidp.us",
	"<https://issuer.oidp.us",
	"_:c14n0 <cbdccard:2_ou>",
	"_:c14n0 <cbdccard:3_rol",
	"_:c14n0 <cbdccard:4_eid",
	"_:c14n0 <cbdccard:5_rh>",
}

// DefaultManager implements the default schema for fabric:
// - 4 attributes (OU, Role, EID, RH)
// - all in bytes format except for Role
// - fixed positions
// - no other attributes
// - a "hidden" usk attribute at position 0
type DefaultManager struct {
}

func (*DefaultManager) NymSignerOpts(schema string) (*bccsp.IdemixNymSignerOpts, error) {
	switch schema {
	case "":
		return &bccsp.IdemixNymSignerOpts{}, nil
	case "w3c-v0.0.1":
		return &bccsp.IdemixNymSignerOpts{
			SKIndex: 24,
		}, nil
	}

	return nil, fmt.Errorf("unsupported schema '%s' for NymSignerOpts", schema)
}

func (*DefaultManager) PublicKeyImportOpts(schema string) (*bccsp.IdemixIssuerPublicKeyImportOpts, error) {
	switch schema {
	case "":
		return &bccsp.IdemixIssuerPublicKeyImportOpts{
			Temporary: true,
			AttributeNames: []string{
				msp.AttributeNameOU,
				msp.AttributeNameRole,
				msp.AttributeNameEnrollmentId,
				msp.AttributeNameRevocationHandle,
			},
		}, nil
	case "w3c-v0.0.1":
		return &bccsp.IdemixIssuerPublicKeyImportOpts{
			Temporary:      true,
			AttributeNames: append([]string{""}, attributeNames...),
		}, nil
	}

	return nil, fmt.Errorf("unsupported schema '%s' for PublicKeyImportOpts", schema)
}

func (*DefaultManager) SignerOpts(schema string, ou *m.OrganizationUnit, role *m.MSPRole) (*bccsp.IdemixSignerOpts, error) {
	switch schema {
	case "":
		return &bccsp.IdemixSignerOpts{
			Attributes: []bccsp.IdemixAttribute{
				{Type: bccsp.IdemixBytesAttribute, Value: []byte(ou.OrganizationalUnitIdentifier)},
				{Type: bccsp.IdemixIntAttribute, Value: msp2.GetIdemixRoleFromMSPRole(role)},
				{Type: bccsp.IdemixHiddenAttribute},
				{Type: bccsp.IdemixHiddenAttribute},
			},
			RhIndex:  rhIdx,
			EidIndex: eidIdx,
		}, nil
	case "w3c-v0.0.1":
		role_str := fmt.Sprintf(
			"_:c14n0 \u003ccbdccard:3_role\u003e \"%d\"^^\u003chttp://www.w3.org/2001/XMLSchema#integer\u003e .",
			msp2.GetIdemixRoleFromMSPRole(role),
		)

		idemixAttrs := []bccsp.IdemixAttribute{}
		for i := range attributeNames {
			if i == 25 {
				idemixAttrs = append(idemixAttrs, bccsp.IdemixAttribute{
					Type:  bccsp.IdemixBytesAttribute,
					Value: []byte(role_str),
				})
			} else if i == 24 {
				idemixAttrs = append(idemixAttrs, bccsp.IdemixAttribute{
					Type:  bccsp.IdemixBytesAttribute,
					Value: []byte(ou.OrganizationalUnitIdentifier),
				})
			} else {
				idemixAttrs = append(idemixAttrs, bccsp.IdemixAttribute{
					Type: bccsp.IdemixHiddenAttribute,
				})
			}
		}

		return &bccsp.IdemixSignerOpts{
			Attributes:       idemixAttrs,
			RhIndex:          27,
			EidIndex:         26,
			SKIndex:          24,
			VerificationType: bccsp.ExpectEidNymRhNym,
		}, nil
	}

	return nil, fmt.Errorf("unsupported schema '%s' for NymSignerOpts", schema)
}

func (*DefaultManager) RhNymAuditOpts(schema string, attrs [][]byte) (*bccsp.RhNymAuditOpts, error) {
	switch schema {
	case "":
		return &bccsp.RhNymAuditOpts{
			RhIndex:          rhIdx,
			SKIndex:          skIdx,
			RevocationHandle: string(attrs[rhIdx]),
		}, nil
	case "w3c-v0.0.1":
		return &bccsp.RhNymAuditOpts{
			RhIndex:          27,
			SKIndex:          24,
			RevocationHandle: string(attrs[27]),
		}, nil
	}

	return nil, fmt.Errorf("unsupported schema '%s' for NymSignerOpts", schema)
}

func (*DefaultManager) EidNymAuditOpts(schema string, attrs [][]byte) (*bccsp.EidNymAuditOpts, error) {
	switch schema {
	case "":
		return &bccsp.EidNymAuditOpts{
			EidIndex:     eidIdx,
			SKIndex:      skIdx,
			EnrollmentID: string(attrs[eidIdx]),
		}, nil
	case "w3c-v0.0.1":
		return &bccsp.EidNymAuditOpts{
			EidIndex:     26,
			SKIndex:      24,
			EnrollmentID: string(attrs[26]),
		}, nil
	}

	return nil, fmt.Errorf("unsupported schema '%s' for NymSignerOpts", schema)
}
