/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"context"
	"fmt"

	msp "github.com/IBM/idemix"
	csp "github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	crypto2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/pkg/errors"
)

type Deserializer struct {
	*crypto2.Deserializer
}

// NewDeserializer returns a new deserializer for the idemix ExpectEidNymRhNym verification strategy
func NewDeserializer(ipk []byte, curveID math.CurveID) (*Deserializer, error) {
	if len(ipk) == 0 {
		return nil, errors.New("invalid ipk")
	}
	cryptoProvider, err := crypto2.NewBCCSPWithDummyKeyStore(curveID, curveID == math.BLS12_381_BBS)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to instantiate crypto provider for curve [%d]", curveID)
	}
	return NewDeserializer(sm, schema, ipk, csp.ExpectEidNymRhNym, nil, cryptoProvider)
}

// NewDeserializer returns a new deserializer for the passed arguments.
// The returned deserializer checks the validly of the deserialized identities.
func NewDeserializer(
	sm SchemaManager,
	schema string,
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

func (d *Deserializer) DeserializeVerifier(raw driver.Identity) (driver.Verifier, error) {
	identity, err := d.Deserialize(raw)
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

func (d *Deserializer) DeserializeAuditInfo(raw []byte) (driver2.AuditInfo, error) {
	return d.Deserializer.DeserializeAuditInfo(raw)
}

func (d *Deserializer) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	return d.Deserializer.DeserializeAuditInfo(raw)
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

func (i *Deserializer) DeserializeSigner(raw []byte) (driver.Signer, error) {
	return nil, errors.New("not supported")
}

func (i *Deserializer) DeserializeAuditInfo(raw []byte) (driver2.AuditInfo, error) {
	return i.Deserializer.DeserializeAuditInfo(raw)
}

func (i *Deserializer) GetAuditInfoMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	return i.Deserializer.DeserializeAuditInfo(auditInfo)
}

func (i *Deserializer) MatchIdentity(id driver.Identity, auditInfo []byte) error {
	matcher, err := i.Deserializer.DeserializeAuditInfo(auditInfo)
	if err != nil {
		return errors.WithMessagef(err, "failed to deserialize audit info")
	}
	return matcher.Match(id)
}

func (i *Deserializer) GetAuditInfo(ctx context.Context, raw []byte, p driver.AuditInfoProvider) ([][]byte, error) {
	auditInfo, err := p.GetAuditInfo(ctx, raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", driver.Identity(raw).String())
	}
	return [][]byte{auditInfo}, nil
}

func (i *Deserializer) Info(id []byte, auditInfoRaw []byte) (string, error) {
	eid := ""
	if len(auditInfoRaw) != 0 {
		err := i.MatchIdentity(id, auditInfoRaw)
		if err != nil {
			return "", errors.WithMessagef(err, "failed to get audit info matcher")
		}
		ai, err := i.DeserializeAuditInfo(auditInfoRaw)
		if err != nil {
			return "", errors.Wrapf(err, "failed to deserialize audit info")
		}
		eid = ai.EnrollmentID()
	}

	return fmt.Sprintf("Idemix: [%s][%s]", eid, driver.Identity(id).UniqueID()), nil
}

func (i *Deserializer) String() string {
	return fmt.Sprintf("Idemix with IPK [%s]", hash.Hashable(i.Ipk).String())
}

type AuditInfoDeserializer struct{}

func (c *AuditInfoDeserializer) DeserializeAuditInfo(raw []byte) (driver2.AuditInfo, error) {
	ai, err := crypto2.DeserializeAuditInfo(raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed deserializing audit info [%s]", string(raw))
	}
	return ai, nil
}
