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

type deserializer struct {
	*common
}

func newDeserializer(ipk []byte, verType bccsp.VerificationType, nymEID []byte) (*deserializer, error) {
	logger.Debugf("Setting up Idemix-based MSP instance")

	curve := math.Curves[math.FP256BN_AMCL]
	cryptoProvider, err := idemix.New(&keystore.Dummy{}, curve, &amcl.Fp256bn{C: curve}, true)
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

	return &deserializer{
		common: &common{
			Ipk:             ipk,
			Csp:             cryptoProvider,
			IssuerPublicKey: issuerPublicKey,
			VerType:         verType,
			NymEID:          nymEID,
		},
	}, nil
}

// NewDeserializer returns a new deserializer for the best effort strategy
func NewDeserializer(ipk []byte) (*deserializer, error) {
	return newDeserializer(ipk, bccsp.BestEffort, nil)
}

func NewDeserializerForNymEID(ipk []byte, nymEID []byte) (*deserializer, error) {
	return newDeserializer(ipk, bccsp.BestEffort, nymEID)
}

func (i *deserializer) DeserializeVerifier(raw view.Identity) (driver.Verifier, error) {
	r, err := i.Deserialize(raw, false)
	if err != nil {
		return nil, err
	}

	return &verifier{
		idd:          i,
		nymPublicKey: r.NymPublicKey,
	}, nil
}

func (i *deserializer) DeserializeSigner(raw []byte) (driver.Signer, error) {
	return nil, errors.New("not supported")
}

func (i *deserializer) DeserializeAuditInfo(raw []byte) (driver.Matcher, error) {
	return i.common.DeserializeAuditInfo(raw)
}

func (i *deserializer) Info(raw []byte, auditInfo []byte) (string, error) {
	r, err := i.Deserialize(raw, false)
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

func (i *deserializer) String() string {
	return fmt.Sprintf("Idemix with IPK [%s]", hash.Hashable(i.Ipk).String())
}

type verifier struct {
	idd          *deserializer
	nymPublicKey bccsp.Key
}

func (v *verifier) Verify(message, sigma []byte) error {
	_, err := v.idd.Csp.Verify(
		v.nymPublicKey,
		sigma,
		message,
		&csp.IdemixNymSignerOpts{
			IssuerPK: v.idd.IssuerPublicKey,
		},
	)
	return err
}
