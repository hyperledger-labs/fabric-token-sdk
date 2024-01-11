/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
)

type TokenTransactionDB interface {
	GetTokenRequest(txID string) ([]byte, error)
	GetTransactionEndorsementAcks(id string) (map[string][]byte, error)
}

// TransactionInfo contains the transaction info.
type TransactionInfo struct {
	// EndorsementAcks contains the endorsement ACKs received at time of dissemination.
	EndorsementAcks map[string][]byte
	// ApplicationMetadata contains the application metadata
	ApplicationMetadata map[string][]byte

	TokenRequest []byte
}

// TransactionInfoProvider allows the retrieval of the transaction info
type TransactionInfoProvider struct {
	sp    view.ServiceProvider
	tms   *token.ManagementService
	ttxDB TokenTransactionDB
}

func newTransactionInfoProvider(sp view.ServiceProvider, tms *token.ManagementService, ttxDB TokenTransactionDB) *TransactionInfoProvider {
	return &TransactionInfoProvider{sp: sp, tms: tms, ttxDB: ttxDB}
}

// TransactionInfo returns the transaction info for the given transaction ID.
func (a *TransactionInfoProvider) TransactionInfo(txID string) (*TransactionInfo, error) {
	endorsementAcks, err := a.ttxDB.GetTransactionEndorsementAcks(txID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to load endorsement acks for [%s]", txID)
	}

	applicationMetadata, err := a.loadTransient(txID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to load transient for [%s]", txID)
	}

	tr, err := a.ttxDB.GetTokenRequest(txID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to load token request for [%s]", txID)
	}

	return &TransactionInfo{
		EndorsementAcks:     endorsementAcks,
		ApplicationMetadata: applicationMetadata,
		TokenRequest:        tr,
	}, nil
}

func (a *TransactionInfoProvider) loadTransient(txID string) (map[string][]byte, error) {
	tm, err := network.GetInstance(a.sp, a.tms.Network(), a.tms.Channel()).GetTransient(txID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to load transient for [%s]", txID)
	}
	if !tm.Exists(TokenRequestMetadata) {
		return nil, nil
	}
	raw := tm.Get(TokenRequestMetadata)

	metadata, err := a.tms.NewMetadataFromBytes(raw)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshal transient for [%s]", txID)
	}

	return metadata.TokenRequestMetadata.Application, nil
}
