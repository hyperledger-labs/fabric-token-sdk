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

type setupListenerProvider struct {
	lp  fabric.SetupListenerProvider
	vkp pp.VersionKeeperProvider
}

func (p *setupListenerProvider) GetListener(tmsID token.TMSID) lookup.Listener {
	return &setupListener{
		Listener: p.lp.GetListener(tmsID),
		vk:       utils.MustGet(p.vkp.Get(tmsID)),
	}
}

type setupListener struct {
	lookup.Listener
	vk *pp.VersionKeeper
}

func (l *setupListener) OnStatus(ctx context.Context, key driver.PKey, value []byte) {
	l.Listener.OnStatus(ctx, key, value)
	l.vk.UpdateVersion()
}
