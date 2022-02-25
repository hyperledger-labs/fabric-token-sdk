/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package driver

type SerializedPublicParameters struct {
	Identifier string
	Raw        []byte
}

func (pp *SerializedPublicParameters) Deserialize(raw []byte) error {
	if err := Unmarshal(raw, pp); err != nil {
		return err
	}
	return nil
}

type PublicParamsFetcher interface {
	Fetch() ([]byte, error)
}

type PublicParameters interface {
	Identifier() string
	TokenDataHiding() bool
	GraphHiding() bool
	MaxTokenValue() uint64
	CertificationDriver() string
	Bytes() ([]byte, error)
}

type PublicParamsManager interface {
	PublicParameters() PublicParameters

	NewCertifierKeyPair() ([]byte, []byte, error)

	ForceFetch() error
}
