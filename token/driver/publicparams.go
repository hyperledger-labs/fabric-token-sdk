/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"

// SerializedPublicParameters is the serialized form of PublicParameters.
type SerializedPublicParameters struct {
	// Identifier is the unique identifier of this public parameters.
	Identifier string
	// Raw is marshalled version of the public parameters.
	Raw []byte
}

// Deserialize deserializes the serialized public parameters.
func (pp *SerializedPublicParameters) Deserialize(raw []byte) error {
	if err := Unmarshal(raw, pp); err != nil {
		return err
	}
	return nil
}

// DefaultPublicParamsFetcher models a public parameters fetcher per namespace.
type DefaultPublicParamsFetcher interface {
	// Fetch fetches the public parameters from a repository for a given namespace.
	Fetch(network driver.Network, channel driver.Channel, namespace driver.Namespace) ([]byte, error)
}

// PublicParamsFetcher models a public parameters fetcher.
type PublicParamsFetcher interface {
	// Fetch fetches the public parameters from a repository.
	Fetch() ([]byte, error)
}

//go:generate counterfeiter -o mock/pp.go -fake-name PublicParameters . PublicParameters

// PublicParameters is the interface that must be implemented by the driver public parameters.
type PublicParameters interface {
	// Identifier returns the unique identifier of this public parameters.
	Identifier() string
	// TokenDataHiding returns true if the token data is hidden
	TokenDataHiding() bool
	// GraphHiding returns true if the token graph is hidden
	GraphHiding() bool
	// MaxTokenValue returns the maximum token value
	MaxTokenValue() uint64
	// CertificationDriver returns the certification driver identifier
	CertificationDriver() string
	// Bytes returns the marshalled version of the public parameters.
	Bytes() ([]byte, error)
	// Auditors returns the list of auditors.
	Auditors() []Identity
	// Precision returns the precision used to represent the token value.
	Precision() uint64
	// String returns a readable version of the public parameters
	String() string

	Serialize() ([]byte, error)
	Validate() error
}

//go:generate counterfeiter -o mock/ppm.go -fake-name PublicParamsManager . PublicParamsManager

// PublicParamsManager is the interface that must be implemented by the driver public parameters manager.
type PublicParamsManager interface {
	// PublicParameters returns the public parameters.
	PublicParameters() PublicParameters
	// NewCertifierKeyPair generates a new key pair for the certifier, if supported
	NewCertifierKeyPair() ([]byte, []byte, error)
}
