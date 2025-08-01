/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

func DeserializeVerifier(ctx context.Context, id driver.Identity) (driver.Verifier, error) {
	genericPublicKey, err := PemDecodeKey(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed parsing received public key")
	}
	publicKey, ok := genericPublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("expected *ecdsa.PublicKey")
	}
	return NewECDSAVerifier(publicKey), nil
}

func Info(ctx context.Context, raw []byte) (string, error) {
	cert, err := PemDecodeCert(raw)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("X509: [%s][%s]", driver.Identity(raw).UniqueID(), cert.Subject.CommonName), nil
}
