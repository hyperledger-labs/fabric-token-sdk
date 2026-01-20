/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"context"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger/fabric-lib-go/bccsp"
)

const (
	IdentityType identity.Type = "x509"
)

var logger = logging.MustGetLogger()

type SignerService interface {
	RegisterSigner(ctx context.Context, identity driver.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error
}

type KeyManager struct {
	sID                driver.SigningIdentity
	id                 []byte
	enrollmentID       string
	identityDescriptor *idriver.IdentityDescriptor
	bccspConfig        *crypto.BCCSP
	keyStore           bccsp.KeyStore
}

// NewKeyManager returns a new X509 provider with the passed BCCSP configuration.
// If the configuration path contains the secret key,
// then the provider can generate also signatures, otherwise it cannot.
func NewKeyManager(
	path string,
	bccspConfig *crypto.BCCSP,
	keyStore crypto.KeyStore,
) (*KeyManager, *crypto.Config, error) {
	p, conf, err := NewKeyManagerFromConf(nil, path, "", bccspConfig, keyStore)
	if err != nil {
		return nil, nil, err
	}
	return p, conf, nil
}

func NewKeyManagerFromConf(
	conf *crypto.Config,
	configPath, keyStoreDirName string,
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
			logger.Errorf("failed loading x509 configuration [%+v]", err)
			return nil, nil, errors.WithMessagef(err, "could not get config from dir [%s]", configPath)
		}
	}
	logger.Debugf("load x509 config, check version...")
	// enforce version
	if conf.Version != crypto.ProtobufProtocolVersionV1 {
		return nil, nil, errors.Errorf("unsupported protocol version: %d", conf.Version)
	}
	logger.Debugf("load x509 config, new signing key manager...")
	p, err := newSigningKeyManager(conf, bccspConfig, keyStore)
	if err == nil {
		logger.Debugf("load x509 config, new signing key manager...done [%v]", p)
		return p, conf, nil
	}
	// load as verify only
	p, conf, err = newVerifyingKeyManager(conf, bccspConfig)
	if err != nil {
		return nil, nil, err
	}
	return p, conf, nil
}

func newSigningKeyManager(conf *crypto.Config, bccspConfig *crypto.BCCSP, keyStore crypto.KeyStore) (*KeyManager, error) {
	sID, err := crypto.GetSigningIdentity(conf, bccspConfig, keyStore)
	if err != nil {
		return nil, err
	}
	idRaw, err := sID.Serialize()
	if err != nil {
		return nil, err
	}
	return newKeyManager(sID, idRaw, bccspConfig, keyStore)
}

func newVerifyingKeyManager(conf *crypto.Config, bccspConfig *crypto.BCCSP) (*KeyManager, *crypto.Config, error) {
	conf, err := crypto.RemovePrivateSigner(conf)
	if err != nil {
		return nil, nil, err
	}
	idRaw, err := crypto.SerializeIdentity(conf)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to load identity")
	}
	p, err := newKeyManager(nil, idRaw, bccspConfig, nil)
	if err != nil {
		return nil, nil, err
	}
	return p, conf, nil
}

func newKeyManager(sID driver.SigningIdentity, id []byte, bccspConfig *crypto.BCCSP, keyStore bccsp.KeyStore) (*KeyManager, error) {
	enrollmentID, err := crypto.GetEnrollmentID(id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get enrollment id")
	}
	revocationHandle, err := crypto.GetRevocationHandle(id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting revocation handle")
	}
	ai := &AuditInfo{
		EID: enrollmentID,
		RH:  revocationHandle,
	}
	auditInfoRaw, err := ai.Bytes()
	if err != nil {
		return nil, err
	}
	identityDescriptor := &idriver.IdentityDescriptor{
		Identity:  id,
		AuditInfo: auditInfoRaw,
		Signer:    sID,
	}
	return &KeyManager{
		identityDescriptor: identityDescriptor,
		sID:                sID,
		id:                 id,
		enrollmentID:       enrollmentID,
		bccspConfig:        bccspConfig,
		keyStore:           keyStore,
	}, nil
}

func (p *KeyManager) IsRemote() bool {
	return p.sID == nil
}

func (p *KeyManager) Identity(context.Context, []byte) (*idriver.IdentityDescriptor, error) {
	return p.identityDescriptor, nil
}

func (p *KeyManager) EnrollmentID() string {
	return p.enrollmentID
}

func (p *KeyManager) DeserializeVerifier(ctx context.Context, raw []byte) (driver.Verifier, error) {
	return crypto.DeserializeVerifier(raw)
}

func (p *KeyManager) DeserializeSigner(ctx context.Context, raw []byte) (driver.Signer, error) {
	return crypto.DeserializeIdentity(raw, p.bccspConfig, p.keyStore)
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
