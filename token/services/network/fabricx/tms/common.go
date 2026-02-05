/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tms

import (
	"context"
	"io"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/msp"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
)

type Signer interface {
	Sign(msg []byte) ([]byte, error)
	Serialize() ([]byte, error)
}

type SigningIdentityProvider interface {
	DefaultSigningIdentity(network, channel string) (Signer, error)
	DefaultIdentity(network, channel string) (view.Identity, error)
}

type EnvelopeBroadcaster interface {
	Broadcast(network, channel string, txID driver.TxID, env *cb.Envelope) error
}

type fnsSigningIdentityProvider struct {
	fnsProvider *fabric.NetworkServiceProvider
}

type signerWrapper struct {
	signerWithPublicVersion
}

func (w *signerWrapper) Serialize() ([]byte, error) {
	return w.GetPublicVersion().Serialize()
}

type signerWithPublicVersion interface {
	Signer
	GetPublicVersion() msp.Identity
}

func (p *fnsSigningIdentityProvider) DefaultSigningIdentity(network, channel string) (Signer, error) {
	fns, err := p.fnsProvider.FabricNetworkService(network)
	if err != nil {
		return nil, errors.Wrapf(err, "fns for [%s] not found", network)
	}

	return &signerWrapper{
		signerWithPublicVersion: fns.LocalMembership().DefaultSigningIdentity().(signerWithPublicVersion),
	}, nil
}

func (p *fnsSigningIdentityProvider) DefaultIdentity(network, channel string) (view.Identity, error) {
	fns, err := p.fnsProvider.FabricNetworkService(network)
	if err != nil {
		return nil, errors.Wrapf(err, "fns for [%s] not found", network)
	}
	return fns.LocalMembership().DefaultIdentity(), nil
}

type fnsBroadcaster struct {
	fnsProvider *fabric.NetworkServiceProvider
}

func (p *fnsBroadcaster) Broadcast(network, channel string, txID driver.TxID, env *cb.Envelope) error {
	fns, err := p.fnsProvider.FabricNetworkService(network)
	if err != nil {
		return errors.Wrapf(err, "fns for [%s] not found", network)
	}

	ch, err := fns.Channel(channel)
	if err != nil {
		return errors.Wrapf(err, "failed to get channel [%s]", channel)
	}

	final := make(chan error)
	go func() { final <- getFinality(ch.Finality(), txID) }()

	logger.Infof("Send transaction [txID=%v] for ordering", txID)
	if err := fns.Ordering().Broadcast(context.Background(), env); err != nil {
		return errors.Wrapf(err, "failed broadcasting on [%s,%s]", network, channel)
	}

	logger.Infof("Wait for finality")
	return <-final
}

func getFinality(finality driver.Finality, txID string) error {
	logger.Infof("wait for finality [txID=%v]", txID)
	for range finalityEOFRetries {
		err := finality.IsFinal(context.Background(), txID)
		if err == nil {
			return nil
		}
		if !errors.Is(err, io.EOF) {
			return errors.Wrapf(err, "finality failed")
		}
		logger.Warnf("EOF returned. Maybe block 0 is not yet committed. Sleep and retry...")
		time.Sleep(finalityRetryDuration)
	}
	return errors.New("max retries reached")
}
