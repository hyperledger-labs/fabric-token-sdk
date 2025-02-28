/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	msp2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
)

const (
	IdentityType identity.Type = "x509"
)

var logger = logging.MustGetLogger("token-sdk.services.identity.msp.x509")

type SignerService interface {
	RegisterSigner(identity driver.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error
}

type KeyManager struct {
	sID          driver.SigningIdentity
	id           []byte
	enrollmentID string
}

// NewKeyManager returns a new X509 provider with the passed BCCSP configuration.
// If the configuration path contains the secret key,
// then the provider can generate also signatures, otherwise it cannot.
func NewKeyManager(mspConfigPath, keyStorePath, mspID string, signerService SignerService, bccspConfig *msp2.BCCSP) (*KeyManager, *msp2.Config, error) {
	conf, err := msp2.GetConfig(mspConfigPath, mspID)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "could not get msp config from dir [%s]", mspConfigPath)
	}
	p, conf, err := NewKeyManagerFromConf(conf, mspConfigPath, keyStorePath, mspID, signerService, bccspConfig)
	if err != nil {
		return nil, nil, err
	}
	return p, conf, nil
}

func NewKeyManagerFromConf(conf *msp2.Config, mspConfigPath, keyStorePath, mspID string, signerService SignerService, bccspConfig *msp2.BCCSP) (*KeyManager, *msp2.Config, error) {
	if conf == nil {
		logger.Debugf("load msp config from [%s:%s]", mspConfigPath, mspID)
		var err error
		conf, err = msp2.GetConfig(mspConfigPath, mspID)
		if err != nil {
			return nil, nil, errors.WithMessagef(err, "could not get msp config from dir [%s]", mspConfigPath)
		}
	}
	logger.Debugf("msp config [%d]", conf.Type)
	p, err := newSigningKeyManager(conf, mspConfigPath, keyStorePath, mspID, signerService, bccspConfig)
	if err == nil {
		return p, conf, nil
	}
	// load as verify only
	p, conf, err = newVerifyingKeyManager(conf, mspConfigPath, mspID)
	if err != nil {
		return nil, nil, err
	}
	return p, conf, err
}

func newSigningKeyManager(conf *msp2.Config, mspConfigPath, keyStorePath, mspID string, signerService SignerService, bccspConfig *msp2.BCCSP) (*KeyManager, error) {
	sID, err := msp2.GetSigningIdentity(conf, mspConfigPath, keyStorePath, bccspConfig)
	if err != nil {
		return nil, err
	}
	idRaw, err := sID.Serialize()
	if err != nil {
		return nil, err
	}
	if signerService != nil {
		logger.Debugf("register signer [%s][%s]", mspID, driver.Identity(idRaw))
		err = signerService.RegisterSigner(idRaw, sID, sID, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed registering x509 signer")
		}
	}
	return newKeyManager(sID, idRaw)
}

func newVerifyingKeyManager(conf *msp2.Config, mspConfigPath, mspID string) (*KeyManager, *msp2.Config, error) {
	conf, err := msp2.RemoveSigningIdentityInfo(conf)
	if err != nil {
		return nil, nil, err
	}
	idRaw, err := msp2.SerializeFromMSP(conf, mspID, mspConfigPath)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to load msp identity at [%s]", mspConfigPath)
	}
	p, err := newKeyManager(nil, idRaw)
	if err != nil {
		return nil, nil, err
	}
	return p, conf, nil
}

func newKeyManager(sID driver.SigningIdentity, id []byte) (*KeyManager, error) {
	enrollmentID, err := msp2.GetEnrollmentID(id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get enrollment id")
	}
	return &KeyManager{sID: sID, id: id, enrollmentID: enrollmentID}, nil
}

func (p *KeyManager) IsRemote() bool {
	return p.sID == nil
}

func (p *KeyManager) Identity([]byte) (driver.Identity, []byte, error) {
	revocationHandle, err := msp2.GetRevocationHandle(p.id)
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

func (p *KeyManager) EnrollmentID() string {
	return p.enrollmentID
}

func (p *KeyManager) DeserializeVerifier(raw []byte) (driver.Verifier, error) {
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

func (p *KeyManager) DeserializeSigner(raw []byte) (driver.Signer, error) {
	return nil, errors.New("not supported")
}

func (p *KeyManager) Info(raw []byte, auditInfo []byte) (string, error) {
	si := &msp.SerializedIdentity{}
	err := proto.Unmarshal(raw, si)
	if err != nil {
		return "", err
	}
	cert, err := msp2.PemDecodeCert(si.IdBytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("MSP.x509: [%s][%s][%s]", driver.Identity(raw).UniqueID(), si.Mspid, cert.Subject.CommonName), nil
}

func (p *KeyManager) Anonymous() bool {
	return false
}

func (p *KeyManager) String() string {
	return fmt.Sprintf("X509 KeyManager for EID [%s]", p.enrollmentID)
}

func (p *KeyManager) SerializedIdentity() (driver.SigningIdentity, error) {
	return p.sID, nil
}

func (p *KeyManager) IdentityType() identity.Type {
	return IdentityType
}
