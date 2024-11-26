/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"fmt"

	csp "github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	msp2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix/msp"
	"github.com/pkg/errors"
)

type Deserializer struct {
	*msp2.Deserializer
}

// NewEidNymRhNymDeserializer returns a new deserializer that expects EID and RH Nyms identities.
// The returned deserializer checks the validly of the deserialized identities.
func NewEidNymRhNymDeserializer(
	sm SchemaManager,
	schema string,
	ipk []byte,
	curveID math.CurveID,
) (*Deserializer, error) {
	cryptoProvider, err := msp2.NewBCCSPWithDummyKeyStore(curveID, curveID == math.BLS12_381_BBS)
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
	logger.Debugf("Setting up Idemix-based MSP instance")

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
		Deserializer: &msp2.Deserializer{
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
		return nil, err
	}

	return &msp2.NymSignatureVerifier{
		CSP:           d.Deserializer.Csp,
		IPK:           d.Deserializer.IssuerPublicKey,
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
	identity, err := d.Deserializer.DeserializeAgainstNymEID(raw, nymEID)
	if err != nil {
		return nil, err
	}

	return &msp2.NymSignatureVerifier{
		CSP:           d.Deserializer.Csp,
		IPK:           d.Deserializer.IssuerPublicKey,
		NymPK:         identity.NymPublicKey,
		SchemaManager: d.SchemaManager,
		Schema:        d.Schema,
	}, nil
}

func (d *Deserializer) String() string {
	return fmt.Sprintf("Idemix with IPK [%s]", hash.Hashable(d.Ipk).String())
}

type AuditInfoDeserializer struct{}

func (c *AuditInfoDeserializer) DeserializeAuditInfo(raw []byte) (driver2.AuditInfo, error) {
	ai, err := msp2.DeserializeAuditInfo(raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed deserializing audit info [%s]", string(raw))
	}
	return ai, nil
}
