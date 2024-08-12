/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	bccsp "github.com/IBM/idemix/bccsp/types"
	im "github.com/IBM/idemix/idemixmsp"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	m "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
)

type DeserializedIdentity struct {
	Identity           *Identity
	NymPublicKey       bccsp.Key
	SerializedIdentity *m.SerializedIdentity
	OU                 *m.OrganizationUnit
	Role               *m.MSPRole
}

type Deserializer struct {
	Name            string
	Ipk             []byte
	Csp             bccsp.BCCSP
	IssuerPublicKey bccsp.Key
	RevocationPK    bccsp.Key
	Epoch           int
	VerType         bccsp.VerificationType
	NymEID          []byte
	RhNym           []byte
	SchemaManager   SchemaManager
	Schema          string
}

func (d *Deserializer) Deserialize(raw []byte) (*DeserializedIdentity, error) {
	return d.DeserializeAgainstNymEID(raw, nil)
}

func (d *Deserializer) DeserializeAgainstNymEID(raw []byte, nymEID []byte) (*DeserializedIdentity, error) {
	si := &m.SerializedIdentity{}
	err := proto.Unmarshal(raw, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to msp.SerializedIdentity{}")
	}

	serialized := new(im.SerializedIdemixIdentity)
	err = proto.Unmarshal(si.IdBytes, serialized)
	if err != nil {
		return nil, errors.Wrap(err, "could not deserialize a SerializedIdemixIdentity")
	}
	if serialized.NymX == nil || serialized.NymY == nil {
		return nil, errors.Errorf("unable to deserialize idemix identity: pseudonym is invalid")
	}

	// match schema
	if serialized.Schema != d.Schema {
		return nil, errors.Errorf("unable to deserialize idemix identity: schema does not match [%s]!=[%s]", serialized.Schema, d.Schema)
	}

	// Import NymPublicKey
	var rawNymPublicKey []byte
	rawNymPublicKey = append(rawNymPublicKey, serialized.NymX...)
	rawNymPublicKey = append(rawNymPublicKey, serialized.NymY...)
	NymPublicKey, err := d.Csp.KeyImport(
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

	idemix := d
	if len(nymEID) != 0 {
		idemix = &Deserializer{
			Name:            d.Name,
			Ipk:             d.Ipk,
			Csp:             d.Csp,
			IssuerPublicKey: d.IssuerPublicKey,
			RevocationPK:    d.RevocationPK,
			Epoch:           d.Epoch,
			VerType:         d.VerType,
			NymEID:          nymEID,
			SchemaManager:   d.SchemaManager,
		}
	}

	id, err := NewIdentity(
		idemix,
		NymPublicKey,
		role,
		ou,
		serialized.Proof,
		d.VerType,
		d.SchemaManager,
		d.Schema,
	)
	if err != nil {
		return nil, errors.Wrap(err, "cannot deserialize")
	}
	if err := id.Validate(); err != nil {
		return nil, errors.Wrap(err, "cannot deserialize, invalid identity")
	}

	return &DeserializedIdentity{
		Identity:           id,
		NymPublicKey:       NymPublicKey,
		SerializedIdentity: si,
		OU:                 ou,
		Role:               role,
	}, nil
}

func (d *Deserializer) DeserializeAuditInfo(raw []byte) (*AuditInfo, error) {
	ai, err := DeserializeAuditInfo(raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed deserializing audit info [%s]", string(raw))
	}
	ai.Csp = d.Csp
	ai.IssuerPublicKey = d.IssuerPublicKey
	ai.SchemaManager = d.SchemaManager
	ai.Schema = d.Schema
	return ai, nil
}
