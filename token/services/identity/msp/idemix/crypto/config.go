/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/IBM/idemix"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix/crypto/protos-go/config"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
)

type (
	Config                   = config.IdemixConfig
	SerializedIdemixIdentity = config.SerializedIdemixIdentity
	SerializedIdentity       = msp.SerializedIdentity
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
	// CRI contains a serialized Credential Revocation Information
	CredentialRevocationInformation []byte `protobuf:"bytes,6,opt,name=credential_revocation_information,json=credentialRevocationInformation,proto3" json:"credential_revocation_information,omitempty"`
	// RevocationHandle is the handle used to single out this credential and determine its revocation status
	RevocationHandle string `protobuf:"bytes,7,opt,name=revocation_handle,json=revocationHandle,proto3" json:"revocation_handle,omitempty"`
	// CurveID specifies the name of the Idemix curve to use, defaults to 'amcl.Fp256bn'
	CurveID string `protobuf:"bytes,8,opt,name=curve_id,json=curveID" json:"curveID,omitempty"`
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

func NewConfig(dir string, id string) (*Config, error) {
	ipkBytes, err := ReadFile(filepath.Join(dir, idemix.IdemixConfigDirMsp, idemix.IdemixConfigFileIssuerPublicKey))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read issuer public key file")
	}
	return NewConfigWithIPK(ipkBytes, dir, id, true)
}

func NewConfigWithIPK(issuerPublicKey []byte, dir string, id string, ignoreVerifyOnlyWallet bool) (*Config, error) {
	conf, err := newConfigWithIPK(issuerPublicKey, dir, id, ignoreVerifyOnlyWallet)
	if err != nil {
		logger.Debugf("failed reading idemix msp configuration from [%s]: [%s], try adding 'msp'...", dir, err)
		// Try with "msp"
		conf, err = newConfigWithIPK(issuerPublicKey, filepath.Join(dir, "msp"), id, ignoreVerifyOnlyWallet)
		if err != nil {
			return nil, errors.Wrapf(err, "failed reading idemix msp configuration from [%s] and with 'msp'", dir)
		}
	}
	return conf, nil
}

func newConfigWithIPK(issuerPublicKey []byte, dir string, id string, ignoreVerifyOnlyWallet bool) (*Config, error) {
	mspConfig, err := NewIdemixConfig(issuerPublicKey, dir, id, ignoreVerifyOnlyWallet)
	if err != nil {
		// load it using the fabric-ca format
		mspConfig2, err2 := NewFabricCAIdemixConfig(issuerPublicKey, dir, id)
		if err2 != nil {
			return nil, errors.Wrapf(err2, "cannot get idemix msp config from [%s]: [%s]", dir, err)
		}
		mspConfig = mspConfig2
	}
	return mspConfig, nil
}

// NewIdemixConfig returns the configuration for the Idemix MSP of the specified type
func NewIdemixConfig(issuerPublicKey []byte, dir string, ID string, ignoreVerifyOnlyWallet bool) (*Config, error) {
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
	}

	return assembleConfig(issuerPublicKey, signer, ID)
}

// NewFabricCAIdemixConfig returns the configuration for the Idemix MSP generated by Fabric-CA
func NewFabricCAIdemixConfig(issuerPublicKey []byte, dir string, ID string) (*Config, error) {
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
		}
	} else {
		if !os.IsNotExist(errors.Cause(err)) {
			return nil, errors.Wrapf(err, "failed to read the content of signer config at [%s]", path)
		}
	}

	return assembleConfig(issuerPublicKey, signer, ID)
}

func NewConfigFromRawSigner(issuerPublicKey []byte, signerRaw []byte, ID string) (*Config, error) {
	var signer *config.IdemixSignerConfig
	if len(signerRaw) != 0 {
		signer = &config.IdemixSignerConfig{}
		if err := proto.Unmarshal(signerRaw, signer); err != nil {
			// is the format Fabric-CA generate?
			si := &SignerConfig{}
			if err2 := json.Unmarshal(signerRaw, si); err2 != nil {
				return nil, errors.Wrapf(
					errors.Wrapf(err, "failed to unmarhal IdemixSignerConfig"),
					"failed to unmarshal SignerConfig [%s]", err2)
			}
			signer = &config.IdemixSignerConfig{
				Cred:                            si.Cred,
				Sk:                              si.Sk,
				OrganizationalUnitIdentifier:    si.OrganizationalUnitIdentifier,
				Role:                            int32(si.Role),
				EnrollmentId:                    si.EnrollmentID,
				CredentialRevocationInformation: si.CredentialRevocationInformation,
				RevocationHandle:                si.RevocationHandle,
			}
		}
	}
	idemixConfig := &config.IdemixConfig{
		Name:   ID,
		Ipk:    issuerPublicKey,
		Signer: signer,
	}
	return idemixConfig, nil
}

func assembleConfig(issuerPublicKey []byte, signer *config.IdemixSignerConfig, ID string) (*Config, error) {
	idemixConfig := &config.IdemixConfig{
		Name:   ID,
		Ipk:    issuerPublicKey,
		Signer: signer,
	}
	return idemixConfig, nil
}
