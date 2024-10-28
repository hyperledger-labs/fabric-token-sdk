/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/chaincode"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

func NewTokenExecutorProvider() *tokenFetcherProvider {
	return &tokenFetcherProvider{}
}

type tokenFetcherProvider struct{}

func (p *tokenFetcherProvider) GetExecutor(network, channel string) (driver.TokenQueryExecutor, error) {
	return &tokenFetcher{network: network, channel: channel}, nil
}

type tokenFetcher struct {
	network string
	channel string
}

func (f *tokenFetcher) QueryTokens(context view.Context, namespace string, IDs []*token.ID) ([][]byte, error) {
	idsRaw, err := json.Marshal(IDs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling ids")
	}

	payloadBoxed, err := context.RunView(chaincode.NewQueryView(
		namespace,
		QueryTokensFunctions,
		idsRaw,
	).WithNetwork(f.network).WithChannel(f.channel))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to query the token chaincode for tokens")
	}

	// Unbox
	raw, ok := payloadBoxed.([]byte)
	if !ok {
		return nil, errors.Errorf("expected []byte from TCC, got [%T]", payloadBoxed)
	}
	var tokens [][]byte
	if err := json.Unmarshal(raw, &tokens); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal response")
	}

	return tokens, nil
}

func NewSpentTokenExecutorProvider() *spentTokenFetcherProvider {
	return &spentTokenFetcherProvider{}
}

type spentTokenFetcherProvider struct{}

func (p *spentTokenFetcherProvider) GetSpentExecutor(network, channel string) (driver.SpentTokenQueryExecutor, error) {
	return &spentTokenFetcher{network: network, channel: channel}, nil
}

type spentTokenFetcher struct {
	network string
	channel string
}

func (f *spentTokenFetcher) QuerySpentTokens(context view.Context, namespace string, IDs []*token.ID, meta []string) ([]bool, error) {
	sIDs := make([]string, len(IDs))
	var err error
	for i, id := range IDs {
		sIDs[i], err = keys.CreateTokenKey(id.TxId, id.Index)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot compute spent id for [%v]", id)
		}
	}

	idsRaw, err := json.Marshal(sIDs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling ids")
	}

	payloadBoxed, err := context.RunView(chaincode.NewQueryView(
		namespace,
		AreTokensSpent,
		idsRaw,
	).WithNetwork(f.network).WithChannel(f.channel))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to query the token chaincode for tokens spent")
	}

	// Unbox
	raw, ok := payloadBoxed.([]byte)
	if !ok {
		return nil, errors.Errorf("expected []byte from TCC, got [%T]", payloadBoxed)
	}
	var spent []bool
	if err := json.Unmarshal(raw, &spent); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal esponse")
	}

	return spent, nil
}
