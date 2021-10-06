/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

func (s *Service) GetAuditorVerifier(id view.Identity) (driver.Verifier, error) {
	d, err := s.Deserializer()
	if err != nil {
		return nil, err
	}
	return d.GetAuditorVerifier(id)
}

func (s *Service) GetOwnerVerifier(id view.Identity) (driver.Verifier, error) {
	d, err := s.Deserializer()
	if err != nil {
		return nil, err
	}
	return d.GetOwnerVerifier(id)
}

func (s *Service) GetIssuerVerifier(id view.Identity) (driver.Verifier, error) {
	d, err := s.Deserializer()
	if err != nil {
		return nil, err
	}
	return d.GetIssuerVerifier(id)
}

func (s *Service) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	d, err := s.Deserializer()
	if err != nil {
		return nil, err
	}
	return d.GetOwnerMatcher(raw)
}
