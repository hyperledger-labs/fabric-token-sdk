/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorsement

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/comm/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/endorsement/fsc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
)

const (
	FSCEndorsementKey = "services.network.fabric.fsc_endorsement"
)

var (
	logger = logging.MustGetLogger()
)

type ServiceProvider struct {
	lazy.Provider[token2.TMSID, Service]
}

func NewServiceProvider(
	fnsp *fabric.NetworkServiceProvider,
	tmsp *token2.ManagementServiceProvider,
	configService common.Configuration,
	viewManager fsc.ViewManager,
	viewRegistry fsc.ViewRegistry,
	identityProvider fsc.IdentityProvider,
	keyTranslator translator.KeyTranslator,
	storeServiceManager ttxdb.StoreServiceManager,
) *ServiceProvider {
	l := &loader{
		fnsp:                fnsp,
		tmsp:                tmsp,
		configService:       configService,
		viewManager:         viewManager,
		viewRegistry:        viewRegistry,
		identityProvider:    identityProvider,
		keyTranslator:       keyTranslator,
		storeServiceManager: storeServiceManager,
		fabricProvider:      fnsp,
	}
	return &ServiceProvider{Provider: lazy.NewProviderWithKeyMapper(key, l.load)}
}

type Service interface {
	Endorse(context view.Context, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error)
}

type loader struct {
	tmsp                *token2.ManagementServiceProvider
	fnsp                *fabric.NetworkServiceProvider
	configService       common.Configuration
	viewManager         fsc.ViewManager
	viewRegistry        fsc.ViewRegistry
	identityProvider    fsc.IdentityProvider
	keyTranslator       translator.KeyTranslator
	storeServiceManager ttxdb.StoreServiceManager
	fabricProvider      *fabric.NetworkServiceProvider
}

func (l *loader) load(tmsID token2.TMSID) (Service, error) {
	configuration, err := l.configService.ConfigurationFor(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get configuration for [%s]", tmsID)
	}

	if !configuration.IsSet(FSCEndorsementKey) {
		logger.Debugf("chaincode endorsement enabled...")
		return NewChaincodeEndorsementService(tmsID), nil
	}

	logger.Debugf("FSC endorsement enabled...")
	return fsc.NewEndorsementService(
		NewNamespaceTxProcessor(l.fnsp),
		tmsID,
		configuration,
		l.viewRegistry,
		l.viewManager,
		l.identityProvider,
		l.keyTranslator,
		func(txID string, namespace string, rws *fabric.RWSet) (fsc.Translator, error) {
			return translator.New(
				txID,
				translator.NewRWSetWrapper(&fsc.RWSWrapper{Stub: rws}, namespace, txID),
				l.keyTranslator,
			), nil
		},
		NewEndorserService(l.tmsp, l.fabricProvider),
		l.tmsp,
		NewStorageProvider(l.storeServiceManager),
	)
}

func key(tmsID token2.TMSID) string {
	return tmsID.Network + tmsID.Channel + tmsID.Namespace
}

// NamespaceTxProcessor models a namespace transaction processor for fabric
type NamespaceTxProcessor struct {
	networkServiceProvider *fabric.NetworkServiceProvider
}

// NewNamespaceTxProcessor returns a new instance of NamespaceTxProcessor
func NewNamespaceTxProcessor(networkServiceProvider *fabric.NetworkServiceProvider) *NamespaceTxProcessor {
	return &NamespaceTxProcessor{networkServiceProvider: networkServiceProvider}
}

// EnableTxProcessing signals the fabric committer to process all transactions in the network specified by the given tms id
func (n *NamespaceTxProcessor) EnableTxProcessing(tmsID token2.TMSID) error {
	nw, err := n.networkServiceProvider.FabricNetworkService(tmsID.Network)
	if err != nil {
		return errors.WithMessagef(err, "failed getting fabric network service for [%s]", tmsID.Network)
	}
	ch, err := nw.Channel(tmsID.Channel)
	if err != nil {
		return errors.Wrapf(err, "failed getting channel [%s]", tmsID.Channel)
	}
	committer := ch.Committer()
	logger.Debug("this node is an endorser, prepare it...")
	// if I'm an endorser, I need to process all token transactions
	if err := committer.ProcessNamespace(tmsID.Namespace); err != nil {
		return errors.WithMessagef(err, "failed to add namespace to committer [%s]", tmsID.Namespace)
	}

	return nil
}

