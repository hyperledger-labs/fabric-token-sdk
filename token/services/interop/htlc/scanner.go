/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"bytes"
	"context"
	"crypto"
	"encoding/base64"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/translator"
	"go.uber.org/zap/zapcore"
)

// ScanForPreImage scans the ledger for a preimage of the passed image, taking into account the timeout
func ScanForPreImage(ctx view.Context, image []byte, hashFunc crypto.Hash, hashEncoding encoding.Encoding, timeout time.Duration, opts ...token.ServiceOption) ([]byte, error) {
	logger.Debugf("scanning for preimage of [%s] with timeout [%s]", base64.StdEncoding.EncodeToString(image), timeout)

	tokenOptions, err := token.CompileServiceOptions(opts...)
	if err != nil {
		return nil, err
	}
	ch, err := fabric.GetFabricNetworkService(ctx, tokenOptions.Network).Channel(tokenOptions.Channel)
	if err != nil {
		return nil, err
	}
	tms := token.GetManagementService(ctx, opts...)

	var preImage []byte
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := ch.Delivery().Scan(c, "", func(tx *fabric.ProcessedTransaction) (bool, error) {
		logger.Debugf("scanning [%s]...", tx.TxID())

		rws, err := ch.Vault().GetEphemeralRWSet(tx.Results())
		if err != nil {
			return false, err
		}

		found := false
		for _, ns := range rws.Namespaces() {
			if ns == tms.Namespace() {
				found = true
				break
			}
		}
		if !found {
			logger.Debugf("scanning [%s] does not contain namespace [%s]", tx.TxID(), tms.Namespace())
			return false, nil
		}

		ns := tms.Namespace()
		w := translator.New(tx.TxID(), fabric2.NewRWSWrapper(rws), tms.Namespace())
		for i := 0; i < rws.NumWrites(ns); i++ {
			k, v, err := rws.GetWriteAt(ns, i)
			if err != nil {
				return false, err
			}
			if f, err := w.IsTransferMetadataKeyWithSubKey(k, ClaimPreImage); err == nil && f {
				// hash + encoding
				hash := hashFunc.New()
				if _, err = hash.Write(v); err != nil {
					return false, err
				}
				recomputedImage := hash.Sum(nil)
				encoding := hashEncoding.New()
				recomputedImage = []byte(encoding.EncodeToString(recomputedImage))

				// compare
				if !bytes.Equal(image, recomputedImage) {
					continue
				}

				// found
				preImage = v
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("preimage of [%s] found [%s]",
						base64.StdEncoding.EncodeToString(image),
						base64.StdEncoding.EncodeToString(v),
					)
				}
				return true, nil
			}
		}
		logger.Debugf("scanning for preimage on [%s] not found", tx.TxID())
		return false, nil
	}); err != nil {
		return nil, err
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("scanning for preimage of [%s] with timeout [%s] found, [%s]",
			base64.StdEncoding.EncodeToString(image),
			timeout,
			base64.StdEncoding.EncodeToString(preImage),
		)
	}

	return preImage, nil
}
