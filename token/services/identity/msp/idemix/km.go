/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"fmt"

	"github.com/IBM/idemix"
	bccsp "github.com/IBM/idemix/bccsp/types"
	"github.com/IBM/idemix/idemixmsp"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

const (
	Any          bccsp.SignatureType = 100
	IdentityType identity.Type       = "idemix"
)

type SignerService interface {
	RegisterSigner(identity driver.Identity, signer driver.Signer, verifier driver.Verifier, info []byte) error
}

type KeyManager struct {
	*msp.Deserializer
	userKey       bccsp.Key
	conf          idemixmsp.IdemixMSPConfig
	SignerService SignerService

	sigType bccsp.SignatureType
	verType bccsp.VerificationType
}

func NewKeyManager(conf1 *msp.Config, signerService SignerService, sigType bccsp.SignatureType, cryptoProvider bccsp.BCCSP) (*KeyManager, error) {
	logger.Debugf("Setting up Idemix-based MSP instance")

	if conf1 == nil {
		return nil, errors.Errorf("setup error: nil conf reference")
	}

	var conf idemixmsp.IdemixMSPConfig
	err := proto.Unmarshal(conf1.Config, &conf)
	if err != nil {
		return nil, errors.Wrap(err, "failed unmarshalling idemix provider config")
	}

	logger.Debugf("Setting up Idemix MSP instance %s", conf.Name)

	// Import Issuer Public Key
	issuerPublicKey, err := cryptoProvider.KeyImport(
		conf.Ipk,
		&bccsp.IdemixIssuerPublicKeyImportOpts{
			Temporary: true,
			AttributeNames: []string{
				idemix.AttributeNameOU,
				idemix.AttributeNameRole,
				idemix.AttributeNameEnrollmentId,
				idemix.AttributeNameRevocationHandle,
			},
		})
	if err != nil {
		return nil, err
	}

	// IMPORTANT: we generate an ephemeral revocation key public key because
	// it is never used in the current idemix implementations.
	// This might change in the future
	RevocationKey, err := cryptoProvider.KeyGen(
		&bccsp.IdemixRevocationKeyGenOpts{Temporary: true},
	)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to generate revocation key")
	}
	RevocationPublicKey, err := RevocationKey.PublicKey()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to extract revocation public key")
	}

	if conf.Signer == nil {
		// No credential in config, so we don't set up a default signer
		return nil, errors.Errorf("no signer information found")
	}

	var userKey bccsp.Key
	if len(conf.Signer.Sk) != 0 && len(conf.Signer.Cred) != 0 {
		// A credential is present in the config, so we set up a default signer
		logger.Debugf("the signer contains key material, load it")

		// Import User secret key
		userKey, err = cryptoProvider.KeyImport(conf.Signer.Sk, &bccsp.IdemixUserSecretKeyImportOpts{Temporary: true})
		if err != nil {
			return nil, errors.WithMessage(err, "failed importing signer secret key")
		}

		// Verify credential
		valid, err := cryptoProvider.Verify(
			userKey,
			conf.Signer.Cred,
			nil,
			&bccsp.IdemixCredentialSignerOpts{
				IssuerPK: issuerPublicKey,
				Attributes: []bccsp.IdemixAttribute{
					{Type: bccsp.IdemixHiddenAttribute},
					{Type: bccsp.IdemixHiddenAttribute},
					{Type: bccsp.IdemixBytesAttribute, Value: []byte(conf.Signer.EnrollmentId)},
					{Type: bccsp.IdemixHiddenAttribute},
				},
			},
		)
		if err != nil || !valid {
			return nil, errors.WithMessage(err, "credential is not cryptographically valid")
		}
		logger.Debugf("the signer contains key material, load it, done.")
	} else {
		logger.Debugf("the signer does not contain full key material [cred=%d,sk=%d]", len(conf.Signer.Cred), len(conf.Signer.Sk))
	}

	var verType bccsp.VerificationType
	switch sigType {
	case bccsp.Standard:
		verType = bccsp.ExpectStandard
	case bccsp.EidNymRhNym:
		verType = bccsp.ExpectEidNymRhNym
	case Any:
		verType = bccsp.BestEffort
	default:
		panic("invalid sig type")
	}
	if verType == bccsp.BestEffort {
		sigType = bccsp.Standard
	}

	return &KeyManager{
		Deserializer: &msp.Deserializer{
			Name:            conf.Name,
			Csp:             cryptoProvider,
			IssuerPublicKey: issuerPublicKey,
			RevocationPK:    RevocationPublicKey,
			Epoch:           0,
			VerType:         verType,
		},
		userKey:       userKey,
		conf:          conf,
		SignerService: signerService,
		sigType:       sigType,
		verType:       verType,
	}, nil
}

