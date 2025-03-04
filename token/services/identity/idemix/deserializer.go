/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"fmt"

	idemix2 "github.com/IBM/idemix"
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
	logger.Debugf("new deserialized for dlog idemix")
	cryptoProvider, err := crypto2.NewBCCSPWithDummyKeyStore(curveID, curveID == math.BLS12_381_BBS)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to instantiate crypto provider for curve [%d]", curveID)
	}
	return NewDeserializerWithProvider(ipk, csp.ExpectEidNymRhNym, nil, cryptoProvider)
}

// NewDeserializerWithProvider returns a new serialized for the passed arguments
func NewDeserializerWithProvider(
	ipk []byte,
	verType csp.VerificationType,
	nymEID []byte,
	cryptoProvider csp.BCCSP,
) (*Deserializer, error) {
	return NewDeserializerWithBCCSP(ipk, verType, nymEID, cryptoProvider)
}

func NewDeserializerWithBCCSP(ipk []byte, verType csp.VerificationType, nymEID []byte, cryptoProvider csp.BCCSP) (*Deserializer, error) {
	logger.Debugf("Setting up Idemix-based deserailizer instance")

	// Import Issuer Public Key
	var issuerPublicKey csp.Key
	var err error
	if len(ipk) != 0 {
		issuerPublicKey, err = cryptoProvider.KeyImport(
			ipk,
			&csp.IdemixIssuerPublicKeyImportOpts{
				Temporary: true,
				AttributeNames: []string{
					idemix2.AttributeNameOU,
					idemix2.AttributeNameRole,
					idemix2.AttributeNameEnrollmentId,
					idemix2.AttributeNameRevocationHandle,
				},
			})
		if err != nil {
			return nil, err
		}
	}

	return &Deserializer{
		Deserializer: &crypto2.Deserializer{
			Ipk:             ipk,
			Csp:             cryptoProvider,
			IssuerPublicKey: issuerPublicKey,
			VerType:         verType,
			NymEID:          nymEID,
		},
	}, nil
}

func (i *Deserializer) DeserializeVerifier(raw driver.Identity) (driver.Verifier, error) {
	identity, err := i.Deserialize(raw, true)
	if err != nil {
		return nil, err
	}

	return &crypto2.NymSignatureVerifier{
		CSP:   i.Deserializer.Csp,
		IPK:   i.Deserializer.IssuerPublicKey,
		NymPK: identity.NymPublicKey,
	}, nil
}

func (i *Deserializer) DeserializeVerifierAgainstNymEID(raw []byte, nymEID []byte) (driver.Verifier, error) {
	identity, err := i.Deserializer.DeserializeAgainstNymEID(raw, true, nymEID)
	if err != nil {
		return nil, err
	}

	return &crypto2.NymSignatureVerifier{
		CSP:   i.Deserializer.Csp,
		IPK:   i.Deserializer.IssuerPublicKey,
		NymPK: identity.NymPublicKey,
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

func (i *Deserializer) GetAuditInfo(raw []byte, p driver.AuditInfoProvider) ([][]byte, error) {
	auditInfo, err := p.GetAuditInfo(raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", driver.Identity(raw).String())
	}
	return [][]byte{auditInfo}, nil
}

func (i *Deserializer) Info(raw []byte, auditInfo []byte) (string, error) {
	eid := ""
	if len(auditInfo) != 0 {
		ai, err := crypto2.DeserializeAuditInfo(auditInfo)
		if err != nil {
			return "", err
		}
		if err := ai.Match(raw); err != nil {
			return "", err
		}
		eid = ai.EnrollmentID()
	}

	return fmt.Sprintf("Idemix: [%s][%s]", eid, driver.Identity(raw).UniqueID()), nil
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
