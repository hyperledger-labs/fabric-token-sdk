/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
)

type AuditorCheckServiceProvider struct {
	tmsProvider     common.TokenManagementServiceProvider
	networkProvider common.NetworkProvider
	checkers        []common.NamedChecker
}

func NewAuditorCheckServiceProvider(tmsProvider common.TokenManagementServiceProvider, networkProvider common.NetworkProvider, checkers []common.NamedChecker) *AuditorCheckServiceProvider {
	return &AuditorCheckServiceProvider{
		tmsProvider:     tmsProvider,
		networkProvider: networkProvider,
		checkers:        checkers,
	}
}

func (a *AuditorCheckServiceProvider) CheckService(id token.TMSID, adb *auditdb.DB, tdb *tokens.Tokens) (auditor.CheckService, error) {
	return common.NewChecksService(append(common.NewDefaultCheckers(a.tmsProvider, a.networkProvider, adb, tdb, id), a.checkers...)), nil
}

type OwnerCheckServiceProvider struct {
	tmsProvider     common.TokenManagementServiceProvider
	networkProvider common.NetworkProvider
	checkers        []common.NamedChecker
}

func NewOwnerCheckServiceProvider(tmsProvider common.TokenManagementServiceProvider, networkProvider common.NetworkProvider, checkers []common.NamedChecker) *OwnerCheckServiceProvider {
	return &OwnerCheckServiceProvider{
		tmsProvider:     tmsProvider,
		networkProvider: networkProvider,
		checkers:        checkers,
	}
}

func (a *OwnerCheckServiceProvider) CheckService(id token.TMSID, txdb *ttxdb.DB, tdb *tokens.Tokens) (ttx.CheckService, error) {
	return common.NewChecksService(append(common.NewDefaultCheckers(a.tmsProvider, a.networkProvider, txdb, tdb, id), a.checkers...)), nil
}
