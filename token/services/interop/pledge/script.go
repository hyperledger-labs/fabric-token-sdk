/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"
)

const ScriptType = "pledge" // pledge script

type Script struct {
	Sender             view.Identity
	Recipient          view.Identity
	DestinationNetwork string
	Deadline           time.Time
	Issuer             view.Identity
	ID                 string
}

// Validate checks that all fields of pledge script are correctly set
func (s *Script) Validate() error {
	if s.Sender.IsNone() {
		return errors.New("invalid pledge script: empty sender")
	}
	if s.Recipient.IsNone() {
		return errors.New("invalid pledge script: empty recipient")
	}
	if s.DestinationNetwork == "" {
		return errors.New("invalid pledge script: empty destination network")
	}
	if s.Deadline.Before(time.Now()) {
		return errors.New("invalid pledge script: deadline already elapsed")
	}
	if s.Issuer.IsNone() {
		return errors.New("invalid pledge script: empty issuer")
	}
	if s.ID == "" {
		return errors.New("invalid pledge script: empty identifier")
	}
	return nil
}
