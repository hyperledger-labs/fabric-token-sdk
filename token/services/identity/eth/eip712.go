/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package eth

import (
	"encoding/binary"

	"golang.org/x/crypto/sha3"
)

// Domain holds the EIP-712 domain separator fields that identify a specific
// token SDK deployment.  Every deployment should choose a unique Name +
// Version + ChainID combination so that a signature produced for one network
// cannot be replayed on another.
type Domain struct {
	// Name is a human-readable label for the signing domain, e.g. "FabricTokenSDK".
	Name string
	// Version is the domain version string, e.g. "1".
	Version string
	// ChainID is the EIP-155 chain identifier of the target EVM network.
	ChainID uint64
}

// EndorsementRequest is the typed data that co-signers sign off-chain to
// approve a pending token operation before it is submitted to the ledger.
//
// The three fields map directly to the EIP-712 type string:
//
//	EndorsementRequest(string tmsID,string txID,uint64 deadline)
//
// tmsID identifies the Token Management System (network:channel:namespace).
// txID  is the transaction identifier of the pending token request.
// deadline is a Unix timestamp after which the approval is considered void
// (use 0 to express no expiry).
type EndorsementRequest struct {
	TMSID    string
	TxID     string
	Deadline uint64
}

// endorsementTypeString is the canonical EIP-712 type string for EndorsementRequest.
const endorsementTypeString = "EndorsementRequest(string tmsID,string txID,uint64 deadline)"

// domainTypeString is the canonical EIP-712 type string for the domain separator.
const domainTypeString = "EIP712Domain(string name,string version,uint64 chainID)"

// HashEndorsementRequest returns the 32-byte EIP-712 digest for req under the
// given domain.  Pass this digest directly to Signer.Sign — the signer will
// keccak256 it once more, producing the final value that is actually signed
// (matching the Ethereum convention of always signing a hash).
//
// The computation follows EIP-712 exactly:
//
//	digest = keccak256("\x19\x01" || domainSeparator(domain) || structHash(req))
func HashEndorsementRequest(domain Domain, req EndorsementRequest) []byte {
	domainSep := hashDomain(domain)
	structHash := hashEndorsementStruct(req)

	// EIP-712 envelope: 0x19 0x01 || domainSeparator || structHash
	buf := make([]byte, 2+32+32)
	buf[0] = 0x19
	buf[1] = 0x01
	copy(buf[2:34], domainSep)
	copy(buf[34:], structHash)

	return keccak256(buf)
}

// hashDomain computes the EIP-712 domain separator for d.
func hashDomain(d Domain) []byte {
	typeHash := keccak256([]byte(domainTypeString))
	nameHash := keccak256([]byte(d.Name))
	versionHash := keccak256([]byte(d.Version))
	chainIDPadded := uint64ToBytes32(d.ChainID)

	buf := make([]byte, 4*32)
	copy(buf[0:32], typeHash)
	copy(buf[32:64], nameHash)
	copy(buf[64:96], versionHash)
	copy(buf[96:128], chainIDPadded)

	return keccak256(buf)
}

// hashEndorsementStruct computes the EIP-712 struct hash for req.
func hashEndorsementStruct(req EndorsementRequest) []byte {
	typeHash := keccak256([]byte(endorsementTypeString))
	tmsIDHash := keccak256([]byte(req.TMSID))
	txIDHash := keccak256([]byte(req.TxID))
	deadlinePadded := uint64ToBytes32(req.Deadline)

	buf := make([]byte, 4*32)
	copy(buf[0:32], typeHash)
	copy(buf[32:64], tmsIDHash)
	copy(buf[64:96], txIDHash)
	copy(buf[96:128], deadlinePadded)

	return keccak256(buf)
}

// keccak256 computes the Ethereum-compatible Keccak-256 hash of data.
// It uses the pre-standardisation variant (legacy Keccak) that Ethereum
// adopted, which differs from the NIST SHA3-256 standard.
func keccak256(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}

// uint64ToBytes32 encodes v as a 32-byte big-endian value (ABI uint256 encoding).
func uint64ToBytes32(v uint64) []byte {
	b := make([]byte, 32)
	binary.BigEndian.PutUint64(b[24:], v)
	return b
}
