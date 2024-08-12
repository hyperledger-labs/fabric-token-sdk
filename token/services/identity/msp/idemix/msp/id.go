/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"bytes"
	"time"

	bccsp "github.com/IBM/idemix/bccsp/types"
	im "github.com/IBM/idemix/idemixmsp"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
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

var logger = logging.MustGetLogger("token-sdk.services.identity.msp.idemix")

// SchemaManager handles the various credential schemas. A credential schema
// contains information about the number of attributes, which attributes
// must be disclosed when creating proofs, the format of the attributes etc.
type SchemaManager interface {
	// SignerOpts returns the options for the passed arguments
	SignerOpts(schema string, ou *m.OrganizationUnit, role *m.MSPRole) (*bccsp.IdemixSignerOpts, error)
	// NymSignerOpts returns the options that `schema` uses to verify a nym signature
	NymSignerOpts(schema string) (*bccsp.IdemixNymSignerOpts, error)
	// EidNymAuditOpts returns the options that `sid` must use to audit an EIDNym
	EidNymAuditOpts(schema string, attrs [][]byte) (*bccsp.EidNymAuditOpts, error)
	// RhNymAuditOpts returns the options that `sid` must use to audit an RhNym
	RhNymAuditOpts(schema string, attrs [][]byte) (*bccsp.RhNymAuditOpts, error)
}

type Identity struct {
	NymPublicKey bccsp.Key
	Deserializer *Deserializer
	ID           *msp.IdentityIdentifier
	Role         *m.MSPRole
	OU           *m.OrganizationUnit
	// AssociationProof contains cryptographic proof that this identity
	// belongs to the MSP id.provider, i.e., it proves that the pseudonym
	// is constructed from a secret key on which the CA issued a credential.
	AssociationProof []byte
	VerificationType bccsp.VerificationType

	SchemaManager SchemaManager
	Schema        string
}

func NewIdentity(
	deserializer *Deserializer,
	NymPublicKey bccsp.Key,
	role *m.MSPRole,
	ou *m.OrganizationUnit,
	proof []byte,
	verificationType bccsp.VerificationType,
	SchemaManager SchemaManager,
	Schema string,
) (*Identity, error) {
	id := &Identity{}
	id.Deserializer = deserializer
	id.NymPublicKey = NymPublicKey
	id.Role = role
	id.OU = ou
	id.AssociationProof = proof
	id.VerificationType = verificationType
	id.SchemaManager = SchemaManager
	id.Schema = Schema

	raw, err := NymPublicKey.Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal nym public key")
	}
	id.ID = &msp.IdentityIdentifier{
		Mspid: deserializer.Name,
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
	return id.Deserializer.Name
}

func (id *Identity) GetOrganizationalUnits() []*msp.OUIdentifier {
	// we use the (serialized) public key of this MSP as the CertifiersIdentifier
	certifiersIdentifier, err := id.Deserializer.IssuerPublicKey.Bytes()
	if err != nil {
		logger.Errorf("Failed to marshal ipk in GetOrganizationalUnits: %s", err)
		return nil
	}

	return []*msp.OUIdentifier{{CertifiersIdentifier: certifiersIdentifier, OrganizationalUnitIdentifier: id.OU.OrganizationalUnitIdentifier}}
}

func (id *Identity) Validate() error {
	// logger.Debugf("Validating identity %+v", id)
	if id.GetMSPIdentifier() != id.Deserializer.Name {
		return errors.Errorf("the supplied identity does not belong to this msp")
	}
	return id.verifyProof()
}

func (id *Identity) Verify(msg []byte, sig []byte) error {
	opts, err := id.SchemaManager.NymSignerOpts(id.Schema)
	if err != nil {
		return err
	}
	opts.IssuerPK = id.Deserializer.IssuerPublicKey

	_, err = id.Deserializer.Csp.Verify(
		id.NymPublicKey,
		sig,
		msg,
		opts,
	)
	return err
}

func (id *Identity) SatisfiesPrincipal(principal *m.MSPPrincipal) error {
	return errors.Errorf("not supported")
}

func (id *Identity) Serialize() ([]byte, error) {
	serialized := &im.SerializedIdemixIdentity{}

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
	if len(id.Deserializer.NymEID) != 0 {
		metadata = &bccsp.IdemixSignerMetadata{
			EidNym: id.Deserializer.NymEID,
			RhNym:  id.Deserializer.RhNym,
		}
	}

	opts, err := id.SchemaManager.SignerOpts(id.Schema, id.OU, id.Role)
	if err != nil {
		return errors.Wrapf(err, "could obtain signer opts for schema '%s'", id.Schema)
	}
	opts.Epoch = id.Deserializer.Epoch
	opts.VerificationType = id.VerificationType
	opts.Metadata = metadata
	opts.RevocationPublicKey = id.Deserializer.RevocationPK

	valid, err := id.Deserializer.Csp.Verify(
		id.Deserializer.IssuerPublicKey,
		id.AssociationProof,
		nil,
		opts,
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
	opts, err := id.SchemaManager.NymSignerOpts(id.Schema)
	if err != nil {
		return nil, err
	}
	opts.Nym = id.NymKey
	opts.IssuerPK = id.Deserializer.IssuerPublicKey

	sig, err := id.Deserializer.Csp.Sign(
		id.UserKey,
		msg,
		opts,
	)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

type NymSignatureVerifier struct {
	CSP           bccsp.BCCSP
	IPK           bccsp.Key
	NymPK         bccsp.Key
	SchemaManager SchemaManager
	Schema        string
}

func (v *NymSignatureVerifier) Verify(message, sigma []byte) error {
	opts, err := v.SchemaManager.NymSignerOpts(v.Schema)
	if err != nil {
		return err
	}
	opts.IssuerPK = v.IPK

	_, err = v.CSP.Verify(
		v.NymPK,
		sigma,
		message,
		opts,
	)
	return err
}
