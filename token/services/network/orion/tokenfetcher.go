/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

func NewTokenExecutorProvider() *tokenFetcherProvider {
	return &tokenFetcherProvider{}
}

type tokenFetcherProvider struct{}

func (p *tokenFetcherProvider) GetExecutor(network, _ string) (driver2.TokenQueryExecutor, error) {
	return &tokenFetcher{network: network}, nil
}

type tokenFetcher struct {
	network string
}

func (f *tokenFetcher) QueryTokens(context view.Context, namespace string, IDs []*token.ID) ([][]byte, error) {
	resBoxed, err := view2.GetManager(context).InitiateView(NewRequestQueryTokensView(f.network, namespace, IDs), context.Context())
	if err != nil {
		return nil, err
	}
	return resBoxed.([][]byte), nil
}

type spentTokenFetcherProvider struct {
	keyTranslator translator.KeyTranslator
}

func NewSpentTokenExecutorProvider(keyTranslator translator.KeyTranslator) *spentTokenFetcherProvider {
	return &spentTokenFetcherProvider{keyTranslator: keyTranslator}
}

func (p *spentTokenFetcherProvider) GetSpentExecutor(network, channel string) (driver2.SpentTokenQueryExecutor, error) {
	return &spentTokenFetcher{
		network:       network,
		channel:       channel,
		keyTranslator: p.keyTranslator,
	}, nil
}

type spentTokenFetcher struct {
	network       string
	channel       string
	keyTranslator translator.KeyTranslator
}

func (f *spentTokenFetcher) QuerySpentTokens(context view.Context, namespace string, IDs []*token.ID, meta []string) ([]bool, error) {
	sIDs := make([]string, len(IDs))
	var err error
	for i, id := range IDs {
		sIDs[i], err = f.keyTranslator.CreateOutputKey(id.TxId, id.Index)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot compute spent id for [%v]", id)
		}
	}

	resBoxed, err := view2.GetManager(context).InitiateView(NewRequestSpentTokensView(f.network, namespace, sIDs), context.Context())
	if err != nil {
		return nil, err
	}
	return resBoxed.([]bool), nil
}
