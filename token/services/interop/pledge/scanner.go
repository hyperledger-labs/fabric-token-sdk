/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
)

const (
	ScanForPledgeIDStartingTransaction = "pledge.IDExists.StartingTransaction"
)

// WithStartingTransaction sets the starting transaction for the scan
func WithStartingTransaction(txID string) token.ServiceOption {
	return func(o *token.ServiceOptions) error {
		if o.Params == nil {
			o.Params = map[string]interface{}{}
		}
		o.Params[ScanForPledgeIDStartingTransaction] = txID
		return nil
	}
}

// IDExists scans the ledger for a pledge identifier, taking into account the timeout
// IDExists returns true, if entry identified by key (MetadataKey+pledgeID) is occupied.
func IDExists(ctx view.Context, pledgeID string, timeout time.Duration, opts ...token.ServiceOption) (bool, error) {
	logger.Infof("scanning for pledgeID of [%s] with timeout [%s]", pledgeID, timeout)
	tokenOptions, err := token.CompileServiceOptions(opts...)
	if err != nil {
		return false, err
	}
	tms := token.GetManagementService(ctx, opts...)

	net := network.GetInstance(ctx, tms.Network(), tms.Channel())
	if net == nil {
		return false, errors.Errorf("cannot find network [%s:%s]", tms.Namespace(), tms.Channel())
	}

	startingTxID, err := tokenOptions.ParamAsString(ScanForPledgeIDStartingTransaction)
	if err != nil {
		return false, errors.Wrapf(err, "invalid starting transaction param")
	}

	pledgeKey := MetadataKey + pledgeID
	v, err := net.LookupTransferMetadataKey(tms.Namespace(), startingTxID, pledgeKey, timeout, opts...)
	if err != nil {
		return false, errors.Wrapf(err, "failed to lookup transfer metadata for pledge ID [%s]", pledgeID)
	}
	if len(v) != 0 {
		return true, nil
	}
	return false, nil
}
