/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interop

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/exchange"
	"github.com/pkg/errors"
)

// VerifyTransferFromExchangeScript validates the owners of the transfer in the exchange script
func VerifyTransferFromExchangeScript(senderRawOwner []byte, outRawOwner []byte) error {
	sender, err := identity.UnmarshallRawOwner(senderRawOwner)
	if err != nil {
		return err
	}
	script := &exchange.Script{}
	err = json.Unmarshal(sender.Identity, script)
	if err != nil {
		return err
	}

	if time.Now().Before(script.Deadline) {
		// this should be a claim
		if !script.Recipient.Equal(outRawOwner) {
			return errors.Errorf("owner of output token does not correspond to recipient in exchange request")
		}
	} else {
		// this should be a reclaim
		if !script.Sender.Equal(outRawOwner) {
			return errors.Errorf("owner of output token does not correspond to sender in exchange request")
		}
	}

	return nil
}
