/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemixnym

import (
	"context"
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemixnym/nym"
)

const (
	IdentityType identity.Type = "idemixnym"
)

type IdentityStoreService interface {
	// GetSignerInfo returns the signer info bound to the given identity
	GetSignerInfo(ctx context.Context, id []byte) ([]byte, error)
}

type KeyManager struct {
	backend              *idemix.KeyManager
	identityStoreService IdentityStoreService
}

func NewKeyManager(backend *idemix.KeyManager, identityStoreService IdentityStoreService) *KeyManager {
	return &KeyManager{
		backend:              backend,
		identityStoreService: identityStoreService,
	}
}

func (k *KeyManager) DeserializeVerifier(ctx context.Context, raw []byte) (driver.Verifier, error) {
	return &nym.Verifier{
		NymEID: raw,
		Backed: k.backend.Deserializer,
	}, nil
}

func (k *KeyManager) DeserializeSigner(ctx context.Context, raw []byte) (driver.Signer, error) {
	signerInfoRaw, err := k.identityStoreService.GetSignerInfo(ctx, raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve signer info")
	}
	auditInfo := &nym.AuditInfo{}
	if err := json.Unmarshal(signerInfoRaw, auditInfo); err != nil {
		return nil, errors.Wrapf(err, "failed to deserialize audit info")
	}
	auditInfoRaw, err := auditInfo.Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to deserialize audit info")
	}

	id, signer, err := nym.NewSignerProviderImpl(k, auditInfoRaw).NewSigner(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "cannot deserializer signer, cannot create signer")
	}

	return &nym.Signer{
		Creator: id,
		Signer:  signer,
	}, nil
}

func (k *KeyManager) EnrollmentID() string {
	return k.backend.EnrollmentID()
}

func (k *KeyManager) IsRemote() bool {
	return k.backend.IsRemote()
}

func (k *KeyManager) Anonymous() bool {
	return k.backend.Anonymous()
}

func (k *KeyManager) IdentityType() identity.Type {
	return IdentityType
}

func (k *KeyManager) Identity(ctx context.Context, _ []byte) (*idriver.IdentityDescriptor, error) {
	descriptor, err := k.backend.Identity(ctx, nil)
	if err != nil {
		return nil, err
	}

	// compile options and check for idemix
	ai, err := crypto.DeserializeAuditInfo(descriptor.AuditInfo)
	if err != nil {
		return nil, err
	}

	auditInfo := &nym.AuditInfo{
		AuditInfo:       ai,
		IdemixSignature: descriptor.Identity,
	}
	auditInfoRaw, err := json.Marshal(auditInfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal audit info")
	}

	signerProvider := nym.NewSignerProviderImpl(k, descriptor.AuditInfo)
	id, signer, err := signerProvider.NewSigner(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create signer")
	}

	return &idriver.IdentityDescriptor{
		// Commitment to the Enrollment ID is the new identity
		Identity:   ai.EidNymAuditData.Nym.Bytes(),
		AuditInfo:  auditInfoRaw,
		Signer:     &nym.Signer{Creator: id, Signer: signer},
		SignerInfo: auditInfoRaw,
		Verifier:   nil,
		Ephemeral:  false,
	}, nil
}
