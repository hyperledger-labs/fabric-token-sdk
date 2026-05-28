/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

// ClaimSignature carries the preimage used to unlock a hash escrow script.
type ClaimSignature struct {
	Preimage []byte
}

// ClaimSigner embeds the preimage in the signature payload.
// No claimant signature is required: anyone can submit if they know a valid preimage.
type ClaimSigner struct {
	Preimage []byte
}

// Sign returns a serialized claim signature containing the preimage.
func (cs *ClaimSigner) Sign(tokenRequestAndTxID []byte) ([]byte, error) {
	claimSignature := ClaimSignature{
		Preimage: cs.Preimage,
	}

	return json.Marshal(claimSignature)
}

// ClaimVerifier checks that the preimage unlocks one of the two script hashes.
type ClaimVerifier struct {
	Script *Script
}

func (cv *ClaimVerifier) Verify(tokenRequestAndTxID, claimSignature []byte) error {
	_ = tokenRequestAndTxID
	sig := &ClaimSignature{}
	err := json.Unmarshal(claimSignature, sig)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal claim signature")
	}

	if len(sig.Preimage) == 0 {
		return errors.New("invalid claim signature, empty preimage")
	}
	if cv.Script == nil {
		return errors.New("invalid claim verifier, script is nil")
	}
	if _, _, _, err = cv.Script.ResolveOwnerAndHashForPreimage(sig.Preimage); err != nil {
		return errors.WithMessage(err, "preimage does not unlock hash escrow script")
	}

	return nil
}

// Verifier validates claims for hash-based escrow scripts.
// A valid claim carries a preimage that resolves to either recipient or sender.
type Verifier struct {
	Script *Script
}

func (v *Verifier) Verify(msg []byte, sigma []byte) error {
	claimVerifier := &ClaimVerifier{Script: v.Script}
	if err := claimVerifier.Verify(msg, sigma); err != nil {
		return errors.WithMessage(err, "failed verifying hash escrow claim signature")
	}

	return nil
}
