/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"encoding/asn1"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

const (
	SerializedIdentityType = "si"
)

// RawOwner encodes an owner of an identity
type RawOwner struct {
	// Type encodes the type of the owner (currently it can only be a SerializedIdentity)
	Type string `protobuf:"bytes,1,opt,name=type,json=type,proto3" json:"type,omitempty"`
	// Identity encodes the identity
	Identity []byte `protobuf:"bytes,2,opt,name=identity,proto3" json:"identity,omitempty"`
}

func UnmarshallRawOwner(id view.Identity) (*RawOwner, error) {
	si := &RawOwner{}
	_, err := asn1.Unmarshal(id, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to RawOwner")
	}
	return si, nil
}

func MarshallRawOwner(o *RawOwner) (view.Identity, error) {
	raw, err := asn1.Marshal(*o)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to RawOwner")
	}
	return raw, nil
}

type DeserializeVerifierProvider interface {
	DeserializeVerifier(id view.Identity) (driver.Verifier, error)
}

// RawOwnerIdentityDeserializer takes as MSP identity and returns an ECDSA verifier
type RawOwnerIdentityDeserializer struct {
	DeserializeVerifierProvider
}

func NewRawOwnerIdentityDeserializer(dvp DeserializeVerifierProvider) *RawOwnerIdentityDeserializer {
	return &RawOwnerIdentityDeserializer{
		DeserializeVerifierProvider: dvp,
	}
}

func (deserializer *RawOwnerIdentityDeserializer) DeserializeVerifier(id view.Identity) (driver.Verifier, error) {
	si, err := UnmarshallRawOwner(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to RawOwner")
	}
	return deserializer.DeserializeVerifierProvider.DeserializeVerifier(si.Identity)
}

func (deserializer *RawOwnerIdentityDeserializer) DeserializeSigner(raw []byte) (driver.Signer, error) {
	return nil, errors.Errorf("signer deserialization not supported")
}

func (deserializer *RawOwnerIdentityDeserializer) Info(raw []byte, auditInfo []byte) (string, error) {
	return "info not supported", nil
}
