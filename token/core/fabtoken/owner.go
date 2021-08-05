/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"

	"github.com/golang/protobuf/proto"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

const (
	SerializedIdentityType = "si"
)

type ByteStringer func([]byte) string

var (
	typeFormatters = map[string]ByteStringer{
		SerializedIdentityType: serializedIdentityToBytes,
	}
)

func RegisterTypeFormatter(t string, stringer ByteStringer) {
	typeFormatters[t] = stringer
}

func serializedIdentityToBytes(in []byte) string {
	si := &msp.SerializedIdentity{}
	err := proto.Unmarshal(in, si)
	if err != nil {
		return fmt.Sprintf("badly encoded identity (%v)", err)
	}
	block, _ := pem.Decode(si.IdBytes)
	if block == nil {
		return fmt.Sprintf("badly encoded PEM (%s)", base64.StdEncoding.EncodeToString(si.IdBytes))
	}
	if block.Type != "CERTIFICATE" {
		return fmt.Sprintf("PEM with invalid type (%s)", block.Type)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Sprintf("badly encoded certificate (%v)", err)
	}
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return fmt.Sprintf("badly encoded public key (%v)", err)
	}
	return fmt.Sprintf("{MSP: '%s', PubKey: '%s'}", si.Mspid, base64.StdEncoding.EncodeToString(pubKeyBytes))
}

// RawOwnerIdentityDeserializer takes as MSP identity and returns an ECDSA verifier
type RawOwnerIdentityDeserializer struct {
	*fabric.MSPX509IdentityDeserializer
}

func NewRawOwnerIdentityDeserializer() *RawOwnerIdentityDeserializer {
	return &RawOwnerIdentityDeserializer{
		MSPX509IdentityDeserializer: &fabric.MSPX509IdentityDeserializer{},
	}
}

func (deserializer *RawOwnerIdentityDeserializer) GetVerifier(id view.Identity) (driver.Verifier, error) {
	return deserializer.MSPX509IdentityDeserializer.GetVerifier(id)
}

func (deserializer *RawOwnerIdentityDeserializer) DeserializeVerifier(raw []byte) (driver2.Verifier, error) {
	return deserializer.GetVerifier(raw)
}

func (deserializer *RawOwnerIdentityDeserializer) DeserializeSigner(raw []byte) (driver2.Signer, error) {
	return nil, errors.Errorf("signer deserialization not supported")
}

func (deserializer *RawOwnerIdentityDeserializer) Info(raw []byte, auditInfo []byte) (string, error) {
	return "info not supported", nil
}
