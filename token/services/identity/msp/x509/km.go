/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
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
func NewKeyManager(path, mspID string, signerService SignerService, bccspConfig *msp.BCCSP) (*KeyManager, *msp.Config, error) {
	conf, err := msp.LoadConfig(path, "", mspID)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "could not get msp config from dir [%s]", path)
	}
	p, conf, err := NewKeyManagerFromConf(conf, path, "", mspID, signerService, bccspConfig)
	if err != nil {
		return nil, nil, err
	}
	return p, conf, nil
}

func NewKeyManagerFromConf(conf *msp.Config, mspConfigPath, keyStoreDirName, mspID string, signerService SignerService, bccspConfig *msp.BCCSP) (*KeyManager, *msp.Config, error) {
	if conf == nil {
		logger.Debugf("load msp config from [%s:%s]", mspConfigPath, mspID)
		var err error
		conf, err = msp.LoadConfig(mspConfigPath, keyStoreDirName, mspID)
		if err != nil {
			return nil, nil, errors.WithMessagef(err, "could not get msp config from dir [%s]", mspConfigPath)
		}
	}
	logger.Debugf("msp config [%d]", conf.Type)
	p, err := newSigningKeyManager(conf, mspID, signerService, bccspConfig)
	if err == nil {
		return p, conf, nil
	}
	// load as verify only
	p, conf, err = newVerifyingKeyManager(conf, mspID)
	if err != nil {
		return nil, nil, err
	}
	return p, conf, err
}

func newSigningKeyManager(conf *msp.Config, mspID string, signerService SignerService, bccspConfig *msp.BCCSP) (*KeyManager, error) {
	sID, err := msp.GetSigningIdentity(conf, bccspConfig)
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

func newVerifyingKeyManager(conf *msp.Config, mspID string) (*KeyManager, *msp.Config, error) {
	conf, err := msp.RemoveSigningIdentityInfo(conf)
	if err != nil {
		return nil, nil, err
	}
	idRaw, err := msp.SerializeFromMSP(conf, mspID)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to load msp identity")
	}
	p, err := newKeyManager(nil, idRaw)
	if err != nil {
		return nil, nil, err
	}
	return p, conf, nil
}

func newKeyManager(sID driver.SigningIdentity, id []byte) (*KeyManager, error) {
	enrollmentID, err := msp.GetEnrollmentID(id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get enrollment id")
	}
	return &KeyManager{sID: sID, id: id, enrollmentID: enrollmentID}, nil
}

func (p *KeyManager) IsRemote() bool {
	return p.sID == nil
}

func (p *KeyManager) Identity([]byte) (driver.Identity, []byte, error) {
	revocationHandle, err := msp.GetRevocationHandle(p.id)
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
	return msp.DeserializeVerifier(raw)
}

func (p *KeyManager) DeserializeSigner(raw []byte) (driver.Signer, error) {
	return nil, errors.New("not supported")
}

func (p *KeyManager) Info(raw []byte, auditInfo []byte) (string, error) {
	return msp.Info(raw)
}

func (p *KeyManager) Anonymous() bool {
	return false
}

func (p *KeyManager) String() string {
	return fmt.Sprintf("X509 KeyManager for EID [%s]", p.enrollmentID)
}

func (p *KeyManager) IdentityType() identity.Type {
	return IdentityType
}

func (p *KeyManager) SigningIdentity() (driver.SigningIdentity, error) {
	return p.sID, nil
}
