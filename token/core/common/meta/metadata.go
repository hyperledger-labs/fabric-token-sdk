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
func TransferActionMetadata(attrs map[any]any) map[string][]byte {
	return ActionMetadata(attrs, TransferMetadataPrefix)
}

// IssueActionMetadata extracts the transfer metadata from the passed attributes and
// sets them to the passed metadata
func IssueActionMetadata(attrs map[any]any) map[string][]byte {
	return ActionMetadata(attrs, IssueMetadataPrefix)
}

// ActionMetadata extracts the metadata that has the passed prefix from the passed attributes and
// sets them to the passed metadata
func ActionMetadata(attrs map[any]any, prefix string) map[string][]byte {
	metadata := map[string][]byte{}
	for key, value := range attrs {
		k, ok1 := key.(string)
		v, ok2 := value.([]byte)
		if ok1 && ok2 {
			if after, ok := strings.CutPrefix(k, prefix); ok {
				mKey := after
				metadata[mKey] = v
			}
		}
	}

	return metadata
}
