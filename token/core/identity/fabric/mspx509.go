/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabric

import (
	ecdsa2 "crypto/ecdsa"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/api"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
)

// MSPX509IdentityDeserializer takes as MSP identity and returns an ECDSA verifier
type MSPX509IdentityDeserializer struct{}

func (deserializer *MSPX509IdentityDeserializer) GetVerifier(id view.Identity) (api.Verifier, error) {
	si := &msp.SerializedIdentity{}
	err := proto.Unmarshal(id, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to msp.SerializedIdentity{}")
	}
	genericPublicKey, err := PemDecodeKey(si.IdBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed parsing received public key")
	}
	publicKey, ok := genericPublicKey.(*ecdsa2.PublicKey)
	if !ok {
		return nil, errors.New("expected *ecdsa.PublicKey")
	}
	return NewVerifier(publicKey), nil
}
