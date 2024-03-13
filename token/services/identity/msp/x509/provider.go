/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/config"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.services.identity.msp.x509")

type SignerService interface {
	RegisterSigner(identity view.Identity, signer driver.Signer, verifier driver.Verifier) error
}

type Provider struct {
	sID          driver.SigningIdentity
	id           []byte
	enrollmentID string
}

// NewProvider returns a new X509 provider. If the configuration path contains the secret key,
// then the provider can generate also signatures, otherwise it cannot.
func NewProvider(mspConfigPath, keyStorePath, mspID string, signerService SignerService) (*Provider, error) {
	return NewProviderWithBCCSPConfig(mspConfigPath, keyStorePath, mspID, signerService, nil)
}

// NewProviderWithBCCSPConfig returns a new X509 provider with the passed BCCSP configuration.
// If the configuration path contains the secret key,
// then the provider can generate also signatures, otherwise it cannot.
func NewProviderWithBCCSPConfig(mspConfigPath, keyStorePath, mspID string, signerService SignerService, bccspConfig *config.BCCSP) (*Provider, error) {
	p, err := newProvider(mspConfigPath, keyStorePath, mspID, signerService, bccspConfig)
	if err == nil {
		return p, nil
	}

	// load as verify only
	idRaw, err := SerializeFromMSP(mspID, mspConfigPath)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to load msp identity at [%s]", mspConfigPath)
	}
	enrollmentID, err := GetEnrollmentID(idRaw)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to extract endorllment id from msp identity at [%s]", mspConfigPath)
	}
	return &Provider{id: idRaw, enrollmentID: enrollmentID}, nil
}

func newProvider(mspConfigPath, keyStorePath, mspID string, signerService SignerService, bccspConfig *config.BCCSP) (*Provider, error) {
	sID, err := GetSigningIdentity(mspConfigPath, keyStorePath, mspID, bccspConfig)
	if err != nil {
		return nil, err
	}
	idRaw, err := sID.Serialize()
	if err != nil {
		return nil, err
	}
	if signerService != nil {
		logger.Debugf("register signer [%s][%s]", mspID, view.Identity(idRaw))
		err = signerService.RegisterSigner(idRaw, sID, sID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed registering x509 signer")
		}
	}
	enrollmentID, err := GetEnrollmentID(idRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting enrollment id for [%s:%s]", mspConfigPath, mspID)
	}

	return &Provider{sID: sID, id: idRaw, enrollmentID: enrollmentID}, nil
}

func (p *Provider) IsRemote() bool {
	return p.sID == nil
}

func (p *Provider) Identity(opts *common.IdentityOptions) (view.Identity, []byte, error) {
	revocationHandle, err := GetRevocationHandle(p.id)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed getting revocation handle")
	}
	ai := &AuditInfo{
		EnrollmentId:     p.enrollmentID,
		RevocationHandle: revocationHandle,
	}
	infoRaw, err := ai.Bytes()
	if err != nil {
		return nil, nil, err
	}

	return p.id, infoRaw, nil
}

func (p *Provider) EnrollmentID() string {
	return p.enrollmentID
}

func (p *Provider) DeserializeVerifier(raw []byte) (driver.Verifier, error) {
	si := &msp.SerializedIdentity{}
	err := proto.Unmarshal(raw, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to msp.SerializedIdentity{}")
	}
	genericPublicKey, err := PemDecodeKey(si.IdBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed parsing received public key")
	}
	publicKey, ok := genericPublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("expected *ecdsa.PublicKey")
	}

	// TODO: check the validity of the identity against the msp

	return NewECDSAVerifier(publicKey), nil
}

func (p *Provider) DeserializeSigner(raw []byte) (driver.Signer, error) {
	return nil, errors.New("not supported")
}

func (p *Provider) Info(raw []byte, auditInfo []byte) (string, error) {
	si := &msp.SerializedIdentity{}
	err := proto.Unmarshal(raw, si)
	if err != nil {
		return "", err
	}
	cert, err := PemDecodeCert(si.IdBytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("MSP.x509: [%s][%s][%s]", view.Identity(raw).UniqueID(), si.Mspid, cert.Subject.CommonName), nil
}

func (p *Provider) SerializedIdentity() (driver.SigningIdentity, error) {
	return p.sID, nil
}

func (p *Provider) String() string {
	return fmt.Sprintf("X509 Provider for EID [%s]", p.enrollmentID)
}
