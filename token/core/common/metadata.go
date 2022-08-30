/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

// SetTransferActionMetadata extracts the transfer metadata from the passed attributes and
// sets them to the passed metadata
func SetTransferActionMetadata(attrs map[interface{}]interface{}, metadata map[string][]byte) {
	for key, value := range attrs {
		k, ok1 := key.(string)
		v, ok2 := value.([]byte)
		if ok1 && ok2 {
			if strings.HasPrefix(k, token.TransferMetadataPrefix) {
				mKey := strings.TrimPrefix(k, token.TransferMetadataPrefix)
				metadata[mKey] = v
			}
		}
	}
}
