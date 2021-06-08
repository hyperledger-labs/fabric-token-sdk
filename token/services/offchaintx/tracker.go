/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package offchaintx

import (
	"reflect"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/offchaintx/api"
	"github.com/pkg/errors"
)

type channel struct {
	ch api.Channel
}

func (c *channel) Receive(id, ttype string, value uint64, sig []byte) error {
	return c.ch.Receive(id, ttype, value, sig)
}

func (c *channel) Send(id, ttype string, value uint64) error {
	return c.ch.Send(id, ttype, value)
}

func (c *channel) Net() ([]*api.Transfer, error) {
	return c.ch.Net()
}

func OpenChannelTo(sp view2.ServiceProvider, me view.Identity, id string, recipient view.Identity) (*channel, error) {
	tracker, err := getTrackerService(sp).Tracker(me)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting tracker")
	}
	ch, err := tracker.OpenChannelTo(id, recipient)
	if err != nil {
		return nil, errors.WithMessage(err, "failed opening channel to")
	}
	return &channel{ch: ch}, nil
}

func GetChannel(sp view2.ServiceProvider, me view.Identity, id string) (*channel, error) {
	tracker, err := getTrackerService(sp).Tracker(me)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting tracker")
	}
	ch, err := tracker.Channel(id)
	if err != nil {
		return nil, errors.WithMessage(err, "failed opening channel to")
	}
	return &channel{ch: ch}, nil
}

func getTrackerService(sp view2.ServiceProvider) api.TrackerService {
	s, err := sp.GetService(reflect.TypeOf((*api.TrackerService)(nil)))
	if err != nil {
		panic(err)
	}
	return s.(api.TrackerService)
}