// EndorserService wraps the FSC's endorser service
type EndorserService struct {
	tmsProvider    *token2.ManagementServiceProvider
	fabricProvider *fabric.NetworkServiceProvider
}

// NewEndorserService returns a new instance of EndorserService
func NewEndorserService(tmsProvider *token2.ManagementServiceProvider, fabricProvider *fabric.NetworkServiceProvider) *EndorserService {
	return &EndorserService{tmsProvider: tmsProvider, fabricProvider: fabricProvider}
}

func (e *EndorserService) ReceiveTx(ctx view.Context) (*endorser.Transaction, error) {
	_, tx, err := endorser.NewTransactionFromBytes(ctx, session.ReadFirstMessageOrPanic(ctx))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to received transaction for approval")
	}
	return tx, nil
}

func (e *EndorserService) Endorse(ctx view.Context, tx *endorser.Transaction, identities ...view.Identity) (any, error) {
	return ctx.RunView(endorser.NewEndorsementOnProposalResponderView(tx, identities...))
}

func (e *EndorserService) NewTransaction(context view.Context, opts ...fabric.TransactionOption) (*endorser.Transaction, error) {
	_, tx, err := endorser.NewTransaction(context, opts...)
	return tx, err
}

func (e *EndorserService) CollectEndorsements(ctx view.Context, tx *endorser.Transaction, timeOut time.Duration, endorsers ...view.Identity) error {
	_, err := ctx.RunView(endorser.NewParallelCollectEndorsementsOnProposalView(
		tx,
		endorsers...,
	).WithTimeout(timeOut))
	return err
}

func (e *EndorserService) EndorserID(tmsID token2.TMSID) (view.Identity, error) {
	var endorserIDLabel string
	tms, err := e.tmsProvider.GetManagementService(token2.WithTMSID(tmsID))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting management service for [%s]", tmsID)
	}
	if err := tms.Configuration().UnmarshalKey("services.network.fabric.fsc_endorsement.id", &endorserIDLabel); err != nil {
		return nil, errors.WithMessagef(err, "failed to load endorserID")
	}
	fns, err := e.fabricProvider.FabricNetworkService(tmsID.Network)
	if err != nil {
		return nil, errors.WithMessagef(err, "cannot find fabric network for [%s]", tmsID.Network)
	}

	var endorserID view.Identity
	if len(endorserIDLabel) == 0 {
		endorserID = fns.LocalMembership().DefaultIdentity()
	} else {
		var err error
		endorserID, err = fns.LocalMembership().GetIdentityByID(endorserIDLabel)
		if err != nil {
			return nil, errors.WithMessagef(err, "cannot find local endorser identity for [%s]", endorserIDLabel)
		}
	}
	if endorserID.IsNone() {
		return nil, errors.Errorf("cannot find local endorser identity for [%s]", endorserIDLabel)
	}
	if _, err := fns.SignerService().GetSigner(endorserID); err != nil {
		return nil, errors.WithMessagef(err, "cannot find fabric signer for identity [%s:%s]", endorserIDLabel, endorserID)
	}
	return endorserID, nil
}

// StorageProvider wraps ttxdb.StoreServiceManager
type StorageProvider struct {
	ttxdb.StoreServiceManager
}

// NewStorageProvider returns a new instance of StorageProvider
func NewStorageProvider(storeServiceManager ttxdb.StoreServiceManager) *StorageProvider {
	return &StorageProvider{StoreServiceManager: storeServiceManager}
}

// GetStorage returns the fsc.Storage instance for the given tms id.
func (s *StorageProvider) GetStorage(id token2.TMSID) (fsc.Storage, error) {
	return s.StoreServiceByTMSId(id)
}
