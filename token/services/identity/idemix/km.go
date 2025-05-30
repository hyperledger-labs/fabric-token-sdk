/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"fmt"

	"github.com/IBM/idemix"
	bccsp "github.com/IBM/idemix/bccsp/types"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	crypto2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto/protos-go/config"
	"github.com/pkg/errors"
)

const (
	Any          bccsp.SignatureType = 100
	IdentityType identity.Type       = "idemix"
)

type (
	SKI = []byte
)

type SignerService interface {
	RegisterSigner(identity driver.Identity, signer driver.Signer, verifier driver.Verifier, info []byte) error
}

type KeyManager struct {
	*crypto2.Deserializer
	userKeySKI    SKI
	conf          *config.IdemixConfig
	SignerService SignerService

	sigType bccsp.SignatureType
	verType bccsp.VerificationType
}

func NewKeyManager(conf *crypto2.Config, signerService SignerService, sigType bccsp.SignatureType, csp bccsp.BCCSP) (*KeyManager, error) {
	if conf == nil {
		return nil, errors.New("no idemix config provided")
	}
	if conf.Version != crypto2.ProtobufProtocolVersionV1 {
		return nil, errors.Errorf("unsupported protocol version [%d]", conf.Version)
	}

	logger.Debugf("setting up Idemix key manager instance %s", conf.Name)

	// Import Issuer Public Key
	issuerPublicKey, err := csp.KeyImport(
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
	RevocationKey, err := csp.KeyGen(
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
	var userKeySKI SKI

	// use the ski
	if len(conf.Signer.Ski) != 0 {
		// load with ski
		userKey, err = csp.GetKey(conf.Signer.Ski)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to retrieve user key with ski [%s]", conf.Signer.Ski)
		}
	} else {
		if len(conf.Signer.Sk) != 0 && len(conf.Signer.Cred) != 0 {
			// A credential is present in the config, so we set up a default signer
			logger.Debugf("the signer contains key material, load it")

			// Import User secret key
			userKey, err = csp.KeyImport(conf.Signer.Sk, &bccsp.IdemixUserSecretKeyImportOpts{})
			if err != nil {
				return nil, errors.WithMessage(err, "failed importing signer secret key")
			}
		}
	}

	if userKey != nil {
		userKeySKI = userKey.SKI()
		conf.Signer.Ski = userKeySKI
		// Verify credential
		valid, err := csp.Verify(
			userKey,
			conf.Signer.Cred,
			nil,
			&bccsp.IdemixCredentialSignerOpts{
				IssuerPK: issuerPublicKey,
				Attributes: []bccsp.IdemixAttribute{
					{Type: bccsp.IdemixHiddenAttribute},
					{Type: bccsp.IdemixHiddenAttribute},
					{Type: bccsp.IdemixBytesAttribute, Value: []byte(conf.Signer.EnrollmentId)},
					{Type: bccsp.IdemixHiddenAttribute, Value: []byte(conf.Signer.RevocationHandle)},
				},
			},
		)
		if err != nil || !valid {
			return nil, errors.WithMessage(err, "credential is not cryptographically valid")
		}
		logger.Debugf("the signer contains key material, load it, done.")
	} else {
		logger.Debugf("the signer does not contain full key material, it will be considered remote [cred=%d,sk=%d]", len(conf.Signer.Cred), len(conf.Signer.Sk))
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
		return nil, errors.Errorf("unsupported signature type %d", sigType)
	}
	if verType == bccsp.BestEffort {
		sigType = bccsp.Standard
	}

	return &KeyManager{
		Deserializer: &crypto2.Deserializer{
			Name:            conf.Name,
			Csp:             csp,
			Ipk:             conf.Ipk,
			IssuerPublicKey: issuerPublicKey,
			RevocationPK:    RevocationPublicKey,
			Epoch:           0,
			VerType:         verType,
		},
		userKeySKI:    userKeySKI,
		conf:          conf,
		SignerService: signerService,
		sigType:       sigType,
		verType:       verType,
	}, nil
}

func (p *KeyManager) Identity(auditInfo []byte) (driver.Identity, []byte, error) {
	// Load the user key
	userKey, err := p.Csp.GetKey(p.userKeySKI)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to retrieve user key with ski [%s]", p.userKeySKI)
	}

	// Derive nymPublicKey
	nymKey, err := p.Csp.KeyDeriv(
		userKey,
		&bccsp.IdemixNymKeyDerivationOpts{
			Temporary: false,
			IssuerPK:  p.IssuerPublicKey,
		},
	)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "failed deriving nym")
	}
	nymPublicKey, err := nymKey.PublicKey()
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
		RhIndex:  crypto2.RHIndex,
		EidIndex: crypto2.EIDIndex,
		CRI:      p.conf.Signer.CredentialRevocationInformation,
		SigType:  sigType,
		Metadata: signerMetadata,
	}
	proof, err := p.Csp.Sign(
		userKey,
		nil,
		sigOpts,
	)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "failed to setup cryptographic proof of identity")
	}

	// Set up default signer
	id, err := crypto2.NewIdentity(p.Deserializer, nymPublicKey, proof, p.verType)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "failed to create identity")
	}
	sID := &crypto2.SigningIdentity{
		CSP:          p.Csp,
		Identity:     id,
		NymKeySKI:    nymPublicKey.SKI(),
		UserKeySKI:   p.userKeySKI,
		EnrollmentId: enrollmentID,
	}
	raw, err := sID.Serialize()
	if err != nil {
		return nil, nil, errors.WithMessage(err, "failed to serialize identity")
	}

	if p.SignerService != nil {
		if err := p.SignerService.RegisterSigner(raw, sID, sID, nil); err != nil {
			return nil, nil, errors.WithMessage(err, "failed to register signer")
		}
	}

	var infoRaw []byte
	switch sigType {
	case bccsp.Standard:
		infoRaw = nil
	case bccsp.EidNymRhNym:
		auditInfo := &crypto2.AuditInfo{
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
		logger.Debugf("new idemix identity generated with [%s:%s]", enrollmentID, hash.Hashable(rh))
		infoRaw, err = auditInfo.Bytes()
		if err != nil {
			return nil, nil, errors.WithMessage(err, "failed to serialize auditInfo")
		}
	default:
		return nil, nil, errors.Errorf("unsupported signature type [%d]", sigType)
	}
	return raw, infoRaw, nil
}

func (p *KeyManager) IsRemote() bool {
	return len(p.userKeySKI) == 0
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
		ai := &crypto2.AuditInfo{
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

	return fmt.Sprintf("Idemix: [%s][%s]", eid, driver.Identity(raw).UniqueID()), nil
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
	id, err := p.Deserialize(raw, true)
	if err != nil {
		return nil, err
	}

	si := &crypto2.SigningIdentity{
		CSP:          p.Csp,
		Identity:     id.Identity,
		UserKeySKI:   p.userKeySKI,
		NymKeySKI:    id.NymPublicKey.SKI(),
		EnrollmentId: p.conf.Signer.EnrollmentId,
	}

	// the only way to verify if this signing identity correspond to this key manager
	// is to generate a signature and verify it.
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
