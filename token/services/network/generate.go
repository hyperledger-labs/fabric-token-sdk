/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package network

//go:generate counterfeiter -o mocks/envelope.go -fake-name Envelope github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver.Envelope
//go:generate counterfeiter -o mocks/local_membership.go -fake-name LocalMembership github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver.LocalMembership
//go:generate counterfeiter -o mocks/ledger.go -fake-name Ledger github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver.Ledger
//go:generate counterfeiter -o mocks/network.go -fake-name Network github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver.Network
//go:generate counterfeiter -o mocks/driver.go -fake-name Driver github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver.Driver
//go:generate counterfeiter -o mocks/service_provider.go -fake-name ServiceProvider github.com/hyperledger-labs/fabric-token-sdk/token.ServiceProvider
//go:generate counterfeiter -o mocks/finality_listener.go -fake-name FinalityListener . FinalityListener
//go:generate counterfeiter -o mocks/finality_listener_manager.go -fake-name FinalityListenerManager github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver.FinalityListenerManager
//go:generate counterfeiter -o mocks/token_query_executor_provider.go -fake-name TokenQueryExecutorProvider github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver.TokenQueryExecutorProvider
//go:generate counterfeiter -o mocks/token_query_executor.go -fake-name TokenQueryExecutor github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver.TokenQueryExecutor
//go:generate counterfeiter -o mocks/spent_token_query_executor_provider.go -fake-name SpentTokenQueryExecutorProvider github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver.SpentTokenQueryExecutorProvider
//go:generate counterfeiter -o mocks/spent_token_query_executor.go -fake-name SpentTokenQueryExecutor github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver.SpentTokenQueryExecutor
