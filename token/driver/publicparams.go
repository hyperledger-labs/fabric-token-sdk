/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

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

// PublicParamsFetcher models a public parameters fetcher.
type PublicParamsFetcher interface {
	// Fetch fetches the public parameters from a repository.
	Fetch() ([]byte, error)
}

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
	Auditors() []view.Identity
	// Precision returns the precision used to represent the token value.
	Precision() uint64
}

// PublicParamsManager is the interface that must be implemented by the driver public parameters manager.
type PublicParamsManager interface {
	// PublicParameters returns the public parameters.
	PublicParameters() PublicParameters
	// NewCertifierKeyPair generates a new key pair for the certifier, if supported
	NewCertifierKeyPair() ([]byte, []byte, error)
	// Update fetches the public parameters from the backend and write them locally
	Update() error
	// Fetch fetches the public parameters
	Fetch() ([]byte, error)
	// SerializePublicParameters returns the public params in a serialized form
	SerializePublicParameters() ([]byte, error)
	// Validate validates the public parameters
	Validate() error
}
