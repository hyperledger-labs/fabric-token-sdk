/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package meta

import (
	"strings"
)

const (
	// TransferMetadataPrefix is the prefix for the metadata of a transfer action
	TransferMetadataPrefix = "TransferMetadataPrefix"
	// IssueMetadataPrefix is the prefix for the metadata of an issue action
	IssueMetadataPrefix = "IssueMetadataPrefix"
	// PublicMetadataPrefix is the prefix for the metadata that will be published on the ledger without further validation
	PublicMetadataPrefix = "pub."
)

// TransferActionMetadata extracts the transfer metadata from the passed attributes and
// sets them to the passed metadata
func TransferActionMetadata(attrs map[interface{}]interface{}) map[string][]byte {
	return ActionMetadata(attrs, TransferMetadataPrefix)
}

// IssueActionMetadata extracts the transfer metadata from the passed attributes and
// sets them to the passed metadata
func IssueActionMetadata(attrs map[interface{}]interface{}) map[string][]byte {
	return ActionMetadata(attrs, IssueMetadataPrefix)
}

// ActionMetadata extracts the metadata that has the passed prefix from the passed attributes and
// sets them to the passed metadata
func ActionMetadata(attrs map[interface{}]interface{}, prefix string) map[string][]byte {
	metadata := map[string][]byte{}
	for key, value := range attrs {
		k, ok1 := key.(string)
		v, ok2 := value.([]byte)
		if ok1 && ok2 {
			if strings.HasPrefix(k, prefix) {
				mKey := strings.TrimPrefix(k, prefix)
				metadata[mKey] = v
			}
		}
	}
	return metadata
}
