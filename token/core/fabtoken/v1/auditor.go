/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// AuditorService is a service that handles auditing of token requests.
type AuditorService struct{}

// NewAuditorService returns a new instance of AuditorService.
func NewAuditorService() *AuditorService {
	return &AuditorService{}
}

// AuditorCheck verifies if the passed tokenRequest matches the tokenRequestMetadata.
// In fabtoken, this function is a no-op as the token request contains token
// information in the clear. Therefore, it always returns nil.
func (s *AuditorService) AuditorCheck(ctx context.Context, request *driver.TokenRequest, metadata *driver.TokenRequestMetadata, anchor driver.TokenRequestAnchor) error {
	return nil
}
