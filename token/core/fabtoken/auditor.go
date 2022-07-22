/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import "github.com/hyperledger-labs/fabric-token-sdk/token/driver"

// AuditorCheck verifies if the passed tokenRequest matches the tokenRequestMetadata
// fabtoken does not make use of AuditorCheck as the token request contains token
// information in the clear
func (s *Service) AuditorCheck(tokenRequest *driver.TokenRequest, tokenRequestMetadata *driver.TokenRequestMetadata, txID string) error {
	return nil
}
