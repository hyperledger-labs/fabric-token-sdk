/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
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

var logger = logging.MustGetLogger("token-sdk.services.identity.idemix")

type Identity struct {
	NymPublicKey bccsp.Key
	Idemix       *Deserializer
	// AssociationProof contains cryptographic proof that this identity is valid.
	AssociationProof []byte
	VerificationType bccsp.VerificationType
}

func NewIdentity(idemix *Deserializer, nymPublicKey bccsp.Key, proof []byte, verificationType bccsp.VerificationType) (*Identity, error) {
	id := &Identity{}
	id.Idemix = idemix
	id.NymPublicKey = nymPublicKey
	id.AssociationProof = proof
	id.VerificationType = verificationType
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
	// This is an assumption on how the underlying idemix implementation work.
	// TODO: change this in future version
	serialized.NymX = raw[:len(raw)/2]
	serialized.NymY = raw[len(raw)/2:]
	serialized.Proof = id.AssociationProof

	idemixIDBytes, err := proto.Marshal(serialized)
	if err != nil {
		return nil, err
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
	*Identity    `json:"-"`
	Cred         []byte
	UserKey      bccsp.Key `json:"-"`
	NymKey       bccsp.Key `json:"-"`
	EnrollmentId string
}

func (id *SigningIdentity) Sign(msg []byte) ([]byte, error) {
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
