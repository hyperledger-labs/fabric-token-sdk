/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tms

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	tokens2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk")

type PostInitializer struct {
	tmsProvider    *token3.ManagementServiceProvider
	fabricNSP      *fabric.NetworkServiceProvider
	orionNSP       *orion.NetworkServiceProvider
	tokensProvider *tokens2.Manager
	filterProvider TransactionFilterProvider[*AcceptTxInDBsFilter]

	networkProvider *network.Provider
	ownerManager    *ttx.Manager
	auditorManager  *auditor.Manager
}

func NewPostInitializer(tmsProvider *token3.ManagementServiceProvider, fabricNSP *fabric.NetworkServiceProvider, orionNSP *orion.NetworkServiceProvider, tokensProvider *tokens2.Manager, auditDBProvider *auditdb.Manager, ttxDBProvider *ttxdb.Manager, networkProvider *network.Provider, ownerManager *ttx.Manager, auditorManager *auditor.Manager) (*PostInitializer, error) {
	return &PostInitializer{
		tmsProvider:     tmsProvider,
		tokensProvider:  tokensProvider,
		networkProvider: networkProvider,
		ownerManager:    ownerManager,
		auditorManager:  auditorManager,
		filterProvider: &acceptTxInDBFilterProvider{
			auditDBProvider: auditDBProvider,
			ttxDBProvider:   ttxDBProvider,
		},
		fabricNSP: fabricNSP,
		orionNSP:  orionNSP,
	}, nil
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

func (p *PostInitializer) fabricNetworkService(id string) (*fabric.NetworkService, error) {
	if p.fabricNSP == nil {
		return nil, errors.New("fabric nsp not found")
	}
	return p.fabricNSP.FabricNetworkService(id)
}

func (p *PostInitializer) orionNetworkService(id string) (*orion.NetworkService, error) {
	if p.orionNSP == nil {
		return nil, errors.New("orion nsp not found")
	}
	return p.orionNSP.NetworkService(id)
}

func (p *PostInitializer) ConnectNetwork(networkID, channel, namespace string) error {
	GetTMSProvider := func() *token3.ManagementServiceProvider {
		return p.tmsProvider
	}
	GetTokenRequest := func(tms *token3.ManagementService, txID string) ([]byte, error) {
		if ownerDB, err := p.ownerManager.DB(tms.ID()); err == nil {
			if tr, err := ownerDB.GetTokenRequest(txID); len(tr) != 0 && err == nil {
				return tr, nil
			}
		}
		if auditorDB, err := p.auditorManager.Auditor(tms.ID()); err == nil {
			return auditorDB.GetTokenRequest(txID)
		}
		return nil, errors.New("failed to get auditor manager")
	}
	n, err := p.fabricNetworkService(networkID)
	if err != nil {
		// ORION

		// register processor
		ons, err := p.orionNetworkService(networkID)
		if err != nil {
			return err
		}
		tmsID := token3.TMSID{
			Network:   ons.Name(),
			Channel:   channel,
			Namespace: namespace,
		}
		transactionFilter, err := p.filterProvider.New(tmsID)
		if err != nil {
			return errors.WithMessagef(err, "failed to create transaction filter for [%s]", tmsID)
		}
		if err := ons.Committer().AddTransactionFilter(transactionFilter); err != nil {
			return errors.WithMessagef(err, "failed to fetch attach transaction filter [%s]", tmsID)
		}

		// fetch public params and instantiate the tms
		nw, err := p.networkProvider.GetNetwork(networkID, channel)
		if err != nil {
			return errors.WithMessagef(err, "failed to get network")
		}
		ppRaw, err := nw.FetchPublicParameters(namespace)
		if err != nil {
			return errors.WithMessagef(err, "failed to fetch public parameters for [%s]", tmsID)
		}
		_, err = p.tmsProvider.GetManagementService(token3.WithTMSID(tmsID), token3.WithPublicParameter(ppRaw))
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
	if err := n.ProcessorManager().AddProcessor(
		namespace,
		fabric2.NewTokenRWSetProcessor(
			n.Name(),
			namespace,
			common.NewLazyGetter[*tokens2.Tokens](func() (*tokens2.Tokens, error) {
				return p.tokensProvider.Tokens(tmsID)
			}).Get,
			GetTMSProvider,
			GetTokenRequest,
		),
	); err != nil {
		return errors.WithMessagef(err, "failed to add processor to fabric network [%s]", networkID)
	}
	transactionFilter, err := p.filterProvider.New(tmsID)
	if err != nil {
		return errors.WithMessagef(err, "failed to create transaction filter for [%s]", tmsID)
	}
	ch, err := n.Channel(channel)
	if err != nil {
		return errors.WithMessagef(err, "failed to get channel for [%s]", tmsID)
	}
	if err := ch.Committer().AddTransactionFilter(transactionFilter); err != nil {
		return errors.WithMessagef(err, "failed to fetch attach transaction filter [%s]", tmsID)
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
		_, err := p.tmsProvider.GetManagementService(token3.WithTMSID(tmsID), token3.WithPublicParameter(ppRaw))
		if err != nil {
			return errors.WithMessagef(err, "failed to instantiate tms [%s]", tmsID)
		}
	}
	return nil
}

type TransactionFilterProvider[F driver2.TransactionFilter] interface {
	New(tmsID token3.TMSID) (F, error)
}

type acceptTxInDBFilterProvider struct {
	ttxDBProvider   *ttxdb.Manager
	auditDBProvider *auditdb.Manager
}

func (p *acceptTxInDBFilterProvider) New(tmsID token3.TMSID) (*AcceptTxInDBsFilter, error) {
	ttxDB, err := p.ttxDBProvider.DBByTMSId(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get transaction db for [%s]", tmsID)
	}
	auditDB, err := p.auditDBProvider.DBByTMSId(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get audit db for [%s]", tmsID)
	}
	return &AcceptTxInDBsFilter{
		ttxDB:   ttxDB,
		auditDB: auditDB,
	}, nil
}

type AcceptTxInDBsFilter struct {
	ttxDB   *ttxdb.DB
	auditDB *auditdb.DB
}

func (t *AcceptTxInDBsFilter) Accept(txID string, env []byte) (bool, error) {
	status, _, err := t.ttxDB.GetStatus(txID)
	if err != nil {
		return false, err
	}
	if status != ttxdb.Unknown {
		return true, nil
	}
	status, _, err = t.auditDB.GetStatus(txID)
	if err != nil {
		return false, err
	}
	return status != auditdb.Unknown, nil
}
