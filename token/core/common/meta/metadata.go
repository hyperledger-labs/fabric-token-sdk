/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package meta

import (
	"strings"
)

const (
	TransferMetadataPrefix = "TransferMetadataPrefix"
)

// TransferActionMetadata extracts the transfer metadata from the passed attributes and
// sets them to the passed metadata
func TransferActionMetadata(attrs map[interface{}]interface{}) map[string][]byte {
	metadata := map[string][]byte{}
	for key, value := range attrs {
		k, ok1 := key.(string)
		v, ok2 := value.([]byte)
		if ok1 && ok2 {
			if strings.HasPrefix(k, TransferMetadataPrefix) {
				mKey := strings.TrimPrefix(k, TransferMetadataPrefix)
				metadata[mKey] = v
			}
		}
	}
	return metadata
}
