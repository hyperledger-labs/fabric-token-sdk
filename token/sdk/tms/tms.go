/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tms

import (
	"context"

	token3 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	tokens2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/pkg/errors"
)

type PostInitializer struct {
	tokensProvider *tokens2.ServiceManager

	networkProvider *network.Provider
	ownerManager    *ttx.ServiceManager
	auditorManager  *auditor.ServiceManager
}

func NewPostInitializer(tokensProvider *tokens2.ServiceManager, networkProvider *network.Provider, ownerManager *ttx.ServiceManager, auditorManager *auditor.ServiceManager) (*PostInitializer, error) {
	return &PostInitializer{
		tokensProvider:  tokensProvider,
		networkProvider: networkProvider,
		ownerManager:    ownerManager,
		auditorManager:  auditorManager,
	}, nil
}

func (p *PostInitializer) PostInit(tms driver.TokenManagerService, networkID, channel, namespace string) error {
	tmsID := token3.TMSID{
		Network:   networkID,
		Channel:   channel,
		Namespace: namespace,
	}

	// restore owner db
	if err := p.ownerManager.RestoreTMS(context.Background(), tmsID); err != nil {
		return errors.WithMessagef(err, "failed to restore onwer dbs for [%s]", tmsID)
	}

	// restore auditor db
	if err := p.auditorManager.RestoreTMS(context.Background(), tmsID); err != nil {
		return errors.WithMessagef(err, "failed to restore auditor dbs for [%s]", tmsID)
	}

	// set supported tokens
	tokens, err := p.tokensProvider.ServiceByTMSId(tmsID)
	if err != nil {
		return errors.WithMessagef(err, "failed to get tokens for [%s]", tmsID)
	}
	supportedTokens := tms.TokensService().SupportedTokenFormats()
	if err := tokens.SetSupportedTokenFormats(supportedTokens); err != nil {
		return errors.WithMessagef(err, "failed to set supported tokens for [%s] to [%s]", tmsID, supportedTokens)
	}

	return nil
}
