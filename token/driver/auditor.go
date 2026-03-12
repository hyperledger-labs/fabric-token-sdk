/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "context"

// AuditorService defines the methods for auditing token transactions.
// It provides mechanisms for authorized auditors to inspect transaction requests
// and their associated metadata and anchor to ensure compliance and validity.
//
//go:generate counterfeiter -o mock/auditor_service.go -fake-name AuditorService . AuditorService
type AuditorService interface {
	// AuditorCheck performs a comprehensive validation of a token request and its metadata.
	// It ensures that the transaction is well-formed and meets all auditing requirements
	// within the context of the provided anchor.
	AuditorCheck(ctx context.Context, request *TokenRequest, metadata *TokenRequestMetadata, anchor TokenRequestAnchor) error
}
