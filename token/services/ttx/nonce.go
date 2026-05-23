/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

// buildAttestationMessage constructs the message that the responder signs
// to prove key-ownership: nonce || identity.
func buildAttestationMessage(nonce []byte, identity []byte) []byte {
	msg := make([]byte, 0, len(nonce)+len(identity))
	msg = append(msg, nonce...)
	msg = append(msg, identity...)

	return msg
}
