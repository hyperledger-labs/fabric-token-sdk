/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interop

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/pkg/errors"
)

// VerifyTransferFromHTLCScript validates the owners of the transfer in the htlc script
func VerifyTransferFromHTLCScript(senderRawOwner []byte, outRawOwner []byte) error {
	sender, err := identity.UnmarshallRawOwner(senderRawOwner)
	if err != nil {
		return err
	}
	script := &htlc.Script{}
	err = json.Unmarshal(sender.Identity, script)
	if err != nil {
		return err
	}

	if time.Now().Before(script.Deadline) {
		// this should be a claim
		if !script.Recipient.Equal(outRawOwner) {
			return errors.Errorf("owner of output token does not correspond to recipient in htlc request")
		}
	} else {
		// this should be a reclaim
		if !script.Sender.Equal(outRawOwner) {
			return errors.Errorf("owner of output token does not correspond to sender in htlc request")
		}
	}

	return nil
}
