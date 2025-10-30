/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"context"
	"fmt"

	csp "github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	crypto2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/schema"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
)

type Deserializer struct {
	*crypto2.Deserializer
}

// NewDeserializer returns a new deserializer for the idemix ExpectEidNymRhNym verification strategy
func NewDeserializer(
	ipk []byte,
	curveID math.CurveID,
) (*Deserializer, error) {
	if len(ipk) == 0 {
		return nil, errors.New("invalid ipk")
	}
	cryptoProvider, err := crypto2.NewBCCSPWithDummyKeyStore(curveID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to instantiate crypto provider for curve [%d]", curveID)
	}
	return NewDeserializerWithProvider(
		schema.NewDefaultManager(),
		schema.DefaultSchema,
		ipk, csp.ExpectEidNymRhNym, nil, cryptoProvider)
}

// NewDeserializerWithProvider returns a new serialized for the passed arguments
func NewDeserializerWithProvider(
	sm SchemaManager,
	schema Schema,
	ipk []byte,
	verType csp.VerificationType,
	nymEID []byte,
	cryptoProvider csp.BCCSP,
) (*Deserializer, error) {
	return NewDeserializerWithBCCSP(sm, schema, ipk, verType, nymEID, cryptoProvider)
}

func NewDeserializerWithBCCSP(
	sm SchemaManager,
	schema Schema,
	ipk []byte,
	verType csp.VerificationType,
	nymEID []byte,
	cryptoProvider csp.BCCSP,
) (*Deserializer, error) {
	logger.Debugf("Setting up Idemix-based deserailizer instance")

	// Import Issuer Public Key
	if len(ipk) == 0 {
		return nil, errors.Errorf("no issuer public key provided")
	}
	var issuerPublicKey csp.Key
	// get the opts from the schema manager
	opts, err := sm.PublicKeyImportOpts(schema)
	if err != nil {
		return nil, errors.Wrapf(err, "could not obtain PublicKeyImportOpts for schema '%s'", schema)
	}
	issuerPublicKey, err = cryptoProvider.KeyImport(
		ipk,
		opts,
	)
	if err != nil {
		return nil, err
	}

	return &Deserializer{
		Deserializer: &crypto2.Deserializer{
			Ipk:             ipk,
			Csp:             cryptoProvider,
			IssuerPublicKey: issuerPublicKey,
			VerType:         verType,
			NymEID:          nymEID,
			SchemaManager:   sm,
			Schema:          schema,
		},
	}, nil
}

func (d *Deserializer) DeserializeVerifier(ctx context.Context, id driver.Identity) (driver.Verifier, error) {
	identity, err := d.Deserialize(ctx, id)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to deserialize identity")
	}

	return &crypto2.NymSignatureVerifier{
		CSP:           d.Csp,
		IPK:           d.IssuerPublicKey,
		NymPK:         identity.NymPublicKey,
		SchemaManager: d.SchemaManager,
		Schema:        d.Schema,
	}, nil
}

func (d *Deserializer) DeserializeVerifierAgainstNymEID(raw []byte, nymEID []byte) (driver.Verifier, error) {
	identity, err := d.DeserializeAgainstNymEID(raw, nymEID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to deserialize identity")
	}

	return &crypto2.NymSignatureVerifier{
		CSP:           d.Csp,
		IPK:           d.IssuerPublicKey,
		NymPK:         identity.NymPublicKey,
		SchemaManager: d.SchemaManager,
		Schema:        d.Schema,
	}, nil
}

func (i *Deserializer) DeserializeAuditInfo(ctx context.Context, raw []byte) (driver2.AuditInfo, error) {
	return i.Deserializer.DeserializeAuditInfo(ctx, raw)
}

func (i *Deserializer) GetAuditInfoMatcher(ctx context.Context, owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	return i.Deserializer.DeserializeAuditInfo(ctx, auditInfo)
}

func (i *Deserializer) MatchIdentity(ctx context.Context, id driver.Identity, ai []byte) error {
	matcher, err := i.Deserializer.DeserializeAuditInfo(ctx, ai)
	if err != nil {
		return errors.WithMessagef(err, "failed to deserialize audit info")
	}
	return matcher.Match(ctx, id)
}

func (i *Deserializer) Info(ctx context.Context, id []byte, auditInfoRaw []byte) (string, error) {
	eid := ""
	if len(auditInfoRaw) != 0 {
		err := i.MatchIdentity(ctx, id, auditInfoRaw)
		if err != nil {
			return "", errors.WithMessagef(err, "failed to get audit info matcher")
		}
		ai, err := i.DeserializeAuditInfo(ctx, auditInfoRaw)
		if err != nil {
			return "", errors.Wrapf(err, "failed to deserialize audit info")
		}
		eid = ai.EnrollmentID()
	}

	return fmt.Sprintf("Idemix: [%s][%s]", eid, driver.Identity(id).UniqueID()), nil
}

func (i *Deserializer) String() string {
	return fmt.Sprintf("Idemix with IPK [%s]", utils.Hashable(i.Ipk).String())
}

type AuditInfoDeserializer struct{}

func (c *AuditInfoDeserializer) DeserializeAuditInfo(ctx context.Context, raw []byte) (driver2.AuditInfo, error) {
	ai, err := crypto2.DeserializeAuditInfo(raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed deserializing audit info [%s]", string(raw))
	}
	return ai, nil
}
