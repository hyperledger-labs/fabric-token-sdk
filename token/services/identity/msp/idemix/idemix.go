/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	bccsp "github.com/IBM/idemix/bccsp/types"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	m "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
)

type Identity struct {
	Identity           *MSPIdentity
	NymPublicKey       bccsp.Key
	SerializedIdentity *m.SerializedIdentity
	OU                 *m.OrganizationUnit
	Role               *m.MSPRole
}

type Idemix struct {
	Name            string
	Ipk             []byte
	Csp             bccsp.BCCSP
	IssuerPublicKey bccsp.Key
	RevocationPK    bccsp.Key
	Epoch           int
	VerType         bccsp.VerificationType
	NymEID          []byte
	RhNym           []byte
}

func (c *Idemix) Deserialize(raw []byte, checkValidity bool) (*Identity, error) {
	return c.DeserializeAgainstNymEID(raw, checkValidity, nil)
}

func (c *Idemix) DeserializeAgainstNymEID(raw []byte, checkValidity bool, nymEID []byte) (*Identity, error) {
	si := &m.SerializedIdentity{}
	err := proto.Unmarshal(raw, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to msp.SerializedIdentity{}")
	}

	serialized := new(m.SerializedIdemixIdentity)
	err = proto.Unmarshal(si.IdBytes, serialized)
	if err != nil {
		return nil, errors.Wrap(err, "could not deserialize a SerializedIdemixIdentity")
	}
	if serialized.NymX == nil || serialized.NymY == nil {
		return nil, errors.Errorf("unable to deserialize idemix identity: pseudonym is invalid")
	}

	// Import NymPublicKey
	var rawNymPublicKey []byte
	rawNymPublicKey = append(rawNymPublicKey, serialized.NymX...)
	rawNymPublicKey = append(rawNymPublicKey, serialized.NymY...)
	NymPublicKey, err := c.Csp.KeyImport(
		rawNymPublicKey,
		&bccsp.IdemixNymPublicKeyImportOpts{Temporary: true},
	)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to import nym public key")
	}

	// OU
	ou := &m.OrganizationUnit{}
	err = proto.Unmarshal(serialized.Ou, ou)
	if err != nil {
		return nil, errors.Wrap(err, "cannot deserialize the OU of the identity")
	}

	// RoleAttribute
	role := &m.MSPRole{}
	err = proto.Unmarshal(serialized.Role, role)
	if err != nil {
		return nil, errors.Wrap(err, "cannot deserialize the role of the identity")
	}

	idemix := c
	if len(nymEID) != 0 {
		idemix = &Idemix{
			Name:            c.Name,
			Ipk:             c.Ipk,
			Csp:             c.Csp,
			IssuerPublicKey: c.IssuerPublicKey,
			RevocationPK:    c.RevocationPK,
			Epoch:           c.Epoch,
			VerType:         c.VerType,
			NymEID:          nymEID,
		}
	}

	id, err := NewMSPIdentityWithVerType(
		idemix,
		NymPublicKey,
		role,
		ou,
		serialized.Proof,
		c.VerType,
	)
	if err != nil {
		return nil, errors.Wrap(err, "cannot deserialize")
	}
	if checkValidity {
		if err := id.Validate(); err != nil {
			return nil, errors.Wrap(err, "cannot deserialize, invalid identity")
		}
	}

	return &Identity{
		Identity:           id,
		NymPublicKey:       NymPublicKey,
		SerializedIdentity: si,
		OU:                 ou,
		Role:               role,
	}, nil
}

func (c *Idemix) DeserializeAuditInfo(raw []byte) (*AuditInfo, error) {
	ai, err := DeserializeAuditInfo(raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed deserializing audit info [%s]", string(raw))
	}
	ai.Csp = c.Csp
	ai.IssuerPublicKey = c.IssuerPublicKey
	return ai, nil
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
