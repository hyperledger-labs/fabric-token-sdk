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
	msp2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509/msp"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.services.identity.msp.x509")

type SignerService interface {
	RegisterSigner(identity view.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error
}

type Provider struct {
	sID          driver.SigningIdentity
	id           []byte
	enrollmentID string
}

// NewProvider returns a new X509 provider with the passed BCCSP configuration.
// If the configuration path contains the secret key,
// then the provider can generate also signatures, otherwise it cannot.
func NewProvider(mspConfigPath, keyStorePath, mspID string, signerService SignerService, bccspConfig *msp2.BCCSP) (*Provider, error) {
	p, err := newSigningProvider(mspConfigPath, keyStorePath, mspID, signerService, bccspConfig)
	if err == nil {
		return p, nil
	}

	// load as verify only
	return newVerifyingProvider(mspConfigPath, mspID)
}

func newSigningProvider(mspConfigPath, keyStorePath, mspID string, signerService SignerService, bccspConfig *msp2.BCCSP) (*Provider, error) {
	sID, err := msp2.GetSigningIdentity(mspConfigPath, keyStorePath, mspID, bccspConfig)
	if err != nil {
		return nil, err
	}
	idRaw, err := sID.Serialize()
	if err != nil {
		return nil, err
	}
	if signerService != nil {
		logger.Debugf("register signer [%s][%s]", mspID, view.Identity(idRaw))
		err = signerService.RegisterSigner(idRaw, sID, sID, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed registering x509 signer")
		}
	}
	return newProvider(sID, idRaw)
}

func newVerifyingProvider(mspConfigPath, mspID string) (*Provider, error) {
	idRaw, err := msp2.SerializeFromMSP(mspID, mspConfigPath)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to load msp identity at [%s]", mspConfigPath)
	}
	return newProvider(nil, idRaw)
}

func newProvider(sID driver.SigningIdentity, id []byte) (*Provider, error) {
	enrollmentID, err := GetEnrollmentID(id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get enrollment id")
	}
	return &Provider{sID: sID, id: id, enrollmentID: enrollmentID}, nil
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
		EID: p.enrollmentID,
		RH:  revocationHandle,
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
	genericPublicKey, err := msp2.PemDecodeKey(si.IdBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed parsing received public key")
	}
	publicKey, ok := genericPublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("expected *ecdsa.PublicKey")
	}

	// TODO: check the validity of the identity against the msp

	return msp2.NewECDSAVerifier(publicKey), nil
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
	cert, err := msp2.PemDecodeCert(si.IdBytes)
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
