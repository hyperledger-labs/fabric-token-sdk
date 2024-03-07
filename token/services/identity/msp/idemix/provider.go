/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"fmt"
	"strconv"

	"github.com/IBM/idemix"
	bccsp "github.com/IBM/idemix/bccsp/types"
	"github.com/IBM/idemix/idemixmsp"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/common"
	m "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

const (
	EIDIndex = 2
	RHIndex  = 3
)

const (
	Any bccsp.SignatureType = 100
)

type SignerService interface {
	RegisterSigner(identity view.Identity, signer driver.Signer, verifier driver.Verifier) error
}

type Provider struct {
	*Idemix
	userKey       bccsp.Key
	conf          idemixmsp.IdemixMSPConfig
	SignerService SignerService

	sigType bccsp.SignatureType
	verType bccsp.VerificationType
}

func NewProvider(conf1 *m.MSPConfig, signerService SignerService, sigType bccsp.SignatureType, cryptoProvider bccsp.BCCSP) (*Provider, error) {
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

	// Import revocation public key
	RevocationPublicKey, err := cryptoProvider.KeyImport(
		conf.RevocationPk,
		&bccsp.IdemixRevocationPublicKeyImportOpts{Temporary: true},
	)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to import revocation public key")
	}

	if conf.Signer == nil {
		// No credential in config, so we don't setup a default signer
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
		if CheckRole(int(conf.Signer.Role), ADMIN) {
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
					{Type: bccsp.IdemixIntAttribute, Value: GetIdemixRoleFromMSPRole(role)},
					{Type: bccsp.IdemixBytesAttribute, Value: []byte(conf.Signer.EnrollmentId)},
					{Type: bccsp.IdemixHiddenAttribute},
				},
			},
		)
		if err != nil || !valid {
			return nil, errors.WithMessage(err, "credential is not cryptographically valid")
		}
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

	return &Provider{
		Idemix: &Idemix{
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

func (p *Provider) Identity(opts *common.IdentityOptions) (view.Identity, []byte, error) {
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
	if CheckRole(int(p.conf.Signer.Role), ADMIN) {
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
	if opts != nil {
		if opts.EIDExtension {
			sigType = bccsp.EidNymRhNym
		}
		if len(opts.AuditInfo) != 0 {
			ai, err := p.DeserializeAuditInfo(opts.AuditInfo)
			if err != nil {
				return nil, nil, err
			}

			signerMetadata = &bccsp.IdemixSignerMetadata{
				EidNymAuditData: ai.EidNymAuditData,
				RhNymAuditData:  ai.RhNymAuditData,
			}
		}
	}

	// Create the cryptographic evidence that this identity is valid
	sigOpts := &bccsp.IdemixSignerOpts{
		Credential: p.conf.Signer.Cred,
		Nym:        nymKey,
		IssuerPK:   p.IssuerPublicKey,
		Attributes: []bccsp.IdemixAttribute{
			{Type: bccsp.IdemixBytesAttribute},
			{Type: bccsp.IdemixIntAttribute},
			{Type: bccsp.IdemixHiddenAttribute},
			{Type: bccsp.IdemixHiddenAttribute},
		},
		RhIndex:  RHIndex,
		EidIndex: EIDIndex,
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
	id, err := NewMSPIdentityWithVerType(p.Idemix, NymPublicKey, role, ou, proof, p.verType)
	if err != nil {
		return nil, nil, err
	}
	sID := &MSPSigningIdentity{
		MSPIdentity:  id,
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
		if err := p.SignerService.RegisterSigner(raw, sID, sID); err != nil {
			return nil, nil, err
		}
	}

	var infoRaw []byte
	switch sigType {
	case bccsp.Standard:
		infoRaw = nil
	case bccsp.EidNymRhNym:
		auditInfo := &AuditInfo{
			Csp:             p.Csp,
			IssuerPublicKey: p.IssuerPublicKey,
			EidNymAuditData: sigOpts.Metadata.EidNymAuditData,
			RhNymAuditData:  sigOpts.Metadata.RhNymAuditData,
			Attributes: [][]byte{
				[]byte(p.conf.Signer.OrganizationalUnitIdentifier),
				[]byte(strconv.Itoa(GetIdemixRoleFromMSPRole(role))),
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

func (p *Provider) IsRemote() bool {
	return p.userKey == nil
}

func (p *Provider) DeserializeVerifier(raw []byte) (driver.Verifier, error) {
	r, err := p.Deserialize(raw, true)
	if err != nil {
		return nil, err
	}

	return r.Identity, nil
}

func (p *Provider) DeserializeSigner(raw []byte) (driver.Signer, error) {
	r, err := p.Deserialize(raw, true)
	if err != nil {
		return nil, err
	}

	nymKey, err := p.Csp.GetKey(r.NymPublicKey.SKI())
	if err != nil {
		return nil, errors.Wrap(err, "cannot find nym secret key")
	}

	si := &MSPSigningIdentity{
		MSPIdentity:  r.Identity,
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

func (p *Provider) Info(raw []byte, auditInfo []byte) (string, error) {
	r, err := p.Deserialize(raw, true)
	if err != nil {
		return "", err
	}

	eid := ""
	if len(auditInfo) != 0 {
		ai := &AuditInfo{
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

	return fmt.Sprintf("MSP.Idemix: [%s][%s][%s][%s][%s]", eid, view.Identity(raw).UniqueID(), r.SerializedIdentity.Mspid, r.OU.OrganizationalUnitIdentifier, r.Role.Role.String()), nil
}

func (p *Provider) String() string {
	return fmt.Sprintf("Idemix Provider [%s]", hash.Hashable(p.Ipk).String())
}

func (p *Provider) EnrollmentID() string {
	return p.conf.Signer.EnrollmentId
}

func (p *Provider) DeserializeSigningIdentity(raw []byte) (driver.SigningIdentity, error) {
	si := &m.SerializedIdentity{}
	err := proto.Unmarshal(raw, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to msp.SerializedIdentity{}")
	}

	serialized := new(m.SerializedIdemixIdentity)
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

	// Role
	role := &m.MSPRole{}
	err = proto.Unmarshal(serialized.Role, role)
	if err != nil {
		return nil, errors.Wrap(err, "cannot deserialize the role of the identity")
	}

	id, _ := NewMSPIdentityWithVerType(p.Idemix, NymPublicKey, role, ou, serialized.Proof, p.verType)
	if err := id.Validate(); err != nil {
		return nil, errors.Wrap(err, "cannot deserialize, invalid identity")
	}
	nymKey, err := p.Csp.GetKey(NymPublicKey.SKI())
	if err != nil {
		return nil, errors.Wrap(err, "cannot find nym secret key")
	}

	return &MSPSigningIdentity{
		MSPIdentity:  id,
		Cred:         p.conf.Signer.Cred,
		UserKey:      p.userKey,
		NymKey:       nymKey,
		EnrollmentId: p.conf.Signer.EnrollmentId,
	}, nil
}
