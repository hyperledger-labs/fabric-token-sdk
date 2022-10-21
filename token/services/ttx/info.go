/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	"github.com/pkg/errors"
)

const (
	// EndorsementAckPrefix is the prefix for the endorsement ACKs.
	EndorsementAckPrefix = "ttx.endorse.ack"
)

// TransactionInfo contains the transaction info.
type TransactionInfo struct {
	// EndorsementAcks contains the endorsement ACKs received at time of dissemination.
	EndorsementAcks map[string][]byte
	// ApplicationMetadata contains the application metadata
	ApplicationMetadata map[string][]byte
}

// TransactionInfoProvider allows the retrieval of the transaction info
type TransactionInfoProvider struct {
	sp  view.ServiceProvider
	tms *token.ManagementService
}

func NewTransactionInfoProvider(sp view.ServiceProvider, tms *token.ManagementService) *TransactionInfoProvider {
	return &TransactionInfoProvider{sp: sp, tms: tms}
}

// TransactionInfo returns the transaction info for the given transaction ID.
func (a *TransactionInfoProvider) TransactionInfo(txID string) (*TransactionInfo, error) {
	endorsementAcks, err := a.loadEndorsementAcks(txID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to load endorsement acks for [%s]", txID)
	}

	applicationMetadata, err := a.loadTransient(txID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to load transient for [%s]", txID)
	}

	return &TransactionInfo{
		EndorsementAcks:     endorsementAcks,
		ApplicationMetadata: applicationMetadata,
	}, nil
}

func (a *TransactionInfoProvider) loadEndorsementAcks(txID string) (map[string][]byte, error) {
	// Load transaction endorsement ACKs
	k := kvs.GetService(a.sp)
	acks := make(map[string][]byte)
	it, err := k.GetByPartialCompositeID(EndorsementAckPrefix, []string{txID})
	if err != nil {
		return nil, errors.WithMessagef(err, "failed loading ack for [%s]", txID)
	}
	defer it.Close()
	for {
		if !it.HasNext() {
			break
		}
		var ack []byte
		key, err := it.Next(&ack)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed loading ack for [%s]", txID)
		}

		objectType, attrs, err := kvs.SplitCompositeKey(key)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed splitting composite key for [%s]", txID)
		}
		if objectType != EndorsementAckPrefix {
			return nil, errors.Errorf("unexpected object type [%s]", objectType)
		}
		if len(attrs) != 2 {
			return nil, errors.Errorf("unexpected number of attributes [%d]", len(attrs))
		}
		acks[attrs[1]] = ack
	}
	return acks, nil
}

func (a *TransactionInfoProvider) loadTransient(txID string) (map[string][]byte, error) {
	tm, err := network.GetInstance(a.sp, a.tms.Network(), a.tms.Channel()).GetTransient(txID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to load transient for [%s]", txID)
	}
	if !tm.Exists(keys.TokenRequestMetadata) {
		return nil, nil
	}
	raw := tm.Get(keys.TokenRequestMetadata)

	metadata, err := a.tms.NewMetadataFromBytes(raw)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshal transient for [%s]", txID)
	}

	return metadata.TokenRequestMetadata.Application, nil
}
