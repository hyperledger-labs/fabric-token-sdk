package db

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"go.uber.org/dig"
)

type AuditorCheckServiceProvider struct {
	tmsProvider     common2.TokenManagementServiceProvider
	networkProvider common2.NetworkProvider
	checkers        []common2.NamedChecker
}

func NewAuditorCheckServiceProvider(in struct {
	dig.In
	TMSProvider     common2.TokenManagementServiceProvider
	NetworkProvider common2.NetworkProvider
	Checkers        []common2.NamedChecker `group:"auditdb-checkers"`
}) *AuditorCheckServiceProvider {
	return &AuditorCheckServiceProvider{tmsProvider: in.TMSProvider, networkProvider: in.NetworkProvider, checkers: in.Checkers}
}

func (a *AuditorCheckServiceProvider) CheckService(id token.TMSID, adb *auditdb.DB, tdb *tokens.Tokens) (auditor.CheckService, error) {
	return common2.NewChecksService(append(common2.NewDefaultCheckers(a.tmsProvider, a.networkProvider, adb, tdb, id), a.checkers...)), nil
}

type OwnerCheckServiceProvider struct {
	tmsProvider     common2.TokenManagementServiceProvider
	networkProvider common2.NetworkProvider
	checkers        []common2.NamedChecker
}

func NewOwnerCheckServiceProvider(in struct {
	dig.In
	TMSProvider     common2.TokenManagementServiceProvider
	NetworkProvider common2.NetworkProvider
	Checkers        []common2.NamedChecker `group:"ttxdb-checkers"`
}) *OwnerCheckServiceProvider {
	return &OwnerCheckServiceProvider{tmsProvider: in.TMSProvider, networkProvider: in.NetworkProvider, checkers: in.Checkers}
}

func (a *OwnerCheckServiceProvider) CheckService(id token.TMSID, txdb *ttxdb.DB, tdb *tokens.Tokens) (ttx.CheckService, error) {
	return common2.NewChecksService(append(common2.NewDefaultCheckers(a.tmsProvider, a.networkProvider, txdb, tdb, id), a.checkers...)), nil
}
