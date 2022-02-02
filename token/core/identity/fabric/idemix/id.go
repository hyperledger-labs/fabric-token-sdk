/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"bytes"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"

	bccsp "github.com/IBM/idemix/bccsp/schemes"
	csp "github.com/IBM/idemix/bccsp/schemes"
	"github.com/golang/protobuf/proto"
	m "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric/msp"
	"github.com/pkg/errors"
)

type Identity struct {
	NymPublicKey bccsp.Key
	Common       *Common
	ID           *msp.IdentityIdentifier
	Role         *m.MSPRole
	OU           *m.OrganizationUnit
	// AssociationProof contains cryptographic proof that this identity
	// belongs to the MSP id.provider, i.e., it proves that the pseudonym
	// is constructed from a secret key on which the CA issued a credential.
	AssociationProof []byte
	VerificationType bccsp.VerificationType
}

func NewIdentity(provider *Common, NymPublicKey bccsp.Key, role *m.MSPRole, ou *m.OrganizationUnit, proof []byte) *Identity {
	return NewIdentityWithVerType(provider, NymPublicKey, role, ou, proof, bccsp.ExpectEidNym)
}

func NewIdentityWithVerType(common *Common, NymPublicKey bccsp.Key, role *m.MSPRole, ou *m.OrganizationUnit, proof []byte, verificationType bccsp.VerificationType) *Identity {
	id := &Identity{}
	id.Common = common
	id.NymPublicKey = NymPublicKey
	id.Role = role
	id.OU = ou
	id.AssociationProof = proof
	id.VerificationType = verificationType

	raw, err := NymPublicKey.Bytes()
	if err != nil {
		panic(fmt.Sprintf("unexpected condition, failed marshalling nym public key [%s]", err))
	}
	id.ID = &msp.IdentityIdentifier{
		Mspid: common.Name,
		Id:    bytes.NewBuffer(raw).String(),
	}

	return id
}

func (id *Identity) Anonymous() bool {
	return true
}

func (id *Identity) ExpiresAt() time.Time {
	// Idemix MSP currently does not use expiration dates or revocation,
	// so we return the zero time to indicate this.
	return time.Time{}
}

func (id *Identity) GetIdentifier() *msp.IdentityIdentifier {
	return id.ID
}

func (id *Identity) GetMSPIdentifier() string {
	return id.Common.Name
}

func (id *Identity) GetOrganizationalUnits() []*msp.OUIdentifier {
	// we use the (serialized) public key of this MSP as the CertifiersIdentifier
	certifiersIdentifier, err := id.Common.IssuerPublicKey.Bytes()
	if err != nil {
		logger.Errorf("Failed to marshal ipk in GetOrganizationalUnits: %s", err)
		return nil
	}

	return []*msp.OUIdentifier{{CertifiersIdentifier: certifiersIdentifier, OrganizationalUnitIdentifier: id.OU.OrganizationalUnitIdentifier}}
}

func (id *Identity) Validate() error {
	// logger.Debugf("Validating identity %+v", id)
	if id.GetMSPIdentifier() != id.Common.Name {
		return errors.Errorf("the supplied identity does not belong to this msp")
	}
	return id.verifyProof()
}

func (id *Identity) Verify(msg []byte, sig []byte) error {
	_, err := id.Common.CSP.Verify(
		id.NymPublicKey,
		sig,
		msg,
		&csp.IdemixNymSignerOpts{
			IssuerPK: id.Common.IssuerPublicKey,
		},
	)
	return err
}

func (id *Identity) SatisfiesPrincipal(principal *m.MSPPrincipal) error {
	panic("not implemented yet")
}

func (id *Identity) Serialize() ([]byte, error) {
	serialized := &m.SerializedIdemixIdentity{}

	raw, err := id.NymPublicKey.Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "could not serialize nym of identity %s", id.ID)
	}
	// This is an assumption on how the underlying idemix implementation work.
	// TODO: change this in future version
	serialized.NymX = raw[:len(raw)/2]
	serialized.NymY = raw[len(raw)/2:]
	ouBytes, err := proto.Marshal(id.OU)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal OU of identity %s", id.ID)
	}

	roleBytes, err := proto.Marshal(id.Role)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal role of identity %s", id.ID)
	}

	serialized.Ou = ouBytes
	serialized.Role = roleBytes
	serialized.Proof = id.AssociationProof

	idemixIDBytes, err := proto.Marshal(serialized)
	if err != nil {
		return nil, err
	}

	sID := &m.SerializedIdentity{Mspid: id.GetMSPIdentifier(), IdBytes: idemixIDBytes}
	idBytes, err := proto.Marshal(sID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal a SerializedIdentity structure for identity %s", id.ID)
	}

	return idBytes, nil
}

func (id *Identity) verifyProof() error {
	// Verify signature
	var metadata *csp.IdemixSignerMetadata
	if len(id.Common.NymEID) != 0 {
		metadata = &csp.IdemixSignerMetadata{
			NymEID: id.Common.NymEID,
		}
	}

	valid, err := id.Common.CSP.Verify(
		id.Common.IssuerPublicKey,
		id.AssociationProof,
		nil,
		&csp.IdemixSignerOpts{
			RevocationPublicKey: id.Common.RevocationPK,
			Attributes: []csp.IdemixAttribute{
				{Type: csp.IdemixBytesAttribute, Value: []byte(id.OU.OrganizationalUnitIdentifier)},
				{Type: csp.IdemixIntAttribute, Value: getIdemixRoleFromMSPRole(id.Role)},
				{Type: csp.IdemixHiddenAttribute},
				{Type: csp.IdemixHiddenAttribute},
			},
			RhIndex:          RHIndex,
			EidIndex:         EIDIndex,
			Epoch:            id.Common.Epoch,
			VerificationType: id.VerificationType,
			Metadata:         metadata,
		},
	)
	if err == nil && !valid {
		panic("unexpected condition, an error should be returned for an invalid signature")
	}

	return err
}

type SigningIdentity struct {
	*Identity
	Cred         []byte
	UserKey      bccsp.Key
	NymKey       bccsp.Key
	EnrollmentId string
}

func (id *SigningIdentity) Sign(msg []byte) ([]byte, error) {
	// logger.Debugf("Idemix identity %s is signing", id.GetIdentifier())

	sig, err := id.Common.CSP.Sign(
		id.UserKey,
		msg,
		&csp.IdemixNymSignerOpts{
			Nym:      id.NymKey,
			IssuerPK: id.Common.IssuerPublicKey,
		},
	)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

func (id *SigningIdentity) GetPublicVersion() driver.Identity {
	return id.Identity
}
