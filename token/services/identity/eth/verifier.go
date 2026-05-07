/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package eth

import (
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

// Verifier checks secp256k1 ECDSA signatures produced by Signer.
// It implements driver.Verifier.
type Verifier struct {
	pubKey *secp256k1.PublicKey
}

// NewVerifier returns a Verifier that checks signatures against pubKey.
func NewVerifier(pubKey *secp256k1.PublicKey) *Verifier {
	return &Verifier{pubKey: pubKey}
}

// Verify re-hashes message with keccak256 and checks the DER-encoded
// signature against the stored public key.
func (v *Verifier) Verify(message, sigBytes []byte) error {
	if v.pubKey == nil {
		return errors.New("secp256k1 verifier: nil public key")
	}

	sig, err := ecdsa.ParseDERSignature(sigBytes)
	if err != nil {
		return errors.Wrap(err, "secp256k1 verifier: failed to parse DER signature")
	}

	digest := keccak256(message)
	if !sig.Verify(digest, v.pubKey) {
		return errors.New("secp256k1 verifier: signature is not valid")
	}

	return nil
}

// AddressFromPublicKey derives the 20-byte Ethereum address from a secp256k1
// public key using the standard Ethereum convention:
// keccak256(uncompressed_public_key_bytes[1:])[12:]
func AddressFromPublicKey(pub *secp256k1.PublicKey) [20]byte {
	uncompressed := pub.SerializeUncompressed() // 65 bytes: 0x04 || X || Y
	hash := keccak256(uncompressed[1:])         // skip 0x04 prefix, hash X||Y

	var addr [20]byte
	copy(addr[:], hash[12:]) // last 20 bytes of the 32-byte hash

	return addr
}
