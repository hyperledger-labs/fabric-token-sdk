/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"context"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

func NewTokenExecutorProvider(viewManager *view2.Manager) *tokenFetcherProvider {
	return &tokenFetcherProvider{viewManager: viewManager}
}

type tokenFetcherProvider struct {
	viewManager *view2.Manager
}

func (p *tokenFetcherProvider) GetExecutor(network, _ string) (driver2.TokenQueryExecutor, error) {
	return &tokenFetcher{viewManager: p.viewManager, network: network}, nil
}

type tokenFetcher struct {
	network     string
	viewManager *view2.Manager
}

func (f *tokenFetcher) QueryTokens(ctx context.Context, namespace string, IDs []*token.ID) ([][]byte, error) {
	resBoxed, err := f.viewManager.InitiateView(NewRequestQueryTokensView(f.network, namespace, IDs), ctx)
	if err != nil {
		return nil, err
	}
	return resBoxed.([][]byte), nil
}

type spentTokenFetcherProvider struct {
	viewManager   *view2.Manager
	keyTranslator translator.KeyTranslator
}

func NewSpentTokenExecutorProvider(viewManager *view2.Manager, keyTranslator translator.KeyTranslator) *spentTokenFetcherProvider {
	return &spentTokenFetcherProvider{viewManager: viewManager, keyTranslator: keyTranslator}
}

func (p *spentTokenFetcherProvider) GetSpentExecutor(network, channel string) (driver2.SpentTokenQueryExecutor, error) {
	return &spentTokenFetcher{
		network:       network,
		channel:       channel,
		keyTranslator: p.keyTranslator,
		viewManager:   p.viewManager,
	}, nil
}

type spentTokenFetcher struct {
	network       string
	channel       string
	keyTranslator translator.KeyTranslator
	viewManager   *view2.Manager
}

func (f *spentTokenFetcher) QuerySpentTokens(ctx context.Context, namespace string, IDs []*token.ID, meta []string) ([]bool, error) {
	sIDs := make([]string, len(IDs))
	var err error
	for i, id := range IDs {
		sIDs[i], err = f.keyTranslator.CreateOutputKey(id.TxId, id.Index)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot compute spent id for [%v]", id)
		}
	}

	resBoxed, err := f.viewManager.InitiateView(NewRequestSpentTokensView(f.network, namespace, sIDs), ctx)
	if err != nil {
		return nil, err
	}
	return resBoxed.([]bool), nil
}
