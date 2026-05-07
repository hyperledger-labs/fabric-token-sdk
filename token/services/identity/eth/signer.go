/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package eth provides secp256k1 identity primitives for the Ethereum/EVM driver.
//
// Identities are Ethereum accounts: a 20-byte address derived from a secp256k1
// public key.  Signatures are ECDSA over secp256k1 using keccak256 as the
// pre-hash, which matches the Ethereum eth_sign and EIP-712 conventions.
//
// Endorsement approvals for off-chain co-signers use the EIP-712 typed-data
// envelope defined in eip712.go.  Callers build an EndorsementRequest, obtain
// its canonical digest via HashEndorsementRequest, then hand that digest to
// Signer.Sign.
package eth

import (
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

// Signer produces secp256k1 ECDSA signatures compatible with Ethereum's
// signing conventions.  It implements driver.Signer.
//
// Sign hashes the supplied message with keccak256 and signs the resulting
// 32-byte digest with the private key.  The returned signature is DER-encoded.
// Callers that want EIP-712 semantics should pass the output of
// HashEndorsementRequest as the message so that the final keccak256 inside
// Sign produces the correct EIP-712 digest.
type Signer struct {
	privKey *secp256k1.PrivateKey
}

// NewSigner returns a Signer backed by the given secp256k1 private key.
func NewSigner(privKey *secp256k1.PrivateKey) *Signer {
	return &Signer{privKey: privKey}
}

// Sign hashes message with keccak256 and returns a DER-encoded ECDSA signature.
func (s *Signer) Sign(message []byte) ([]byte, error) {
	if s.privKey == nil {
		return nil, errors.New("secp256k1 signer: nil private key")
	}

	digest := keccak256(message)
	sig := ecdsa.Sign(s.privKey, digest)

	return sig.Serialize(), nil
}
