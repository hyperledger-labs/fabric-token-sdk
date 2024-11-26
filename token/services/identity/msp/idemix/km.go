/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"fmt"
	"strconv"

	bccsp "github.com/IBM/idemix/bccsp/types"
	im "github.com/IBM/idemix/idemixmsp"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix/msp"
	m "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

const (
	Any bccsp.SignatureType = 100
)

// SchemaManager handles the various credential schemas. A credential schema
// contains information about the number of attributes, which attributes
// must be disclosed when creating proofs, the format of the attributes etc.
type SchemaManager interface {
	// PublicKeyImportOpts returns the options that `schema` uses to import its public keys
	PublicKeyImportOpts(schema string) (*bccsp.IdemixIssuerPublicKeyImportOpts, error)
	// SignerOpts returns the options for the passed arguments
	SignerOpts(schema string, ou *m.OrganizationUnit, role *m.MSPRole) (*bccsp.IdemixSignerOpts, error)
	// NymSignerOpts returns the options that `schema` uses to verify a nym signature
	NymSignerOpts(schema string) (*bccsp.IdemixNymSignerOpts, error)
	// EidNymAuditOpts returns the options that `sid` must use to audit an EIDNym
	EidNymAuditOpts(schema string, attrs [][]byte) (*bccsp.EidNymAuditOpts, error)
	// RhNymAuditOpts returns the options that `sid` must use to audit an RhNym
	RhNymAuditOpts(schema string, attrs [][]byte) (*bccsp.RhNymAuditOpts, error)
}

type SignerService interface {
	RegisterSigner(identity driver.Identity, signer driver.Signer, verifier driver.Verifier, info []byte) error
}

type KeyManager struct {
	*msp.Deserializer
	userKey       bccsp.Key
	conf          im.IdemixMSPConfig
	SignerService SignerService

	sigType bccsp.SignatureType
	verType bccsp.VerificationType

	SchemaManager SchemaManager
	Schema        string
}

func NewKeyManager(
	conf1 *m.MSPConfig,
	signerService SignerService,
	sigType bccsp.SignatureType,
	cryptoProvider bccsp.BCCSP,
	sm SchemaManager,
	schema string,
) (*KeyManager, error) {
	logger.Debugf("Setting up Idemix-based MSP instance")

	if conf1 == nil {
		return nil, errors.Errorf("setup error: nil conf reference")
	}

	var conf im.IdemixMSPConfig
	err := proto.Unmarshal(conf1.Config, &conf)
	if err != nil {
		return nil, errors.Wrap(err, "failed unmarshalling idemix provider config")
	}

	logger.Debugf("Setting up Idemix MSP instance %s", conf.Name)

	// get the opts from the schema manager
	opts, err := sm.PublicKeyImportOpts(schema)
	if err != nil {
		return nil, errors.Wrapf(err, "could not obtain PublicKeyImportOpts for schema '%s'", schema)
	}
	// Import Issuer Public Key
	issuerPublicKey, err := cryptoProvider.KeyImport(
		conf.Ipk,
		opts,
	)
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
		role := &m.MSPRole{
			MspIdentifier: conf.Name,
			Role:          m.MSPRole_MEMBER,
		}
		if msp.CheckRole(int(conf.Signer.Role), msp.ADMIN) {
			role.Role = m.MSPRole_ADMIN
		}
		valid, err := cryptoProvider.Verify(
			userKey,
			conf.Signer.Cred,
			nil,
			&bccsp.IdemixCredentialSignerOpts{
				IssuerPK: issuerPublicKey,
				Attributes: []bccsp.IdemixAttribute{
					{Type: bccsp.IdemixBytesAttribute, Value: []byte(conf.Signer.OrganizationalUnitIdentifier)},
					{Type: bccsp.IdemixIntAttribute, Value: msp.GetIdemixRoleFromMSPRole(role)},
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
			SchemaManager:   sm,
			Schema:          schema,
		},
		userKey:       userKey,
		conf:          conf,
		SignerService: signerService,
		sigType:       sigType,
		verType:       verType,
		SchemaManager: sm,
		Schema:        schema,
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

	role := &m.MSPRole{
		MspIdentifier: p.Name,
		Role:          m.MSPRole_MEMBER,
	}
	if msp.CheckRole(int(p.conf.Signer.Role), msp.ADMIN) {
		role.Role = m.MSPRole_ADMIN
	}

	ou := &m.OrganizationUnit{
		MspIdentifier:                p.Name,
		OrganizationalUnitIdentifier: p.conf.Signer.OrganizationalUnitIdentifier,
		CertifiersIdentifier:         p.IssuerPublicKey.SKI(),
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
	sigOpts, err := p.SchemaManager.SignerOpts(p.Schema, ou, role)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could obtain signer sigOpts for schema %s", p.Schema)
	}
	sigOpts.Credential = p.conf.Signer.Cred
	sigOpts.Nym = nymKey
	sigOpts.IssuerPK = p.IssuerPublicKey
	sigOpts.CRI = p.conf.Signer.CredentialRevocationInformation
	sigOpts.SigType = sigType
	sigOpts.Metadata = signerMetadata
	proof, err := p.Csp.Sign(
		p.userKey,
		nil,
		sigOpts,
	)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "Failed to setup cryptographic proof of identity")
	}

	// Set up default signer
	id, err := msp.NewIdentity(p.Deserializer, NymPublicKey, role, ou, proof, p.verType, p.SchemaManager, p.Schema)
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
			EidNymAuditData: sigOpts.Metadata.EidNymAuditData,
			RhNymAuditData:  sigOpts.Metadata.RhNymAuditData,
			Attributes: [][]byte{
				[]byte(p.conf.Signer.OrganizationalUnitIdentifier),
				[]byte(strconv.Itoa(msp.GetIdemixRoleFromMSPRole(role))),
				[]byte(enrollmentID),
				[]byte(rh),
			},
			Csp:             p.Csp,
			IssuerPublicKey: p.IssuerPublicKey,
			SchemaManager:   p.SchemaManager,
			Schema:          p.Schema,
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
	r, err := p.Deserialize(raw)
	if err != nil {
		return nil, err
	}

	return r.Identity, nil
}

