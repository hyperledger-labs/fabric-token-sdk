/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/IBM/idemix"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto/protos-go/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
)

type (
	Config                   = config.IdemixConfig
	SerializedIdemixIdentity = config.SerializedIdemixIdentity
)

const (
	ExtraPathElement = "msp"

	ProtobufProtocolVersionV1 uint64 = 1
)

// SignerConfig contains the crypto material to set up an idemix signing identity
type SignerConfig struct {
	// Cred represents the serialized idemix credential of the default signer
	Cred []byte `protobuf:"bytes,1,opt,name=Cred,proto3" json:"Cred,omitempty"`
	// Sk is the secret key of the default signer, corresponding to credential Cred
	Sk []byte `protobuf:"bytes,2,opt,name=Sk,proto3" json:"Sk,omitempty"`
	// OrganizationalUnitIdentifier defines the organizational unit the default signer is in
	OrganizationalUnitIdentifier string `protobuf:"bytes,3,opt,name=organizational_unit_identifier,json=organizationalUnitIdentifier" json:"organizational_unit_identifier,omitempty"`
	// Role defines whether the default signer is admin, member, peer, or client
	Role int `protobuf:"varint,4,opt,name=role,json=role" json:"role,omitempty"`
	// EnrollmentID contains the enrollment id of this signer
	EnrollmentID string `protobuf:"bytes,5,opt,name=enrollment_id,json=enrollmentId" json:"enrollment_id,omitempty"`
	// CRI contains a serialized CredentialRevocationInformation
	CredentialRevocationInformation []byte `protobuf:"bytes,6,opt,name=credential_revocation_information,json=credentialRevocationInformation,proto3" json:"credential_revocation_information,omitempty"`
	// RevocationHandle is the handle used to single out this credential and determine its revocation status
	RevocationHandle string `protobuf:"bytes,7,opt,name=revocation_handle,json=revocationHandle,proto3" json:"revocation_handle,omitempty"`
	// CurveID specifies the name of the Idemix curve to use, defaults to 'amcl.Fp256bn'
	CurveID string `protobuf:"bytes,8,opt,name=curve_id,json=curveID" json:"curveID,omitempty"`
	// Schema contains the version of the schema used by this credential
	Schema string `protobuf:"bytes,9,opt,name=schema,json=schema" json:"schema,omitempty"`
}

const (
	ConfigDirUser    = "user"
	ConfigFileSigner = "SignerConfig"
)

func ReadFile(file string) ([]byte, error) {
	fileCont, err := os.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read file %s", file)
	}

	return fileCont, nil
}

func NewConfig(dir string) (*Config, error) {
	ipkBytes, err := ReadFile(filepath.Join(dir, idemix.IdemixConfigDirMsp, idemix.IdemixConfigFileIssuerPublicKey))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read issuer public key file")
	}
	return NewConfigWithIPK(ipkBytes, dir, true)
}

func NewConfigWithIPK(issuerPublicKey []byte, dir string, ignoreVerifyOnlyWallet bool) (*Config, error) {
	conf, err := newConfigWithIPK(issuerPublicKey, dir, ignoreVerifyOnlyWallet)
	if err != nil {
		logger.Debugf("failed reading idemix configuration from [%s]: [%s], try adding extra path element...", dir, err)
		// Try with ExtraPathElement
		conf, err = newConfigWithIPK(issuerPublicKey, filepath.Join(dir, ExtraPathElement), ignoreVerifyOnlyWallet)
		if err != nil {
			return nil, errors.Wrapf(err, "failed reading idemix configuration from [%s] and with extra path element", dir)
		}
	}
	return conf, nil
}

func newConfigWithIPK(issuerPublicKey []byte, dir string, ignoreVerifyOnlyWallet bool) (*Config, error) {
	config, err := NewIdemixConfig(issuerPublicKey, dir, ignoreVerifyOnlyWallet)
	if err != nil {
		// load it using the fabric-ca format
		config2, err2 := NewFabricCAIdemixConfig(issuerPublicKey, dir)
		if err2 != nil {
			return nil, errors.Wrapf(err2, "cannot get idemix config from [%s]: [%s]", dir, err)
		}
		config = config2
	}
	return config, nil
}

