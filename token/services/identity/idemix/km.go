/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"context"
	"fmt"

	bccsp "github.com/IBM/idemix/bccsp/types"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	crypto2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto/protos-go/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/schema"
)

const (
	Any          bccsp.SignatureType = 100
	IdentityType identity.Type       = "idemix"
)

type (
	SKI    = []byte
	Schema = string
)

// SchemaManager handles the various credential schemas. A credential schema
// contains information about the number of attributes, which attributes
// must be disclosed when creating proofs, the format of the attributes etc.
type SchemaManager interface {
	// PublicKeyImportOpts returns the options that `schema` uses to import its public keys
	PublicKeyImportOpts(schema string) (*bccsp.IdemixIssuerPublicKeyImportOpts, error)
	// SignerOpts returns the options for the passed arguments
	SignerOpts(schema string) (*bccsp.IdemixSignerOpts, error)
	// NymSignerOpts returns the options that `schema` uses to verify a nym signature
	NymSignerOpts(schema string) (*bccsp.IdemixNymSignerOpts, error)
	// EidNymAuditOpts returns the options that `sid` must use to audit an EIDNym
	EidNymAuditOpts(schema string, attrs [][]byte) (*bccsp.EidNymAuditOpts, error)
	// RhNymAuditOpts returns the options that `sid` must use to audit an RhNym
	RhNymAuditOpts(schema string, attrs [][]byte) (*bccsp.RhNymAuditOpts, error)
}

type SignerService interface {
	RegisterSigner(ctx context.Context, identity driver.Identity, signer driver.Signer, verifier driver.Verifier, info []byte) error
}

type KeyManager struct {
	*crypto2.Deserializer
	userKeySKI    SKI
	conf          *config.IdemixConfig
	SignerService SignerService

	sigType bccsp.SignatureType
	verType bccsp.VerificationType

	SchemaManager SchemaManager
	Schema        string
}

func NewKeyManager(
	conf *crypto2.Config,
	signerService SignerService,
	sigType bccsp.SignatureType,
	csp bccsp.BCCSP,
) (*KeyManager, error) {
	return NewKeyManagerWithSchema(
		conf,
		signerService,
		sigType,
		csp,
		schema.NewDefaultManager(),
		schema.DefaultSchema,
	)
}

func NewKeyManagerWithSchema(
	conf *crypto2.Config,
	signerService SignerService,
	sigType bccsp.SignatureType,
	csp bccsp.BCCSP,
	sm SchemaManager,
	schemaName string,
) (*KeyManager, error) {
	if conf == nil {
		return nil, errors.New("no idemix config provided")
	}
	if conf.Version != crypto2.ProtobufProtocolVersionV1 {
		return nil, errors.Errorf("unsupported protocol version [%d]", conf.Version)
	}

	logger.Debugf("setting up Idemix key manager instance %s", conf.Name)

	// get the opts from the schema manager
	opts, err := sm.PublicKeyImportOpts(schemaName)
	if err != nil {
		return nil, errors.Wrapf(err, "could not obtain PublicKeyImportOpts for schema '%s'", schemaName)
	}
	// Import Issuer Public Key
	issuerPublicKey, err := csp.KeyImport(
		conf.Ipk,
		opts,
	)
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
		return nil, errors.WithMessagef(err, "failed to generate revocation key")
	}
	RevocationPublicKey, err := RevocationKey.PublicKey()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to extract revocation public key")
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
				return nil, errors.WithMessagef(err, "failed importing signer secret key")
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
			return nil, errors.WithMessagef(err, "credential is not cryptographically valid")
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
			SchemaManager:   sm,
			Schema:          schemaName,
		},
		userKeySKI:    userKeySKI,
		conf:          conf,
		SignerService: signerService,
		sigType:       sigType,
		verType:       verType,
		SchemaManager: sm,
		Schema:        schemaName,
	}, nil
}

