/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package api

type AuditorService interface {
	AuditorCheck(tokenRequest *TokenRequest, tokenRequestMetadata *TokenRequestMetadata, txID string) error
}