// NewIdemixConfig returns the configuration for Idemix
func NewIdemixConfig(issuerPublicKey []byte, dir string, ignoreVerifyOnlyWallet bool) (*Config, error) {
	signerConfigPath := filepath.Join(dir, idemix.IdemixConfigDirUser, idemix.IdemixConfigFileSigner)
	if ignoreVerifyOnlyWallet {
		logger.Debugf("check the existence of SignerConfigFull")
		// check if `SignerConfigFull` exists, if yes, use that file
		path := filepath.Join(dir, idemix.IdemixConfigDirUser, SignerConfigFull)
		_, err := os.Stat(path)
		if err == nil {
			logger.Debugf("SignerConfigFull found, use it")
			signerConfigPath = path
		}
	}
	var signer *config.IdemixSignerConfig
	signerBytes, err := os.ReadFile(signerConfigPath)
	if err == nil {
		signer = &config.IdemixSignerConfig{}
		err = proto.Unmarshal(signerBytes, signer)
		if err != nil {
			return nil, err
		}
	} else {
		logger.Debugf("cannot read the signer config file [%s]: [%s]", signerConfigPath, err)
	}

	return assembleConfig(issuerPublicKey, signer)
}

// NewFabricCAIdemixConfig returns the configuration for Idemix generated by Fabric-CA
func NewFabricCAIdemixConfig(issuerPublicKey []byte, dir string) (*Config, error) {
	var signer *config.IdemixSignerConfig
	path := filepath.Join(dir, ConfigDirUser, ConfigFileSigner)
	signerBytes, err := ReadFile(path)
	if err == nil {
		// signerBytes is a json structure, convert it to protobuf
		si := &SignerConfig{}
		if err := json.Unmarshal(signerBytes, si); err != nil {
			return nil, errors.Wrapf(err, "failed to json unmarshal signer config read at [%s]", path)
		}
		signer = &config.IdemixSignerConfig{
			Cred:                            si.Cred,
			Sk:                              si.Sk,
			OrganizationalUnitIdentifier:    si.OrganizationalUnitIdentifier,
			Role:                            int32(si.Role),
			EnrollmentId:                    si.EnrollmentID,
			CredentialRevocationInformation: si.CredentialRevocationInformation,
			RevocationHandle:                si.RevocationHandle,
			Schema:                          si.Schema,
		}
	} else {
		if !os.IsNotExist(errors.Cause(err)) {
			return nil, errors.Wrapf(err, "failed to read the content of signer config at [%s]", path)
		}
	}

	return assembleConfig(issuerPublicKey, signer)
}

func NewConfigFromRaw(issuerPublicKey []byte, configRaw []byte) (*Config, error) {
	config := &config.IdemixConfig{}
	if err := proto.Unmarshal(configRaw, config); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal idemix config at [%s]", string(configRaw))
	}
	// match public keys
	if !bytes.Equal(issuerPublicKey, config.Ipk) {
		return nil, errors.Errorf("public key does not match [%s]=[%s]", utils.Hashable(config.Ipk), utils.Hashable(issuerPublicKey))
	}
	// match version, supported are: ProtobufProtocolVersionV1
	if config.Version != ProtobufProtocolVersionV1 {
		return nil, errors.Errorf("unsupported protocol version: %d", config.Version)
	}

	return config, nil
}

func assembleConfig(issuerPublicKey []byte, signer *config.IdemixSignerConfig) (*Config, error) {
	idemixConfig := &config.IdemixConfig{
		Version: ProtobufProtocolVersionV1,
		Ipk:     issuerPublicKey,
		Signer:  signer,
	}
	return idemixConfig, nil
}
