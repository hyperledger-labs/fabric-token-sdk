/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"os"

	"github.com/pkg/errors"
)

// SignerConfig contains the crypto material to set up an idemix signing identity
type SignerConfig struct {
	// Cred represents the serialized idemix credential of the default signer
	Cred []byte `protobuf:"bytes,1,opt,name=Cred,proto3" json:"Cred,omitempty"`
	// Sk is the secret key of the default signer, corresponding to credential Cred
	Sk []byte `protobuf:"bytes,2,opt,name=Sk,proto3" json:"Sk,omitempty"`
	// OrganizationalUnitIdentifier defines the organizational unit the default signer is in
	OrganizationalUnitIdentifier string `protobuf:"bytes,3,opt,name=organizational_unit_identifier,json=organizationalUnitIdentifier" json:"organizational_unit_identifier,omitempty"`
	// Role defines whether the default signer is admin, member, peer, or client
	Role int `protobuf:"varint,4,opt,name=role,json=role" json:"role,omitempty"`
	// EnrollmentID contains the enrollment id of this signer
	EnrollmentID string `protobuf:"bytes,5,opt,name=enrollment_id,json=enrollmentId" json:"enrollment_id,omitempty"`
	// CRI contains a serialized Credential Revocation Information
	CredentialRevocationInformation []byte `protobuf:"bytes,6,opt,name=credential_revocation_information,json=credentialRevocationInformation,proto3" json:"credential_revocation_information,omitempty"`
	// RevocationHandle is the handle used to single out this credential and determine its revocation status
	RevocationHandle string `protobuf:"bytes,7,opt,name=revocation_handle,json=revocationHandle,proto3" json:"revocation_handle,omitempty"`
	// CurveID specifies the name of the Idemix curve to use, defaults to 'amcl.Fp256bn'
	CurveID string `protobuf:"bytes,8,opt,name=curve_id,json=curveID" json:"curveID,omitempty"`
}

const (
	ConfigDirUser                 = "user"
	ConfigFileIssuerPublicKey     = "IssuerPublicKey"
	ConfigFileRevocationPublicKey = "IssuerRevocationPublicKey"
	ConfigFileSigner              = "SignerConfig"
)

func ReadFile(file string) ([]byte, error) {
	fileCont, err := os.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read file %s", file)
	}

	return fileCont, nil
}
