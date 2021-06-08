/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package api

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

// this what is used to compute the hash; party and counterparty will end up with the same hash
type Transfer struct {
	Sender   string
	Receiver string
	Type     string
	Value    uint64
}

type Channel interface {
	Receive(id, ttype string, value uint64, sig []byte) error
	Send(id, ttype string, value uint64) error
	Net() ([]*Transfer, error)
}

type Tracker interface {
	OpenChannelTo(id string, recipient view.Identity) (Channel, error)

	Channel(id string) (Channel, error)
}

type TrackerService interface {
	Tracker(id view.Identity) (Tracker, error)
}
