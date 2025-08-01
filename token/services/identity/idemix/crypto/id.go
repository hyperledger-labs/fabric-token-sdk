/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"context"

	bccsp "github.com/IBM/idemix/bccsp/types"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

const (
	EIDIndex         = 2
	RHIndex          = 3
	SignerConfigFull = "SignerConfigFull"
)

var logger = logging.MustGetLogger()

type Identity struct {
	NymPublicKey bccsp.Key
	Idemix       *Deserializer
	// AssociationProof contains cryptographic proof that this identity is valid.
	AssociationProof []byte
	VerificationType bccsp.VerificationType

	// Schema related fields
	SchemaManager SchemaManager
	Schema        Schema
}

func NewIdentity(ctx context.Context, idemix *Deserializer, nymPublicKey bccsp.Key, proof []byte, verificationType bccsp.VerificationType, schemaManager SchemaManager, schema Schema) (*Identity, error) {
	id := &Identity{
		Idemix:           idemix,
		NymPublicKey:     nymPublicKey,
		AssociationProof: proof,
		VerificationType: verificationType,
		SchemaManager:    schemaManager,
		Schema:           schema,
	}
	return id, nil
}

func (id *Identity) Validate() error {
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

func (id *Identity) Serialize() ([]byte, error) {
	serialized := &SerializedIdemixIdentity{}

	raw, err := id.NymPublicKey.Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "could not serialize nym")
	}
	serialized.NymPublicKey = raw
	serialized.Proof = id.AssociationProof

	idemixIDBytes, err := proto.Marshal(serialized)
	if err != nil {
		return nil, errors.Wrapf(err, "could not serialize idemix identity")
	}
	return idemixIDBytes, nil
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
				{Type: bccsp.IdemixHiddenAttribute},
				{Type: bccsp.IdemixHiddenAttribute},
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
	*Identity `json:"-"`
	CSP       bccsp.BCCSP `json:"-"`

	EnrollmentId string
	NymKeySKI    []byte
	UserKeySKI   []byte
}

func (id *SigningIdentity) Sign(msg []byte) ([]byte, error) {
	nymKey, err := id.CSP.GetKey(id.NymKeySKI)
	if err != nil {
		return nil, errors.Wrap(err, "cannot find nym secret key")
	}

	// Load the user key
	userKey, err := id.CSP.GetKey(id.UserKeySKI)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve user key with ski [%s]", id.UserKeySKI)
	}

	sig, err := id.Idemix.Csp.Sign(
		userKey,
		msg,
		&bccsp.IdemixNymSignerOpts{
			Nym:      nymKey,
			IssuerPK: id.Idemix.IssuerPublicKey,
		},
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
	Schema        Schema
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