func (p *KeyManager) DeserializeSigner(raw []byte) (driver.Signer, error) {
	r, err := p.Deserialize(raw)
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

func (p *KeyManager) Info(raw []byte, auditInfo []byte) (string, error) {
	r, err := p.Deserialize(raw)
	if err != nil {
		return "", err
	}

	eid := ""
	if len(auditInfo) != 0 {
		ai := &msp.AuditInfo{
			Csp:             p.Csp,
			IssuerPublicKey: p.IssuerPublicKey,
			SchemaManager:   p.SchemaManager,
			Schema:          p.Schema,
		}
		if err := ai.FromBytes(auditInfo); err != nil {
			return "", err
		}
		if err := ai.Match(raw); err != nil {
			return "", err
		}
		eid = ai.EnrollmentID()
	}

	return fmt.Sprintf("MSP.Idemix: [%s][%s][%s][%s][%s]", eid, driver.Identity(raw).UniqueID(), r.SerializedIdentity.Mspid, r.OU.OrganizationalUnitIdentifier, r.Role.Role.String()), nil
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
	si := &m.SerializedIdentity{}
	err := proto.Unmarshal(raw, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to msp.SerializedIdentity{}")
	}

	serialized := new(im.SerializedIdemixIdentity)
	err = proto.Unmarshal(si.IdBytes, serialized)
	if err != nil {
		return nil, errors.Wrap(err, "could not deserialize a SerializedIdemixIdentity")
	}
	if serialized.NymX == nil || serialized.NymY == nil {
		return nil, errors.Errorf("unable to deserialize idemix identity: pseudonym is invalid")
	}

	// Import NymPublicKey
	var rawNymPublicKey []byte
	rawNymPublicKey = append(rawNymPublicKey, serialized.NymX...)
	rawNymPublicKey = append(rawNymPublicKey, serialized.NymY...)
	NymPublicKey, err := p.Csp.KeyImport(
		rawNymPublicKey,
		&bccsp.IdemixNymPublicKeyImportOpts{Temporary: true},
	)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to import nym public key")
	}

	// OU
	ou := &m.OrganizationUnit{}
	err = proto.Unmarshal(serialized.Ou, ou)
	if err != nil {
		return nil, errors.Wrap(err, "cannot deserialize the OU of the identity")
	}

	// RoleAttribute
	role := &m.MSPRole{}
	err = proto.Unmarshal(serialized.Role, role)
	if err != nil {
		return nil, errors.Wrap(err, "cannot deserialize the role of the identity")
	}

	id, _ := msp.NewIdentity(p.Deserializer, NymPublicKey, role, ou, serialized.Proof, p.verType, p.SchemaManager, p.Schema)
	if err := id.Validate(); err != nil {
		return nil, errors.Wrap(err, "cannot deserialize, invalid identity")
	}
	nymKey, err := p.Csp.GetKey(NymPublicKey.SKI())
	if err != nil {
		return nil, errors.Wrap(err, "cannot find nym secret key")
	}

	return &msp.SigningIdentity{
		Identity:     id,
		Cred:         p.conf.Signer.Cred,
		UserKey:      p.userKey,
		NymKey:       nymKey,
		EnrollmentId: p.conf.Signer.EnrollmentId,
	}, nil
}
