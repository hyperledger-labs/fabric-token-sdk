/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"context"

	bccsp "github.com/IBM/idemix/bccsp/types"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
)

type DeserializedIdentity struct {
	Identity     *Identity
	NymPublicKey bccsp.Key
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

func (d *Deserializer) Deserialize(_ context.Context, raw []byte) (*DeserializedIdentity, error) {
	return d.DeserializeAgainstNymEID(raw, nil)
}

func (d *Deserializer) DeserializeAgainstNymEID(identity []byte, nymEID []byte) (*DeserializedIdentity, error) {
	if len(identity) == 0 {
		return nil, errors.Errorf("empty identity")
	}
	serialized := new(SerializedIdemixIdentity)
	err := proto.Unmarshal(identity, serialized)
	if err != nil {
		return nil, errors.Wrap(err, "could not deserialize a SerializedIdemixIdentity")
	}
	if len(serialized.NymPublicKey) == 0 {
		return nil, errors.Errorf("unable to deserialize idemix identity: pseudonym's public key is empty")
	}

	// match schema
	if serialized.Schema != d.Schema {
		return nil, errors.Errorf("unable to deserialize idemix identity: schema does not match [%s]!=[%s]", serialized.Schema, d.Schema)
	}

	// Import NymPublicKey
	NymPublicKey, err := d.Csp.KeyImport(
		serialized.NymPublicKey,
		&bccsp.IdemixNymPublicKeyImportOpts{Temporary: true},
	)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to import nym public key")
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

	id, err := NewIdentity(idemix, NymPublicKey, serialized.Proof, d.VerType, d.SchemaManager, d.Schema)
	if err != nil {
		return nil, errors.Wrap(err, "cannot deserialize")
	}
	if err := id.Validate(); err != nil {
		return nil, errors.Wrap(err, "cannot deserialize, invalid identity")
	}

	return &DeserializedIdentity{
		Identity:     id,
		NymPublicKey: NymPublicKey,
	}, nil
}

func (d *Deserializer) DeserializeAuditInfo(_ context.Context, raw []byte) (*AuditInfo, error) {
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
