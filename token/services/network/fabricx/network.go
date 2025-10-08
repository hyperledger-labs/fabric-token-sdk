/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabricx

import (
	ffabric "github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/lookup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/qe"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"

	"go.opentelemetry.io/otel/trace"
)

type QueryStatesExecutor interface {
}

type Network struct {
	*fabric.Network
	ledger driver.Ledger
}

func NewNetwork(
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
) *Network {
	// first create a fabric network
	tn := fabric.NewNetwork(n, ch, configuration, filterProvider, tokensProvider, viewManager, tmsProvider, endorsementServiceProvider, tokenQueryExecutor, tracerProvider, defaultPublicParamsFetcher, spentTokenQueryExecutor, keyTranslator, flm, llm, setupListenerProvider)

	// we override the ledger created by fabric.NewNetwork with our fabricx specific impl
	l := NewLedger(ch, keyTranslator, queryStateExecutor)

	return &Network{Network: tn, ledger: l}
}

func (n *Network) Ledger() (driver.Ledger, error) {
	return n.ledger, nil
}
