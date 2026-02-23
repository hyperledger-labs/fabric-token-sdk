/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	idemix2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/multisig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
	htlc2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
)

// Deserializer deserializes verifiers associated with issuers, owners, and auditors.
type Deserializer struct {
	*common.Deserializer
}

// NewDeserializer returns a new zkatdlog deserializer.
func NewDeserializer(pp *v1.PublicParams) (*Deserializer, error) {
	if pp == nil {
		return nil, errors.New("failed to get deserializer: nil public parameters")
	}

	des := deserializer.NewTypedVerifierDeserializerMultiplex()
	for _, idemixIssuerPublicKey := range pp.IdemixIssuerPublicKeys {
		idemixDes, err := idemix2.NewDeserializer(idemixIssuerPublicKey.PublicKey, idemixIssuerPublicKey.Curve)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting idemix deserializer for passed public params [%d]", idemixIssuerPublicKey.Curve)
		}
		des.AddTypedVerifierDeserializer(idemix2.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(idemixDes, idemixDes))
	}
	des.AddTypedVerifierDeserializer(x509.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(&x509.IdentityDeserializer{}, &x509.AuditMatcherDeserializer{}))
	des.AddTypedVerifierDeserializer(htlc2.ScriptType, htlc.NewTypedIdentityDeserializer(des))
	des.AddTypedVerifierDeserializer(multisig.Multisig, multisig.NewTypedIdentityDeserializer(des, des))

	return &Deserializer{Deserializer: common.NewDeserializer(idemix2.IdentityType, des, des, des, des, des)}, nil
}

// TokenDeserializer deserializes zkatdlog tokens and metadata.
type TokenDeserializer struct{}

// DeserializeMetadata deserializes the passed bytes into a zkatdlog token metadata.
func (d *TokenDeserializer) DeserializeMetadata(raw []byte) (*token.Metadata, error) {
	metadata := &token.Metadata{}
	if err := metadata.Deserialize(raw); err != nil {
		return nil, err
	}

	return metadata, nil
}

// DeserializeToken deserializes the passed bytes into a zkatdlog token.
func (d *TokenDeserializer) DeserializeToken(raw []byte) (*token.Token, error) {
	token := &token.Token{}
	if err := token.Deserialize(raw); err != nil {
		return nil, err
	}

	return token, nil
}

// PublicParamsDeserializer deserializes zkatdlog public parameters.
type PublicParamsDeserializer struct{}

// DeserializePublicParams deserializes the passed bytes into zkatdlog public parameters.
func (p *PublicParamsDeserializer) DeserializePublicParams(raw []byte, name driver.TokenDriverName, version driver.TokenDriverVersion) (*v1.PublicParams, error) {
	return v1.NewPublicParamsFromBytes(raw, name, version)
}

// EIDRHDeserializer returns enrollment ID and revocation handle behind the owners of token.
type EIDRHDeserializer = deserializer.EIDRHDeserializer

// NewEIDRHDeserializer returns a new zkatdlog EIDRHDeserializer.
func NewEIDRHDeserializer() *EIDRHDeserializer {
	d := deserializer.NewEIDRHDeserializer()
	d.AddDeserializer(idemix2.IdentityType, &idemix2.AuditInfoDeserializer{})
	d.AddDeserializer(x509.IdentityType, &x509.AuditInfoDeserializer{})
	d.AddDeserializer(htlc2.ScriptType, htlc.NewAuditDeserializer(&idemix2.AuditInfoDeserializer{}))
	d.AddDeserializer(multisig.Multisig, &multisig.AuditInfoDeserializer{})

	return d
}
