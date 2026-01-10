/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/common"
	"go.uber.org/dig"
)

func NewAuditorCheckServiceProvider(in struct {
	dig.In
	TMSProvider     common.TokenManagementServiceProvider
	NetworkProvider common.NetworkProvider
	Checkers        []common.NamedChecker `group:"auditdb-checkers"`
}) *db.AuditorCheckServiceProvider {
	return db.NewAuditorCheckServiceProvider(in.TMSProvider, in.NetworkProvider, in.Checkers)
}

func NewOwnerCheckServiceProvider(in struct {
	dig.In
	TMSProvider     common.TokenManagementServiceProvider
	NetworkProvider common.NetworkProvider
	Checkers        []common.NamedChecker `group:"ttxdb-checkers"`
}) *db.OwnerCheckServiceProvider {
	return db.NewOwnerCheckServiceProvider(in.TMSProvider, in.NetworkProvider, in.Checkers)
}
