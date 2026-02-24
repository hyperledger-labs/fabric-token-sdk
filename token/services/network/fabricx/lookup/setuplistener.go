/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package lookup

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/lookup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/pp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
)

// NewSetupListenerProvider returns a new setupListenerProvider instance.
func NewSetupListenerProvider(
	tmsProvider *token.ManagementServiceProvider,
	tokensProvider *tokens.ServiceManager,
	versionKeeperProvider pp.VersionKeeperProvider,
) *setupListenerProvider {
	return &setupListenerProvider{
		lp:  fabric.NewSetupListenerProvider(tmsProvider, tokensProvider),
		vkp: versionKeeperProvider,
	}
}

// setupListenerProvider models a provider for setup listeners.
type setupListenerProvider struct {
	lp  fabric.SetupListenerProvider
	vkp pp.VersionKeeperProvider
}

// GetListener returns a new listener for the given TMS ID.
// The listener will update the version keeper when a status change is notified.
func (p *setupListenerProvider) GetListener(tmsID token.TMSID) lookup.Listener {
	return &setupListener{
		Listener: p.lp.GetListener(tmsID),
		vk:       utils.MustGet(p.vkp.Get(tmsID)),
	}
}

// setupListener models a setup listener that updates a version keeper.
type setupListener struct {
	lookup.Listener
	vk *pp.VersionKeeper
}

// OnStatus notifies the listener of a status change.
// It also updates the version keeper.
func (l *setupListener) OnStatus(ctx context.Context, key driver.PKey, value []byte) {
	l.Listener.OnStatus(ctx, key, value)
	l.vk.UpdateVersion()
}