func (p *KeyManager) Identity(auditInfo []byte) (driver.Identity, []byte, error) {
	// Derive NymPublicKey
	nymKey, err := p.Csp.KeyDeriv(
		p.userKey,
		&bccsp.IdemixNymKeyDerivationOpts{
			Temporary: false,
			IssuerPK:  p.IssuerPublicKey,
		},
	)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "failed deriving nym")
	}
	NymPublicKey, err := nymKey.PublicKey()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed getting public nym key")
	}

	enrollmentID := p.conf.Signer.EnrollmentId
	rh := p.conf.Signer.RevocationHandle
	sigType := p.sigType
	var signerMetadata *bccsp.IdemixSignerMetadata
	if len(auditInfo) != 0 {
		ai, err := p.DeserializeAuditInfo(auditInfo)
		if err != nil {
			return nil, nil, err
		}

		signerMetadata = &bccsp.IdemixSignerMetadata{
			EidNymAuditData: ai.EidNymAuditData,
			RhNymAuditData:  ai.RhNymAuditData,
		}
	}

	// Create the cryptographic evidence that this identity is valid
	sigOpts := &bccsp.IdemixSignerOpts{
		Credential: p.conf.Signer.Cred,
		Nym:        nymKey,
		IssuerPK:   p.IssuerPublicKey,
		Attributes: []bccsp.IdemixAttribute{
			{Type: bccsp.IdemixHiddenAttribute},
			{Type: bccsp.IdemixHiddenAttribute},
			{Type: bccsp.IdemixHiddenAttribute},
			{Type: bccsp.IdemixHiddenAttribute},
		},
		RhIndex:  msp.RHIndex,
		EidIndex: msp.EIDIndex,
		CRI:      p.conf.Signer.CredentialRevocationInformation,
		SigType:  sigType,
		Metadata: signerMetadata,
	}
	proof, err := p.Csp.Sign(
		p.userKey,
		nil,
		sigOpts,
	)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "Failed to setup cryptographic proof of identity")
	}

	// Set up default signer
	id, err := msp.NewIdentity(p.Deserializer, NymPublicKey, proof, p.verType)
	if err != nil {
		return nil, nil, err
	}
	sID := &msp.SigningIdentity{
		Identity:     id,
		Cred:         p.conf.Signer.Cred,
		UserKey:      p.userKey,
		NymKey:       nymKey,
		EnrollmentId: enrollmentID,
	}
	raw, err := sID.Serialize()
	if err != nil {
		return nil, nil, err
	}

	if p.SignerService != nil {
		if err := p.SignerService.RegisterSigner(raw, sID, sID, nil); err != nil {
			return nil, nil, err
		}
	}

	var infoRaw []byte
	switch sigType {
	case bccsp.Standard:
		infoRaw = nil
	case bccsp.EidNymRhNym:
		auditInfo := &msp.AuditInfo{
			Csp:             p.Csp,
			IssuerPublicKey: p.IssuerPublicKey,
			EidNymAuditData: sigOpts.Metadata.EidNymAuditData,
			RhNymAuditData:  sigOpts.Metadata.RhNymAuditData,
			Attributes: [][]byte{
				nil,
				nil,
				[]byte(enrollmentID),
				[]byte(rh),
			},
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("new idemix identity generated with [%s:%s]", enrollmentID, hash.Hashable(auditInfo.Attributes[3]).String())
		}
		infoRaw, err = auditInfo.Bytes()
		if err != nil {
			return nil, nil, err
		}
	default:
		panic("invalid sig type")
	}
	return raw, infoRaw, nil
}

func (p *KeyManager) IsRemote() bool {
	return p.userKey == nil
}

func (p *KeyManager) DeserializeVerifier(raw []byte) (driver.Verifier, error) {
	r, err := p.Deserialize(raw, true)
	if err != nil {
		return nil, err
	}

	return r.Identity, nil
}

func (p *KeyManager) DeserializeSigner(raw []byte) (driver.Signer, error) {
	return p.DeserializeSigningIdentity(raw)
}

func (p *KeyManager) Info(raw []byte, auditInfo []byte) (string, error) {
	eid := ""
	if len(auditInfo) != 0 {
		ai := &msp.AuditInfo{
			Csp:             p.Csp,
			IssuerPublicKey: p.IssuerPublicKey,
		}
		if err := ai.FromBytes(auditInfo); err != nil {
			return "", err
		}
		if err := ai.Match(raw); err != nil {
			return "", err
		}
		eid = ai.EnrollmentID()
	}

	return fmt.Sprintf("MSP.Idemix: [%s][%s]", eid, driver.Identity(raw).UniqueID()), nil
}

func (p *KeyManager) String() string {
	return fmt.Sprintf("Idemix KeyManager [%s]", hash.Hashable(p.Ipk).String())
}

func (p *KeyManager) EnrollmentID() string {
	return p.conf.Signer.EnrollmentId
}

func (p *KeyManager) Anonymous() bool {
	return true
}

func (p *KeyManager) DeserializeSigningIdentity(raw []byte) (driver.SigningIdentity, error) {
	r, err := p.Deserialize(raw, true)
	if err != nil {
		return nil, err
	}

	nymKey, err := p.Csp.GetKey(r.NymPublicKey.SKI())
	if err != nil {
		return nil, errors.Wrap(err, "cannot find nym secret key")
	}

	si := &msp.SigningIdentity{
		Identity:     r.Identity,
		Cred:         p.conf.Signer.Cred,
		UserKey:      p.userKey,
		NymKey:       nymKey,
		EnrollmentId: p.conf.Signer.EnrollmentId,
	}
	msg := []byte("hello world!!!")
	sigma, err := si.Sign(msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed generating verification signature")
	}
	if err := si.Verify(msg, sigma); err != nil {
		return nil, errors.Wrap(err, "failed verifying verification signature")
	}
	return si, nil
}

func (p *KeyManager) IdentityType() identity.Type {
	return IdentityType
}
