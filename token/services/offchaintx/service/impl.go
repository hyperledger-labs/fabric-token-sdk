/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package service

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/offchaintx/api"
)

type trackerService struct{}

func (t *trackerService) Tracker(id view.Identity) (api.Tracker, error) {
	panic("implement me")
}

type tracker struct {
	channels []*channel
}

func (t *tracker) OpenChannelTo(id string, recipient view.Identity) (api.Channel, error) {
	panic("implement me")
}

func (t *tracker) Channel(id string) (api.Channel, error) {
	panic("implement me")
}

type channel struct {
	ID           string
	CounterParty view.Identity
}

func (c *channel) Receive(id, ttype string, value uint64, sig []byte) error {
	panic("implement me")
}

func (c *channel) Send(id, ttype string, value uint64) error {
	panic("implement me")
}

func (c *channel) Net() ([]*api.Transfer, error) {
	panic("implement me")
}

func (c *channel) store() error {
	panic("implement me")
}
