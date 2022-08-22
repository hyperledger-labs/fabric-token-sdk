/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

// AuditorService models the auditor service
type AuditorService interface {
	// AuditorCheck verifies the well-formedness of the passed request with the respect to the passed metadata and anchor
	AuditorCheck(request *TokenRequest, metadata *TokenRequestMetadata, anchor string) error
}
