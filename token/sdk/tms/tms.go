/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tms

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	network2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/processor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/mailman"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk")

type PostInitializer struct {
	sp              view.ServiceProvider
	networkProvider *network.Provider
	ownerManager    *owner.Manager
	auditorManager  *auditor.Manager
}

func NewPostInitializer(sp view.ServiceProvider, networkProvider *network.Provider, ownerManager *owner.Manager, auditorManager *auditor.Manager) *PostInitializer {
	return &PostInitializer{
		sp:              sp,
		networkProvider: networkProvider,
		ownerManager:    ownerManager,
		auditorManager:  auditorManager,
	}
}

func (p *PostInitializer) PostInit(tms driver.TokenManagerService, networkID, channel, namespace string) error {
	tmsID := token3.TMSID{
		Network:   networkID,
		Channel:   channel,
		Namespace: namespace,
	}
	// restore owner db
	if err := p.ownerManager.RestoreTMS(tmsID); err != nil {
		return errors.WithMessagef(err, "failed to restore onwer dbs for [%s]", tmsID)
	}
	// restore auditor db
	if err := p.auditorManager.RestoreTMS(tmsID); err != nil {
		return errors.WithMessagef(err, "failed to restore auditor dbs for [%s]", tmsID)
	}
	return nil
}

func (p *PostInitializer) ConnectNetwork(networkID, channel, namespace string) error {
	n := fabric.GetFabricNetworkService(p.sp, networkID)
	if n == nil && orion.GetOrionNetworkService(p.sp, networkID) != nil {
		// ORION

		// register processor
		ons := orion.GetOrionNetworkService(p.sp, networkID)
		tmsID := token3.TMSID{
			Network:   ons.Name(),
			Channel:   channel,
			Namespace: namespace,
		}
		logger.Debugf("register orion committer processor for [%s]", tmsID)
		tokenStore, err := processor.NewCommonTokenStore(p.sp, tmsID)
		if err != nil {
			return errors.WithMessagef(err, "failed to get token store")
		}
		if err := ons.ProcessorManager().AddProcessor(
			namespace,
			orion2.NewTokenRWSetProcessor(
				ons,
				namespace,
				p.sp,
				network2.NewAuthorizationMultiplexer(&network2.TMSAuthorization{}, &htlc.ScriptOwnership{}),
				network2.NewIssuedMultiplexer(&network2.WalletIssued{}),
				tokenStore,
			),
		); err != nil {
			return errors.WithMessagef(err, "failed to add processor to orion network [%s]", tmsID)
		}

		// fetch public params and instantiate the tms
		nw := network.GetInstance(p.sp, networkID, channel)
		ppRaw, err := nw.FetchPublicParameters(namespace)
		if err != nil {
			return errors.WithMessagef(err, "failed to fetch public parameters for [%s]", tmsID)
		}
		_, err = token3.GetManagementServiceProvider(p.sp).GetManagementService(token3.WithTMSID(tmsID), token3.WithPublicParameter(ppRaw))
		if err != nil {
			return errors.WithMessagef(err, "failed to instantiate tms [%s]", tmsID)
		}
		return nil
	}

	// FABRIC

	// register commit pipeline processor
	logger.Debugf("register fabric committer processor for [%s:%s:%s]", networkID, channel, namespace)
	tmsID := token3.TMSID{
		Network:   n.Name(),
		Channel:   channel,
		Namespace: namespace,
	}
	tokenStore, err := processor.NewCommonTokenStore(p.sp, tmsID)
	if err != nil {
		return errors.WithMessagef(err, "failed to get token store")
	}
	if err := n.ProcessorManager().AddProcessor(
		namespace,
		fabric2.NewTokenRWSetProcessor(
			n,
			namespace,
			p.sp,
			network2.NewAuthorizationMultiplexer(&network2.TMSAuthorization{}, &htlc.ScriptOwnership{}),
			network2.NewIssuedMultiplexer(&network2.WalletIssued{}),
			tokenStore,
		),
	); err != nil {
		return errors.WithMessagef(err, "failed to add processor to fabric network [%s]", networkID)
	}

	// check the vault for public parameters,
	// use them if they exists
	net, err := p.networkProvider.GetNetwork(networkID, channel)
	if err != nil {
		return errors.WithMessagef(err, "cannot find network at [%s]", tmsID)
	}
	v, err := net.Vault(namespace)
	if err != nil {
		return errors.WithMessagef(err, "failed to get network at [%s]", tmsID)
	}
	ppRaw, err := v.QueryEngine().PublicParams()
	if err != nil {
		return errors.WithMessagef(err, "failed to get public params at [%s]", tmsID)
	}
	if len(ppRaw) != 0 {
		// initialize the TMS with the public params from the vault
		_, err := token3.GetManagementServiceProvider(p.sp).GetManagementService(token3.WithTMSID(tmsID), token3.WithPublicParameter(ppRaw))
		if err != nil {
			return errors.WithMessagef(err, "failed to instantiate tms [%s]", tmsID)
		}
	}
	return nil
}

type NetworkProvider interface {
	GetNetwork(network string, channel string) (*network.Network, error)
}

type VaultProvider struct {
	np NetworkProvider
}

func NewVaultProvider(np NetworkProvider) *VaultProvider {
	return &VaultProvider{np: np}
}

func (v *VaultProvider) Vault(tms *token3.ManagementService) (mailman.Vault, mailman.QueryService, error) {
	net, err := v.np.GetNetwork(tms.Network(), tms.Channel())
	if err != nil {
		return nil, nil, errors.Errorf("cannot get network for TMS [%s]", tms.ID())
	}
	vault, err := net.Vault(tms.Namespace())
	if err != nil {
		return nil, nil, errors.Errorf("cannot get network vault for TMS [%s]", tms.ID())
	}
	return vault, tms.Vault().NewQueryEngine(), nil
}
