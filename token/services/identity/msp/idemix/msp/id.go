/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"bytes"
	"time"

	bccsp "github.com/IBM/idemix/bccsp/types"
	"github.com/IBM/idemix/common/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	m "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric/msp"
	"github.com/pkg/errors"
)

const (
	EIDIndex                  = 2
	RHIndex                   = 3
	SignerConfigFull          = "SignerConfigFull"
	IdentityConfigurationType = "idemix"
)

var logger = flogging.MustGetLogger("token-sdk.services.identity.msp.idemix")

type Identity struct {
	NymPublicKey bccsp.Key
	Idemix       *Deserializer
	ID           *msp.IdentityIdentifier
	Role         *m.MSPRole
	OU           *m.OrganizationUnit
	// AssociationProof contains cryptographic proof that this identity
	// belongs to the MSP id.provider, i.e., it proves that the pseudonym
	// is constructed from a secret key on which the CA issued a credential.
	AssociationProof []byte
	VerificationType bccsp.VerificationType
}

func NewIdentity(idemix *Deserializer, NymPublicKey bccsp.Key, role *m.MSPRole, ou *m.OrganizationUnit, proof []byte, verificationType bccsp.VerificationType) (*Identity, error) {
	id := &Identity{}
	id.Idemix = idemix
	id.NymPublicKey = NymPublicKey
	id.Role = role
	id.OU = ou
	id.AssociationProof = proof
	id.VerificationType = verificationType

	raw, err := NymPublicKey.Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal nym public key")
	}
	id.ID = &msp.IdentityIdentifier{
		Mspid: idemix.Name,
		Id:    bytes.NewBuffer(raw).String(),
	}

	return id, nil
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
	return id.Idemix.Name
}

func (id *Identity) GetOrganizationalUnits() []*msp.OUIdentifier {
	// we use the (serialized) public key of this MSP as the CertifiersIdentifier
	certifiersIdentifier, err := id.Idemix.IssuerPublicKey.Bytes()
	if err != nil {
		logger.Errorf("Failed to marshal ipk in GetOrganizationalUnits: %s", err)
		return nil
	}

	return []*msp.OUIdentifier{{CertifiersIdentifier: certifiersIdentifier, OrganizationalUnitIdentifier: id.OU.OrganizationalUnitIdentifier}}
}

func (id *Identity) Validate() error {
	// logger.Debugf("Validating identity %+v", id)
	if id.GetMSPIdentifier() != id.Idemix.Name {
		return errors.Errorf("the supplied identity does not belong to this msp")
	}
	return id.verifyProof()
}

func (id *Identity) Verify(msg []byte, sig []byte) error {
	_, err := id.Idemix.Csp.Verify(
		id.NymPublicKey,
		sig,
		msg,
		&bccsp.IdemixNymSignerOpts{
			IssuerPK: id.Idemix.IssuerPublicKey,
		},
	)
	return err
}

func (id *Identity) SatisfiesPrincipal(principal *m.MSPPrincipal) error {
	return errors.Errorf("not supported")
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
	var metadata *bccsp.IdemixSignerMetadata
	if len(id.Idemix.NymEID) != 0 {
		metadata = &bccsp.IdemixSignerMetadata{
			EidNym: id.Idemix.NymEID,
			RhNym:  id.Idemix.RhNym,
		}
	}

	valid, err := id.Idemix.Csp.Verify(
		id.Idemix.IssuerPublicKey,
		id.AssociationProof,
		nil,
		&bccsp.IdemixSignerOpts{
			RevocationPublicKey: id.Idemix.RevocationPK,
			Attributes: []bccsp.IdemixAttribute{
				{Type: bccsp.IdemixBytesAttribute, Value: []byte(id.OU.OrganizationalUnitIdentifier)},
				{Type: bccsp.IdemixIntAttribute, Value: GetIdemixRoleFromMSPRole(id.Role)},
				{Type: bccsp.IdemixHiddenAttribute},
				{Type: bccsp.IdemixHiddenAttribute},
			},
			RhIndex:          RHIndex,
			EidIndex:         EIDIndex,
			Epoch:            id.Idemix.Epoch,
			VerificationType: id.VerificationType,
			Metadata:         metadata,
		},
	)
	if err == nil && !valid {
		return errors.Errorf("unexpected condition, an error should be returned for an invalid signature")
	}

	return err
}

type SigningIdentity struct {
	*Identity    `json:"-"`
	Cred         []byte
	UserKey      bccsp.Key `json:"-"`
	NymKey       bccsp.Key `json:"-"`
	EnrollmentId string
}

func (id *SigningIdentity) Sign(msg []byte) ([]byte, error) {
	// logger.Debugf("Idemix identity %s is signing", id.GetIdentifier())

	sig, err := id.Idemix.Csp.Sign(
		id.UserKey,
		msg,
		&bccsp.IdemixNymSignerOpts{
			Nym:      id.NymKey,
			IssuerPK: id.Idemix.IssuerPublicKey,
		},
	)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

type NymSignatureVerifier struct {
	CSP   bccsp.BCCSP
	IPK   bccsp.Key
	NymPK bccsp.Key
}

func (v *NymSignatureVerifier) Verify(message, sigma []byte) error {
	_, err := v.CSP.Verify(
		v.NymPK,
		sigma,
		message,
		&bccsp.IdemixNymSignerOpts{
			IssuerPK: v.IPK,
		},
	)
	return err
}
