/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nym

import (
	"context"
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
)

type backedDeserializer interface {
	DeserializeAgainstNymEID(identity []byte, nymEID []byte) (*crypto.DeserializedIdentity, error)
}

type SignerEntry struct {
	Identity  []byte
	AuditInfo []byte
	Label     string
}

func (e *SignerEntry) Bytes() ([]byte, error) {
	return json.Marshal(e)
}

type KeyManager interface {
	Identity(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error)
}

type SignerProviderImpl struct {
	auditInfo  []byte
	keyManager KeyManager
}

func NewSignerProviderImpl(keyManager KeyManager, auditInfo []byte) *SignerProviderImpl {
	return &SignerProviderImpl{
		auditInfo:  auditInfo,
		keyManager: keyManager,
	}
}

func (s *SignerProviderImpl) NewSigner(ctx context.Context) ([]byte, tdriver.Signer, error) {
	id, err := s.keyManager.Identity(ctx, s.auditInfo)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot create new signer")
	}

	return id.Identity, id.Signer, nil
}