func (p *KeyManager) Identity(ctx context.Context, auditInfo []byte) (driver.Identity, []byte, error) {
	logger.DebugfContext(ctx, "get user secret key")
	// Load the user key
	userKey, err := p.Csp.GetKey(p.userKeySKI)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to retrieve user key with ski [%s]", p.userKeySKI)
	}

	// Derive nymPublicKey
	logger.DebugfContext(ctx, "derive nym")
	nymKey, err := p.Csp.KeyDeriv(
		userKey,
		&bccsp.IdemixNymKeyDerivationOpts{
			Temporary: false,
			IssuerPK:  p.IssuerPublicKey,
		},
	)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed deriving nym")
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
		logger.DebugfContext(ctx, "deserialize passed audit info")
		ai, err := p.DeserializeAuditInfo(ctx, auditInfo)
		if err != nil {
			return nil, nil, err
		}
		signerMetadata = &bccsp.IdemixSignerMetadata{
			EidNymAuditData: ai.EidNymAuditData,
			RhNymAuditData:  ai.RhNymAuditData,
		}
	}

	// Create the cryptographic evidence that this identity is valid
	logger.DebugfContext(ctx, "create crypto evidence for the identity")
	sigOpts, err := p.SchemaManager.SignerOpts(p.Schema)
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
		userKey,
		nil,
		sigOpts,
	)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to setup cryptographic proof of identity")
	}

	// Set up default signer
	logger.DebugfContext(ctx, "setup default signer")
	id, err := crypto2.NewIdentity(p.Deserializer, nymPublicKey, proof, p.verType, p.SchemaManager, p.Schema)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to create identity")
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
		return nil, nil, errors.WithMessagef(err, "failed to serialize identity")
	}

	if p.SignerService != nil {
		logger.DebugfContext(ctx, "register signer for identity")
		if err := p.SignerService.RegisterSigner(ctx, raw, sID, sID, nil); err != nil {
			return nil, nil, errors.WithMessagef(err, "failed to register signer")
		}
	}

	logger.DebugfContext(ctx, "prepare audit info")
	var infoRaw []byte
	switch sigType {
	case bccsp.Standard:
		infoRaw = nil
	case bccsp.EidNymRhNym:
		auditInfo := &crypto2.AuditInfo{
			EidNymAuditData: sigOpts.Metadata.EidNymAuditData,
			RhNymAuditData:  sigOpts.Metadata.RhNymAuditData,
			Attributes: [][]byte{
				nil,
				nil,
				[]byte(enrollmentID),
				[]byte(rh),
			},
			Csp:             p.Csp,
			IssuerPublicKey: p.IssuerPublicKey,
			SchemaManager:   p.SchemaManager,
			Schema:          p.Schema,
		}
		logger.DebugfContext(ctx, "new idemix identity generated with [%s:%s]", enrollmentID, hash.Hashable(rh))
		infoRaw, err = auditInfo.Bytes()
		if err != nil {
			return nil, nil, errors.WithMessagef(err, "failed to serialize auditInfo")
		}
	default:
		return nil, nil, errors.Errorf("unsupported signature type [%d]", sigType)
	}
	logger.DebugfContext(ctx, "prepare audit info done")

	return raw, infoRaw, nil
}

func (p *KeyManager) IsRemote() bool {
	return len(p.userKeySKI) == 0
}

func (p *KeyManager) DeserializeVerifier(ctx context.Context, raw []byte) (driver.Verifier, error) {
	r, err := p.Deserialize(ctx, raw)
	if err != nil {
		return nil, err
	}

	return r.Identity, nil
}

func (p *KeyManager) DeserializeSigner(ctx context.Context, raw []byte) (driver.Signer, error) {
	return p.DeserializeSigningIdentity(ctx, raw)
}

func (p *KeyManager) Info(ctx context.Context, raw []byte, auditInfo []byte) (string, error) {
	eid := ""
	if len(auditInfo) != 0 {
		ai := &crypto2.AuditInfo{
			Csp:             p.Csp,
			IssuerPublicKey: p.IssuerPublicKey,
			SchemaManager:   p.SchemaManager,
			Schema:          p.Schema,
		}
		if err := ai.FromBytes(auditInfo); err != nil {
			return "", err
		}
		if err := ai.Match(ctx, raw); err != nil {
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

func (p *KeyManager) DeserializeSigningIdentity(ctx context.Context, raw []byte) (driver.SigningIdentity, error) {
	id, err := p.Deserialize(ctx, raw)
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
