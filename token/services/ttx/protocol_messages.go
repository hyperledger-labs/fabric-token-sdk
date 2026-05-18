/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

// TransactionPayload carries a serialized transaction on the wire.
type TransactionPayload struct {
	Raw []byte `json:"raw"`
}

// SignaturePayload carries a signature on the wire.
type SignaturePayload struct {
	Signature []byte `json:"signature"`
}
