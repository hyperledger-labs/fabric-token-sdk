/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

type (
	// PPHash is used to model the hash of the raw public parameters.
	// This should avoid confusion between the bytes of the public params themselves and its hash.
	PPHash []byte
	// TokenDriverName is the name of a token driver
	TokenDriverName string
	// TokenDriverVersion is the version of a token driver
	TokenDriverVersion uint64
)

type PPReader interface {
	// PublicParametersFromBytes unmarshals the bytes to a PublicParameters instance.
	PublicParametersFromBytes(params []byte) (PublicParameters, error)
}

// PPMFactory contains the static logic of the driver
type PPMFactory interface {
	PPReader
	// NewPublicParametersManager returns a new PublicParametersManager instance from the passed public parameters
	NewPublicParametersManager(pp PublicParameters) (PublicParamsManager, error)
	// DefaultValidator returns a new Validator instance from the passed public parameters
	DefaultValidator(pp PublicParameters) (Validator, error)
}

// PublicParamsFetcher models a public parameters fetcher.
type PublicParamsFetcher interface {
	// Fetch fetches the public parameters from a repository.
	Fetch() ([]byte, error)
}

type Extras = map[string][]byte

//go:generate counterfeiter -o mock/pp.go -fake-name PublicParameters . PublicParameters

// PublicParameters is the interface that must be implemented by the driver public parameters.
type PublicParameters interface {
	// TokenDriverName returns the name of the token driver
	TokenDriverName() TokenDriverName
	// TokenDriverVersion return the version of the token driver
	TokenDriverVersion() TokenDriverVersion
	// TokenDataHiding returns true if the token data is hidden
	TokenDataHiding() bool
	// GraphHiding returns true if the token graph is hidden
	GraphHiding() bool
	// MaxTokenValue returns the maximum token value
	MaxTokenValue() uint64
	// CertificationDriver returns the certification driver identifier
	CertificationDriver() string
	// Auditors returns the list of auditors.
	Auditors() []Identity
	// Issuers returns the list of issuers.
	Issuers() []Identity
	// Precision returns the precision used to represent the token value.
	Precision() uint64
	// String returns a readable version of the public parameters
	String() string
	// Serialize returns the serialized version of this public parameters
	Serialize() ([]byte, error)
	// Validate returns true if the public parameters are well-formed
	Validate() error
	// Extras gives access to extra data, if available.
	// Extras might return nil if no extra data is present.
	Extras() Extras
}

//go:generate counterfeiter -o mock/ppm.go -fake-name PublicParamsManager . PublicParamsManager

// PublicParamsManager is the interface that must be implemented by the driver public parameters' manager.
type PublicParamsManager interface {
	// PublicParameters returns the public parameters.
	PublicParameters() PublicParameters
	// NewCertifierKeyPair generates a new key pair for the certifier, if supported
	NewCertifierKeyPair() ([]byte, []byte, error)
	// PublicParamsHash returns the hash of the raw public parameters
	PublicParamsHash() PPHash
}
