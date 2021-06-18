/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package driver

type AuditorService interface {
	AuditorCheck(tokenRequest *TokenRequest, tokenRequestMetadata *TokenRequestMetadata, txID string) error
}
