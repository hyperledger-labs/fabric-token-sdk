/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509"
	"github.com/pkg/errors"
)

// Deserializer deserializes verifiers associated with issuers, owners, and auditors
type Deserializer struct {
	*common.Deserializer
}

// NewDeserializer returns a deserializer
func NewDeserializer(pp *crypto.PublicParams) (*Deserializer, error) {
	if pp == nil {
		return nil, errors.New("failed to get deserializer: nil public parameters")
	}
	idemixDes, err := idemix.NewDeserializer(pp.IdemixIssuerPK, pp.IdemixCurveID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting idemix deserializer for passed public params [%d]", pp.IdemixCurveID)
	}
	return &Deserializer{
		Deserializer: common.NewDeserializer(
			msp.IdemixIdentity,
			&x509.MSPIdentityDeserializer{},
			idemixDes,
			&x509.MSPIdentityDeserializer{},
			idemixDes,
		),
	}, nil
}

type TokenDeserializer struct{}

func (d *TokenDeserializer) DeserializeMetadata(raw []byte) (*token.Metadata, error) {
	metadata := &token.Metadata{}
	if err := metadata.Deserialize(raw); err != nil {
		return nil, err
	}
	return metadata, nil
}

func (d *TokenDeserializer) DeserializeToken(raw []byte) (*token.Token, error) {
	token := &token.Token{}
	if err := token.Deserialize(raw); err != nil {
		return nil, err
	}
	return token, nil
}

type PublicParamsDeserializer struct{}

func (p *PublicParamsDeserializer) DeserializePublicParams(raw []byte, label string) (*crypto.PublicParams, error) {
	return crypto.NewPublicParamsFromBytes(raw, label)
}

// EIDRHDeserializer returns enrollment ID and revocation handle behind the owners of token
type EIDRHDeserializer struct {
	*common.EIDRHDeserializer[*idemix.AuditInfo]
}

// NewEIDRHDeserializer returns an enrollmentService
func NewEIDRHDeserializer() *EIDRHDeserializer {
	return &EIDRHDeserializer{
		EIDRHDeserializer: common.NewEIDRHDeserializer[*idemix.AuditInfo](&idemix.Idemix{}),
	}
}
