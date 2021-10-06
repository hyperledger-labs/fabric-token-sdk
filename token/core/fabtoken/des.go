/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

func (s *Service) GetAuditorVerifier(id view.Identity) (driver.Verifier, error) {
	return s.Deserializer.GetAuditorVerifier(id)
}

func (s *Service) GetOwnerVerifier(id view.Identity) (driver.Verifier, error) {
	return s.Deserializer.GetOwnerVerifier(id)
}

func (s *Service) GetIssuerVerifier(id view.Identity) (driver.Verifier, error) {
	return s.Deserializer.GetIssuerVerifier(id)
}

func (s *Service) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	return s.Deserializer.GetOwnerMatcher(raw)
}
