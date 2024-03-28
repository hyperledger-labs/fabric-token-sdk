/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

// StoreTransactionRecords stores the transaction records extracted from the passed transaction to the
// token transaction db
func StoreTransactionRecords(context view.Context, tx *Transaction) error {
	return NewOwner(context, tx.TokenRequest.TokenService).Append(tx)
}

// RunView runs passed view within the passed context and using the passed options in a separate goroutine
func RunView(context view.Context, view view.View, opts ...view.RunViewOption) {
	defer func() {
		if r := recover(); r != nil {
			logger.Debugf("panic in RunView: %v", r)
		}
	}()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Debugf("panic in RunView: %v", r)
			}
		}()
		_, err := context.RunView(view, opts...)
		if err != nil {
			logger.Errorf("failed to run view: %s", err)
		}
	}()
}
