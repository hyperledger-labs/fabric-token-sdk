/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tms

import (
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	tokens2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/pkg/errors"
)

type PostInitializer struct {
	tmsProvider    *token3.ManagementServiceProvider
	tokensProvider *tokens2.Manager

	networkProvider *network.Provider
	ownerManager    *ttx.Manager
	auditorManager  *auditor.Manager
}

func NewPostInitializer(tmsProvider *token3.ManagementServiceProvider, tokensProvider *tokens2.Manager, networkProvider *network.Provider, ownerManager *ttx.Manager, auditorManager *auditor.Manager) (*PostInitializer, error) {
	return &PostInitializer{
		tmsProvider:     tmsProvider,
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
	if err := p.ownerManager.RestoreTMS(tmsID); err != nil {
		return errors.WithMessagef(err, "failed to restore onwer dbs for [%s]", tmsID)
	}
	// restore auditor db
	if err := p.auditorManager.RestoreTMS(tmsID); err != nil {
		return errors.WithMessagef(err, "failed to restore auditor dbs for [%s]", tmsID)
	}
	return nil
}
