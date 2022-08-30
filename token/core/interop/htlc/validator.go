/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/pkg/errors"
)

type OperationType int

const (
	None OperationType = iota
	Claim
	Reclaim
)

// VerifyOwner validates the owners of the transfer in the htlc script
func VerifyOwner(senderRawOwner []byte, outRawOwner []byte, now time.Time) (*htlc.Script, OperationType, error) {
	sender, err := identity.UnmarshallRawOwner(senderRawOwner)
	if err != nil {
		return nil, None, err
	}
	script := &htlc.Script{}
	err = json.Unmarshal(sender.Identity, script)
	if err != nil {
		return nil, None, err
	}

	if now.Before(script.Deadline) {
		// this should be a claim
		if !script.Recipient.Equal(outRawOwner) {
			return nil, None, errors.New("owner of output token does not correspond to recipient in htlc request")
		}
		return script, Claim, nil
	} else {
		// this should be a reclaim
		if !script.Sender.Equal(outRawOwner) {
			return nil, None, errors.New("owner of output token does not correspond to sender in htlc request")
		}
		return script, Reclaim, nil
	}
}
