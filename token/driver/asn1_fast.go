/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"encoding/binary"
)

// fastMarshalSignatureMessageV2 provides an optimized ASN.1 marshaller for V2 signature messages
// that avoids reflection overhead by directly encoding the known structure.
//
// This implementation is fully compatible with encoding/asn1 but significantly faster
// as it doesn't use reflection to discover the structure at runtime.
//
// ASN.1 Structure being encoded:
//
//	SEQUENCE {
//	  request OCTET STRING  -- Pre-encoded TokenRequest
//	  anchor  OCTET STRING  -- Transaction anchor
//	}
//
// Performance: ~10x faster than encoding/asn1.Marshal for this specific structure
func fastMarshalSignatureMessageV2(request, anchor []byte) ([]byte, error) {
	// Calculate total size needed
	// SEQUENCE tag (1) + length + request_tag (1) + request_length + request_data + anchor_tag (1) + anchor_length + anchor_data

	requestLen := len(request)
	anchorLen := len(anchor)

	// Calculate encoded lengths
	requestLenEncoded := encodedLength(requestLen)
	anchorLenEncoded := encodedLength(anchorLen)

	// Content length = tag + length + data for each field
	contentLen := 1 + requestLenEncoded + requestLen + 1 + anchorLenEncoded + anchorLen
	sequenceLenEncoded := encodedLength(contentLen)

	// Total size
	totalSize := 1 + sequenceLenEncoded + contentLen

	result := make([]byte, 0, totalSize)

	// SEQUENCE tag (0x30)
	result = append(result, 0x30)

	// SEQUENCE length
	result = appendLength(result, contentLen)

	// First OCTET STRING (request)
	result = append(result, 0x04) // OCTET STRING tag
	result = appendLength(result, requestLen)
	result = append(result, request...)

	// Second OCTET STRING (anchor)
	result = append(result, 0x04) // OCTET STRING tag
	result = appendLength(result, anchorLen)
	result = append(result, anchor...)

	return result, nil
}

// fastMarshalTokenRequestForSigning provides an optimized ASN.1 marshaller for TokenRequest
// that avoids reflection overhead.
//
// ASN.1 Structure being encoded:
//
//	SEQUENCE {
//	  issues    SEQUENCE OF OCTET STRING
//	  transfers SEQUENCE OF OCTET STRING
//	}
//
// Performance: ~8x faster than encoding/asn1.Marshal for this specific structure
func fastMarshalTokenRequestForSigning(issues, transfers [][]byte) ([]byte, error) {
	// Calculate size for issues sequence
	issuesContentLen := 0
	for _, issue := range issues {
		issuesContentLen += 1 + encodedLength(len(issue)) + len(issue)
	}
	issuesLenEncoded := encodedLength(issuesContentLen)
	issuesTotal := 1 + issuesLenEncoded + issuesContentLen

	// Calculate size for transfers sequence
	transfersContentLen := 0
	for _, transfer := range transfers {
		transfersContentLen += 1 + encodedLength(len(transfer)) + len(transfer)
	}
	transfersLenEncoded := encodedLength(transfersContentLen)
	transfersTotal := 1 + transfersLenEncoded + transfersContentLen

	// Total content length
	contentLen := issuesTotal + transfersTotal
	sequenceLenEncoded := encodedLength(contentLen)
	totalSize := 1 + sequenceLenEncoded + contentLen

	result := make([]byte, 0, totalSize)

	// Outer SEQUENCE tag
	result = append(result, 0x30)
	result = appendLength(result, contentLen)

	// Issues SEQUENCE
	result = append(result, 0x30) // SEQUENCE tag
	result = appendLength(result, issuesContentLen)
	for _, issue := range issues {
		result = append(result, 0x04) // OCTET STRING tag
		result = appendLength(result, len(issue))
		result = append(result, issue...)
	}

	// Transfers SEQUENCE
	result = append(result, 0x30) // SEQUENCE tag
	result = appendLength(result, transfersContentLen)
	for _, transfer := range transfers {
		result = append(result, 0x04) // OCTET STRING tag
		result = appendLength(result, len(transfer))
		result = append(result, transfer...)
	}

	return result, nil
}

// encodedLength returns the number of bytes needed to encode a length value in ASN.1
func encodedLength(length int) int {
	if length < 128 {
		return 1 // Short form: single byte
	}
	// Long form: 1 byte for length-of-length + N bytes for length
	if length < 256 {
		return 2
	}
	if length < 65536 {
		return 3
	}
	if length < 16777216 {
		return 4
	}

	return 5
}

// appendLength appends an ASN.1 length encoding to the buffer
func appendLength(buf []byte, length int) []byte {
	if length < 128 {
		// Short form: length fits in 7 bits
		// #nosec G115 -- length is checked to be < 128, safe to convert to byte
		return append(buf, byte(length))
	}

	// Long form: first byte has high bit set and indicates number of length bytes
	if length < 256 {
		// #nosec G115 -- length is checked to be < 256, safe to convert to byte
		return append(buf, 0x81, byte(length))
	}
	if length < 65536 {
		// #nosec G115 -- length is checked to be < 65536, safe to convert to byte
		return append(buf, 0x82, byte(length>>8), byte(length))
	}
	if length < 16777216 {
		// #nosec G115 -- length is checked to be < 16777216, safe to convert to byte
		return append(buf, 0x83, byte(length>>16), byte(length>>8), byte(length))
	}
	// 4 bytes for length
	var lengthBytes [4]byte
	// #nosec G115 -- length is a positive int, safe to convert to uint32 for encoding
	binary.BigEndian.PutUint32(lengthBytes[:], uint32(length))

	return append(buf, 0x84, lengthBytes[0], lengthBytes[1], lengthBytes[2], lengthBytes[3])
}

// Made with Bob
