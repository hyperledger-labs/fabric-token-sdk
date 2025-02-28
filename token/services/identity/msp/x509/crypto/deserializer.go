/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

func DeserializeVerifier(id driver.Identity) (driver.Verifier, error) {
	si := &SerializedIdentity{}
	err := proto.Unmarshal(id, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to SerializedIdentity{}")
	}
	genericPublicKey, err := PemDecodeKey(si.IdBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed parsing received public key")
	}
	publicKey, ok := genericPublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("expected *ecdsa.PublicKey")
	}
	return NewECDSAVerifier(publicKey), nil
}

func Info(raw []byte) (string, error) {
	si := &SerializedIdentity{}
	err := proto.Unmarshal(raw, si)
	if err != nil {
		return "", err
	}
	cert, err := PemDecodeCert(si.IdBytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("MSP.x509: [%s][%s][%s]", driver.Identity(raw).UniqueID(), si.Mspid, cert.Subject.CommonName), nil
}
