/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"fmt"

	"github.com/IBM/idemix"
	bccsp "github.com/IBM/idemix/bccsp/schemes"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.core.identity.msp.idemix")

type Deserializer struct {
	*Idemix
}

// NewDeserializer returns a new deserializer for the idemix ExpectEidNymRhNym verification strategy
func NewDeserializer(ipk []byte, revocationPK []byte, curveID math.CurveID) (*Deserializer, error) {
	cryptoProvider, err := NewBCCSP(curveID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to instantiate crypto provider for curve [%d]", curveID)
	}
	return NewDeserializerWithProvider(ipk, revocationPK, bccsp.ExpectEidNymRhNym, nil, cryptoProvider)
}

// NewDeserializerWithProvider returns a new serialized for the passed arguments
func NewDeserializerWithProvider(
	ipk []byte,
	revocationPK []byte,
	verType bccsp.VerificationType,
	nymEID []byte,
	cryptoProvider bccsp.BCCSP,
) (*Deserializer, error) {
	logger.Debugf("Setting up Idemix-based MSP instance")
	if len(ipk) == 0 {
		return nil, errors.Errorf("empty issuer public key")
	}
	if len(revocationPK) == 0 {
		return nil, errors.Errorf("empty revocation public key")
	}

	// Import Issuer Public Key
	issuerPublicKey, err := cryptoProvider.KeyImport(
		ipk,
		&bccsp.IdemixIssuerPublicKeyImportOpts{
			Temporary: true,
			AttributeNames: []string{
				idemix.AttributeNameOU,
				idemix.AttributeNameRole,
				idemix.AttributeNameEnrollmentId,
				idemix.AttributeNameRevocationHandle,
			},
		},
	)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to input issuer public key")
	}

	revocationPublicKey, err := cryptoProvider.KeyImport(
		revocationPK,
		&bccsp.IdemixRevocationPublicKeyImportOpts{Temporary: true},
	)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to import revocation public key")
	}

	return &Deserializer{
		Idemix: &Idemix{
			IPK:             ipk,
			CSP:             cryptoProvider,
			IssuerPublicKey: issuerPublicKey,
			VerType:         verType,
			NymEID:          nymEID,
			RevocationPK:    revocationPublicKey,
		},
	}, nil
}

// DeserializeVerifier expects raw to contain and Idemix MSP serialized identity.
// The deserializer first checks that the identity is valid with the respect to the verification strategy set
// for this deserializer. Then, it returns a verifier that checks signatures against the passed identity.
func (i *Deserializer) DeserializeVerifier(raw view.Identity) (driver.Verifier, error) {
	identity, err := i.Idemix.Deserialize(raw, true)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to deserialize identity")
	}

	return &NymSignatureVerifier{
		CSP:   i.CSP,
		IPK:   i.IssuerPublicKey,
		NymPK: identity.NymPublicKey,
	}, nil
}

// DeserializerVerifierWithNymEID expects raw to contain and Idemix MSP serialized identity.
// The deserializer first checks that the identity is valid with the respect to the verification strategy set
// for this deserializer and the passed NymEID. Then, it returns a verifier that checks signatures against the passed identity.
func (i *Deserializer) DeserializerVerifierWithNymEID(raw view.Identity, nymEID []byte) (driver.Verifier, error) {
	identity, err := i.Idemix.DeserializeAgainstNymEID(raw, true, nymEID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to deserialize identity against nym eid")
	}

	return &NymSignatureVerifier{
		CSP:   i.CSP,
		IPK:   i.IssuerPublicKey,
		NymPK: identity.NymPublicKey,
	}, nil
}

func (i *Deserializer) DeserializeSigner(raw []byte) (driver.Signer, error) {
	return nil, errors.New("not supported")
}

func (i *Deserializer) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	return i.Idemix.DeserializeAuditInfo(raw)
}

func (i *Deserializer) Info(raw []byte, auditInfo []byte) (string, error) {
	r, err := i.Idemix.Deserialize(raw, true)
	if err != nil {
		return "", err
	}

	eid := ""
	if len(auditInfo) != 0 {
		ai := &AuditInfo{}
		if err := ai.FromBytes(auditInfo); err != nil {
			return "", err
		}
		if err := ai.Match(raw); err != nil {
			return "", err
		}
		eid = ai.EnrollmentID()
	}

	return fmt.Sprintf("MSP.Idemix: [%s][%s][%s][%s][%s]", eid, view.Identity(raw).UniqueID(), r.SerializedIdentity.Mspid, r.OU.OrganizationalUnitIdentifier, r.Role.Role.String()), nil
}

func (i *Deserializer) String() string {
	return fmt.Sprintf("Idemix with IPK [%s]", hash.Hashable(i.IPK).String())
}
