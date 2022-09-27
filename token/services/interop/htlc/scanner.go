/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"bytes"
	"crypto"
	"encoding/base64"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
)

const (
	ScanForPreImageStartingTransaction = "htlc.ScanForPreImage.StartingTransaction"
)

// WithStartingTransaction sets the network name
func WithStartingTransaction(txID string) token.ServiceOption {
	return func(o *token.ServiceOptions) error {
		if o.Params == nil {
			o.Params = map[string]interface{}{}
		}
		o.Params[ScanForPreImageStartingTransaction] = txID
		return nil
	}
}

// ScanForPreImage scans the ledger for a preimage of the passed image, taking into account the timeout
func ScanForPreImage(ctx view.Context, image []byte, hashFunc crypto.Hash, hashEncoding encoding.Encoding, timeout time.Duration, opts ...token.ServiceOption) ([]byte, error) {
	logger.Debugf("scanning for preimage of [%s] with timeout [%s]", base64.StdEncoding.EncodeToString(image), timeout)

	if !hashFunc.Available() {
		return nil, errors.Errorf("passed hash function is not available [%d]", hashFunc)
	}
	if !hashEncoding.Available() {
		return nil, errors.Errorf("passed hash endcoding is not available [%d]", hashEncoding)
	}

	tokenOptions, err := token.CompileServiceOptions(opts...)
	if err != nil {
		return nil, err
	}
	tms := token.GetManagementService(ctx, opts...)

	network := network.GetInstance(ctx, tms.Network(), tms.Channel())
	if network == nil {
		return nil, errors.Errorf("cannot find network [%s:%s]", tms.Namespace(), tms.Channel())
	}

	startingTxID, err := tokenOptions.ParamAsString(ScanForPreImageStartingTransaction)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid starting transaction param")
	}

	claimKey := ClaimKey(image)
	preImage, err := network.LookupTransferMetadataKey(tms.Namespace(), startingTxID, claimKey, timeout, opts...)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to lookup key [%s]", claimKey)
	}
	recomputedImage, err := (&HashInfo{
		HashFunc:     hashFunc,
		HashEncoding: hashEncoding,
	}).Image(preImage)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to compute image of [%x]", preImage)
	}
	if !bytes.Equal(image, recomputedImage) {
		return nil, errors.WithMessagef(err, "pre-image on the ledger does not match the passed image [%x!=%x]", image, recomputedImage)
	}
	return preImage, nil
}
