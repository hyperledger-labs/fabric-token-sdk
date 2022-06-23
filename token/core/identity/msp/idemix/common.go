/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	bccsp "github.com/IBM/idemix/bccsp/schemes"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	m "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
)

type Deserialized struct {
	id           *Identity
	NymPublicKey bccsp.Key
	si           *m.SerializedIdentity
	ou           *m.OrganizationUnit
	role         *m.MSPRole
}

type Common struct {
	Name            string
	IPK             []byte
	CSP             bccsp.BCCSP
	IssuerPublicKey bccsp.Key
	RevocationPK    bccsp.Key
	Epoch           int
	VerType         bccsp.VerificationType
	NymEID          []byte
}

func (c *Common) Deserialize(raw []byte, checkValidity bool) (*Deserialized, error) {
	return c.DeserializeWithNymEID(raw, checkValidity, nil)
}

func (c *Common) DeserializeWithNymEID(raw view.Identity, checkValidity bool, nymEID []byte) (*Deserialized, error) {
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
	NymPublicKey, err := c.CSP.KeyImport(
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

	// Role
	role := &m.MSPRole{}
	err = proto.Unmarshal(serialized.Role, role)
	if err != nil {
		return nil, errors.Wrap(err, "cannot deserialize the role of the identity")
	}

	idCommon := c
	if len(nymEID) != 0 {
		idCommon = &Common{
			Name:            c.Name,
			IPK:             c.IPK,
			CSP:             c.CSP,
			IssuerPublicKey: c.IssuerPublicKey,
			RevocationPK:    c.RevocationPK,
			Epoch:           c.Epoch,
			VerType:         c.VerType,
			NymEID:          nymEID,
		}
	}

	id := NewIdentityWithVerType(
		idCommon,
		NymPublicKey,
		role,
		ou,
		serialized.Proof,
		c.VerType,
	)
	if checkValidity {
		if err := id.Validate(); err != nil {
			return nil, errors.Wrap(err, "cannot deserialize, invalid identity")
		}
	}

	return &Deserialized{
		id:           id,
		NymPublicKey: NymPublicKey,
		si:           si,
		ou:           ou,
		role:         role,
	}, nil
}

func (c *Common) DeserializeAuditInfo(raw []byte) (*AuditInfo, error) {
	ai, err := DeserializeAuditInfo(raw)
	if err != nil {
		return nil, err
	}
	ai.Csp = c.CSP
	ai.IssuerPublicKey = c.IssuerPublicKey
	return ai, nil
}
