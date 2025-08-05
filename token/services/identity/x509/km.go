/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"context"
	"fmt"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

const (
	IdentityType identity.Type = "x509"
)

var logger = logging.MustGetLogger()

type SignerService interface {
	RegisterSigner(ctx context.Context, identity driver.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error
}

type KeyManager struct {
	sID          driver.SigningIdentity
	id           []byte
	enrollmentID string
}

// NewKeyManager returns a new X509 provider with the passed BCCSP configuration.
// If the configuration path contains the secret key,
// then the provider can generate also signatures, otherwise it cannot.
func NewKeyManager(
	path string,
	signerService SignerService,
	bccspConfig *crypto.BCCSP,
	keyStore crypto.KeyStore,
) (*KeyManager, *crypto.Config, error) {
	p, conf, err := NewKeyManagerFromConf(nil, path, "", signerService, bccspConfig, keyStore)
	if err != nil {
		return nil, nil, err
	}
	return p, conf, nil
}

func NewKeyManagerFromConf(
	conf *crypto.Config,
	configPath, keyStoreDirName string,
	signerService SignerService,
	bccspConfig *crypto.BCCSP,
	keyStore crypto.KeyStore,
) (*KeyManager, *crypto.Config, error) {
	if keyStore == nil {
		return nil, nil, errors.New("no keyStore provided")
	}
	if conf == nil {
		logger.Debugf("load x509 config from [%s]", configPath)
		var err error
		conf, err = crypto.LoadConfig(configPath, keyStoreDirName)
		if err != nil {
			return nil, nil, errors.WithMessagef(err, "could not get config from dir [%s]", configPath)
		}
	}
	// enforce version
	if conf.Version != crypto.ProtobufProtocolVersionV1 {
		return nil, nil, errors.Errorf("unsupported protocol version: %d", conf.Version)
	}
	p, err := newSigningKeyManager(conf, signerService, bccspConfig, keyStore)
	if err == nil {
		return p, conf, nil
	}
	// load as verify only
	p, conf, err = newVerifyingKeyManager(conf)
	if err != nil {
		return nil, nil, err
	}
	return p, conf, err
}

func newSigningKeyManager(conf *crypto.Config, signerService SignerService, bccspConfig *crypto.BCCSP, keyStore crypto.KeyStore) (*KeyManager, error) {
	sID, err := crypto.GetSigningIdentity(conf, bccspConfig, keyStore)
	if err != nil {
		return nil, err
	}
	idRaw, err := sID.Serialize()
	if err != nil {
		return nil, err
	}
	if signerService != nil {
		logger.Debugf("register signer [%s]", driver.Identity(idRaw))
		err = signerService.RegisterSigner(context.Background(), idRaw, sID, sID, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed registering x509 signer")
		}
	}
	return newKeyManager(sID, idRaw)
}

func newVerifyingKeyManager(conf *crypto.Config) (*KeyManager, *crypto.Config, error) {
	conf, err := crypto.RemovePrivateSigner(conf)
	if err != nil {
		return nil, nil, err
	}
	idRaw, err := crypto.SerializeIdentity(conf)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to load identity")
	}
	p, err := newKeyManager(nil, idRaw)
	if err != nil {
		return nil, nil, err
	}
	return p, conf, nil
}

func newKeyManager(sID driver.SigningIdentity, id []byte) (*KeyManager, error) {
	enrollmentID, err := crypto.GetEnrollmentID(id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get enrollment id")
	}
	return &KeyManager{sID: sID, id: id, enrollmentID: enrollmentID}, nil
}

func (p *KeyManager) IsRemote() bool {
	return p.sID == nil
}

func (p *KeyManager) Identity(context.Context, []byte) (driver.Identity, []byte, error) {
	revocationHandle, err := crypto.GetRevocationHandle(p.id)
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

func (p *KeyManager) DeserializeVerifier(ctx context.Context, raw []byte) (driver.Verifier, error) {
	return crypto.DeserializeVerifier(raw)
}

func (p *KeyManager) DeserializeSigner(ctx context.Context, raw []byte) (driver.Signer, error) {
	return nil, errors.New("not supported")
}

func (p *KeyManager) Info(ctx context.Context, raw []byte, auditInfo []byte) (string, error) {
	return crypto.Info(raw)
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

func (p *KeyManager) SigningIdentity() driver.SigningIdentity {
	return p.sID
}
