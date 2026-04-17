/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabricx

import (
	ffabric "github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/lookup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/qe"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"

	"go.opentelemetry.io/otel/trace"
)

// NewNetwork returns a new Network instance for the specified FabricX configuration.
// It initializes a base Fabric network and overrides its ledger with a FabricX-specific
// implementation that supports advanced state query executors.
func NewNetwork(
	storeServiceManager ttxdb.StoreServiceManager,
	auditStoreServiceManager auditdb.StoreServiceManager,
	n *ffabric.NetworkService,
	ch *ffabric.Channel,
	configuration common.Configuration,
	filterProvider common.TransactionFilterProvider[*common.AcceptTxInDBsFilter],
	tokensProvider *tokens.ServiceManager,
	viewManager fabric.ViewManager,
	tmsProvider *token.ManagementServiceProvider,
	endorsementServiceProvider fabric.EndorsementServiceProvider,
	tokenQueryExecutor driver.TokenQueryExecutor,
	tracerProvider trace.TracerProvider,
	defaultPublicParamsFetcher fabric.NetworkPublicParamsFetcher,
	spentTokenQueryExecutor driver.SpentTokenQueryExecutor,
	queryStateExecutor qe.QueryStatesExecutor,
	keyTranslator translator.KeyTranslator,
	flm finality.ListenerManager,
	llm lookup.ListenerManager,
	setupListenerProvider fabric.SetupListenerProvider,
	metricsProvider metrics.Provider,
) *fabric.Network {
	// first create a fabric network
	tn := fabric.NewNetwork(
		n,
		ch,
		configuration,
		filterProvider,
		tokensProvider,
		viewManager,
		tmsProvider,
		endorsementServiceProvider,
		tokenQueryExecutor,
		tracerProvider,
		defaultPublicParamsFetcher,
		spentTokenQueryExecutor,
		keyTranslator,
		flm,
		llm,
		setupListenerProvider,
		storeServiceManager,
		auditStoreServiceManager,
		metricsProvider,
		NewLedger(ch, n.Name(), keyTranslator, queryStateExecutor),
	)

	return tn
}
