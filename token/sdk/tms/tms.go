/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tms

import (
	"os"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	network2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/processor"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk")

type PostInitializer struct {
	sp view.ServiceProvider
}

func NewPostInitializer(sp view.ServiceProvider) *PostInitializer {
	return &PostInitializer{sp: sp}
}

func (p *PostInitializer) PostInit(tms driver.TokenManagerService, networkID, channel, namespace string) error {
	// Check public params
	cPP := tms.ConfigManager().TMS().PublicParameters
	if cPP != nil && len(cPP.Path) != 0 {
		logger.Infof("load public parameters from [%s]...", cPP.Path)
		if tms.PublicParamsManager().PublicParameters() != nil {
			logger.Infof("public parameters already available, skip loading from [%s]...", cPP.Path)
		} else {
			ppRaw, err := os.ReadFile(cPP.Path)
			if err != nil {
				return errors.Errorf("failed to load public parameters from [%s]: [%s]", cPP.Path, err)
			}
			if err := tms.PublicParamsManager().SetPublicParameters(ppRaw); err != nil {
				return errors.WithMessagef(err, "failed to set public params for [%s:%s:%s]", networkID, channel, namespace)
			}
			logger.Infof("load public parameters from [%s] done", cPP.Path)
		}
	}

	n := fabric.GetFabricNetworkService(p.sp, networkID)
	if n == nil && orion.GetOrionNetworkService(p.sp, networkID) != nil {
		// register processor
		ons := orion.GetOrionNetworkService(p.sp, networkID)
		tokenStore, err := processor.NewCommonTokenStore(p.sp, token3.TMSID{
			Network:   ons.Name(),
			Channel:   channel,
			Namespace: namespace,
		})
		if err != nil {
			return errors.WithMessagef(err, "failed to get token store")
		}
		if err := ons.ProcessorManager().AddProcessor(
			namespace,
			orion2.NewTokenRWSetProcessor(
				ons,
				namespace,
				p.sp,
				network2.NewAuthorizationMultiplexer(&network2.TMSAuthorization{}, &htlc.ScriptOwnership{}, &pledge.ScriptOwnership{}),
				network2.NewIssuedMultiplexer(&network2.WalletIssued{}),
				tokenStore,
			),
		); err != nil {
			return errors.WithMessagef(err, "failed to add processor to orion network [%s]", networkID)
		}
		// fetch public params
		nw := network.GetInstance(p.sp, networkID, channel)
		ppRaw, err := nw.FetchPublicParameters(namespace)
		if err != nil {
			return errors.WithMessagef(err, "failed to fetch public parameters for [%s:%s:%s]", networkID, channel, namespace)
		}
		if err := tms.PublicParamsManager().SetPublicParameters(ppRaw); err != nil {
			return errors.WithMessagef(err, "failed to set public params for [%s:%s:%s]", networkID, channel, namespace)
		}
		return nil
	}

	// register processor
	tokenStore, err := processor.NewCommonTokenStore(p.sp, token3.TMSID{
		Network:   n.Name(),
		Channel:   channel,
		Namespace: namespace,
	})
	if err != nil {
		return errors.WithMessagef(err, "failed to get token store")
	}
	if err := n.ProcessorManager().AddProcessor(
		namespace,
		fabric2.NewTokenRWSetProcessor(
			n,
			namespace,
			p.sp,
			network2.NewAuthorizationMultiplexer(&network2.TMSAuthorization{}, &htlc.ScriptOwnership{}, &pledge.ScriptOwnership{}),
			network2.NewIssuedMultiplexer(&network2.WalletIssued{}),
			tokenStore,
		),
	); err != nil {
		return errors.WithMessagef(err, "failed to add processor to fabric network [%s]", networkID)
	}
	return nil
}
