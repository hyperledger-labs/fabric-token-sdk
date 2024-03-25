/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	fdriver "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	odriver "github.com/hyperledger-labs/fabric-smart-client/platform/orion/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	tokens2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk")

type PostInitializer struct {
	sp                       view.ServiceProvider
	networkProvider          *network.Provider
	tokenTransactionsManager *ttx.Manager
	auditorManager           *auditor.Manager
}

func NewPostInitializer(sp view.ServiceProvider, networkProvider *network.Provider, ownerManager *ttx.Manager, auditorManager *auditor.Manager) *PostInitializer {
	return &PostInitializer{
		sp:                       sp,
		networkProvider:          networkProvider,
		tokenTransactionsManager: ownerManager,
		auditorManager:           auditorManager,
	}
}

func (p *PostInitializer) PostInit(tms driver.TokenManagerService, networkID, channel, namespace string) error {
	tmsID := token3.TMSID{
		Network:   networkID,
		Channel:   channel,
		Namespace: namespace,
	}
	// restore owner db
	if err := p.tokenTransactionsManager.RestoreTMS(tmsID); err != nil {
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

		// fetch public params and instantiate the tms
		ons := orion.GetOrionNetworkService(p.sp, networkID)
		tmsID := token3.TMSID{Network: ons.Name(), Channel: channel, Namespace: namespace}
		logger.Debugf("register orion committer processor for [%s]", tmsID)
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
	GetTMSProvider := func() *token3.ManagementServiceProvider {
		return token3.GetManagementServiceProvider(p.sp)
	}
	GetTokenRequest := func(tms *token3.ManagementService, txID string) ([]byte, error) {
		tr, err := ttx.Get(p.sp, tms).GetTokenRequest(txID)
		if err != nil || len(tr) == 0 {
			return auditor.GetByTMSID(p.sp, tms.ID()).GetTokenRequest(txID)
		}
		return tr, nil
	}

	if err := n.ProcessorManager().AddProcessor(
		namespace,
		fabric2.NewTokenRWSetProcessor(
			n.Name(),
			namespace,
			common.NewLazyGetter[*tokens2.Tokens](func() (*tokens2.Tokens, error) {
				return tokens2.GetService(p.sp, tmsID)
			}).Get,
			GetTMSProvider,
			GetTokenRequest,
		),
	); err != nil {
		return errors.WithMessagef(err, "failed to add processor to fabric network [%s]", networkID)
	}

	ch, err := n.Channel(channel)
	if err != nil {
		return errors.WithMessagef(err, "failed to get channel [%s]", tmsID)
	}
	if err := ch.Committer().AddStatusReporter(&FabricStatusReporter{
		StatusReporter: &StatusReporter{
			tmsID:                    tmsID,
			tokenTransactionsManager: p.tokenTransactionsManager,
			auditorManager:           p.auditorManager,
		},
	}); err != nil {
		return errors.WithMessagef(err, "failed to add status reporter")
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

type StatusReporter struct {
	tmsID                    token3.TMSID
	tokenTransactionsManager *ttx.Manager
	auditorManager           *auditor.Manager
}

func (s *StatusReporter) Status(txID string) (int, string, []string, error) {
	vc, message, deps, err := s.fromTTX(txID)
	if err == nil && vc != 0 {
		return vc, message, deps, nil
	}
	return s.fromAuditor(txID)
}

func (s *StatusReporter) fromTTX(txID string) (int, string, []string, error) {
	db, err := s.tokenTransactionsManager.DB(s.tmsID)
	if err != nil {
		return 0, "", nil, err
	}
	status, message, err := db.GetStatus(txID)
	if err != nil {
		return 0, "", nil, err
	}
	return int(status), message, nil, err
}

func (s *StatusReporter) fromAuditor(txID string) (int, string, []string, error) {
	db, err := s.auditorManager.Auditor(s.tmsID)
	if err != nil {
		return 0, "", nil, err
	}
	status, message, err := db.GetStatus(txID)
	if err != nil {
		return 0, "", nil, err
	}
	return int(status), message, nil, err
}

type FabricStatusReporter struct {
	*StatusReporter
}

func (s *FabricStatusReporter) Status(txID string) (fdriver.ValidationCode, string, []string, error) {
	vc, message, deps, err := s.StatusReporter.Status(txID)
	var fabricVC fdriver.ValidationCode
	switch ttx.TxStatus(vc) {
	case ttx.Pending:
		fabricVC = fdriver.Busy
	case ttx.Confirmed:
		fabricVC = fdriver.Valid
	case ttx.Deleted:
		fabricVC = fdriver.Invalid
	case ttx.Unknown:
		fabricVC = fdriver.Unknown
	default:
		return fdriver.Unknown, "", nil, errors.Errorf("status not recognized")
	}
	return fabricVC, message, deps, err
}

type OrionStatusReporter struct {
	*StatusReporter
}

func (s *OrionStatusReporter) Status(txID string) (odriver.ValidationCode, string, []string, error) {
	vc, message, deps, err := s.StatusReporter.Status(txID)
	var orionVC odriver.ValidationCode
	switch ttx.TxStatus(vc) {
	case ttx.Pending:
		orionVC = odriver.Busy
	case ttx.Confirmed:
		orionVC = odriver.Valid
	case ttx.Deleted:
		orionVC = odriver.Invalid
	case ttx.Unknown:
		orionVC = odriver.Unknown
	default:
		return odriver.Unknown, "", nil, errors.Errorf("status not recognized")
	}
	return orionVC, message, deps, err
}
