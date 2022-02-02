/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"fmt"

	idemix "github.com/IBM/idemix/bccsp"
	"github.com/IBM/idemix/bccsp/keystore"
	bccsp "github.com/IBM/idemix/bccsp/schemes"
	csp "github.com/IBM/idemix/bccsp/schemes"
	idemix2 "github.com/IBM/idemix/bccsp/schemes/dlog/crypto"
	"github.com/IBM/idemix/bccsp/schemes/dlog/crypto/translator/amcl"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger/fabric/msp"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

var logger = flogging.MustGetLogger("token-sdk.idemix")

type Deserializer struct {
	*Common
}

// NewDeserializer returns a new deserializer for the best effort strategy
func NewDeserializer(ipk []byte, curveID math.CurveID) (*Deserializer, error) {
	return newDeserializer(ipk, bccsp.BestEffort, nil, curveID)
}

func newDeserializer(ipk []byte, verType bccsp.VerificationType, nymEID []byte, curveID math.CurveID) (*Deserializer, error) {
	logger.Debugf("Setting up Idemix-based MSP instance")

	curve := math.Curves[curveID]
	var tr idemix2.Translator
	switch curveID {
	case math.BN254:
		tr = &amcl.Gurvy{C: curve}
	case math.FP256BN_AMCL:
		tr = &amcl.Fp256bn{C: curve}
	case math.FP256BN_AMCL_MIRACL:
		tr = &amcl.Fp256bnMiracl{C: curve}
	default:
		return nil, errors.Errorf("unsupported curve ID: %d", curveID)
	}

	cryptoProvider, err := idemix.New(&keystore.Dummy{}, curve, tr, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting crypto provider")
	}

	// Import Issuer Public Key
	var issuerPublicKey csp.Key
	if len(ipk) != 0 {
		issuerPublicKey, err = cryptoProvider.KeyImport(
			ipk,
			&csp.IdemixIssuerPublicKeyImportOpts{
				Temporary: true,
				AttributeNames: []string{
					msp.AttributeNameOU,
					msp.AttributeNameRole,
					msp.AttributeNameEnrollmentId,
					msp.AttributeNameRevocationHandle,
				},
			})
		if err != nil {
			return nil, err
		}
	}

	return &Deserializer{
		Common: &Common{
			IPK:             ipk,
			CSP:             cryptoProvider,
			IssuerPublicKey: issuerPublicKey,
			VerType:         verType,
			NymEID:          nymEID,
		},
	}, nil
}

func (i *Deserializer) DeserializeVerifier(raw view.Identity) (driver.Verifier, error) {
	r, err := i.Common.Deserialize(raw, false)
	if err != nil {
		return nil, err
	}

	return &Verifier{
		CSP:   i.CSP,
		IPK:   i.IssuerPublicKey,
		NymPK: r.NymPublicKey,
	}, nil
}

func (i *Deserializer) DeserializerVerifierWithNymEID(raw view.Identity, nymEID []byte) (driver.Verifier, error) {
	r, err := i.Common.DeserializeWithNymEID(raw, false, nymEID)
	if err != nil {
		return nil, err
	}

	return &Verifier{
		CSP:   i.CSP,
		IPK:   i.IssuerPublicKey,
		NymPK: r.NymPublicKey,
	}, nil
}

func (i *Deserializer) DeserializeSigner(raw []byte) (driver.Signer, error) {
	return nil, errors.New("not supported")
}

func (i *Deserializer) DeserializeAuditInfo(raw []byte) (driver.Matcher, error) {
	return i.Common.DeserializeAuditInfo(raw)
}

func (i *Deserializer) Info(raw []byte, auditInfo []byte) (string, error) {
	r, err := i.Common.Deserialize(raw, false)
	if err != nil {
		return "", err
	}

	eid := ""
	if len(auditInfo) != 0 {
		ai := &AuditInfo{}
		if err := ai.FromBytes(auditInfo); err != nil {
			return "", err
		}
		if err := ai.Match(view.Identity(raw)); err != nil {
			return "", err
		}
		eid = ai.EnrollmentID()
	}

	return fmt.Sprintf("MSP.Idemix: [%s][%s][%s][%s][%s]", eid, view.Identity(raw).UniqueID(), r.si.Mspid, r.ou.OrganizationalUnitIdentifier, r.role.Role.String()), nil
}

func (i *Deserializer) String() string {
	return fmt.Sprintf("Idemix with IPK [%s]", hash.Hashable(i.IPK).String())
}

type Verifier struct {
	CSP   bccsp.BCCSP
	IPK   bccsp.Key
	NymPK bccsp.Key
}

func (v *Verifier) Verify(message, sigma []byte) error {
	_, err := v.CSP.Verify(
		v.NymPK,
		sigma,
		message,
		&csp.IdemixNymSignerOpts{
			IssuerPK: v.IPK,
		},
	)
	return err
}
