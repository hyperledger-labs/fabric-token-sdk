/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

// GetAuditorVerifier deserializes the verifier for the passed auditor identity
func (s *Service) GetAuditorVerifier(id view.Identity) (driver.Verifier, error) {
	d, err := s.Deserializer()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deserializer")
	}
	return d.GetAuditorVerifier(id)
}

// GetOwnerVerifier deserializes the verifier for the passed owner identity
func (s *Service) GetOwnerVerifier(id view.Identity) (driver.Verifier, error) {
	d, err := s.Deserializer()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deserializer")
	}
	return d.GetOwnerVerifier(id)
}

// GetIssuerVerifier deserializes the verifier for the passed issuer identity
func (s *Service) GetIssuerVerifier(id view.Identity) (driver.Verifier, error) {
	d, err := s.Deserializer()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deserializer")
	}
	return d.GetIssuerVerifier(id)
}

// GetOwnerMatcher deserializes the passed bytes into a Matcher
// The Matcher can be used later to match an identity to its audit information
func (s *Service) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	d, err := s.Deserializer()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deserializer")
	}
	return d.GetOwnerMatcher(raw)
}

func (s *Service) Recipients(raw []byte) ([]view.Identity, error) {
	d, err := s.Deserializer()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deserializer")
	}
	return d.Recipients(raw)
}

func (s *Service) Match(id view.Identity, ai []byte) error {
	d, err := s.Deserializer()
	if err != nil {
		return errors.Wrap(err, "failed to get deserializer")
	}
	return d.Match(id, ai)
}

func (s *Service) GetOwnerAuditInfo(raw []byte, p driver.AuditInfoProvider) ([][]byte, error) {
	d, err := s.Deserializer()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deserializer")
	}
	return d.GetOwnerAuditInfo(raw, p)
}
