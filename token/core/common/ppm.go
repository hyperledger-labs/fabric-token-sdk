/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/pkg/errors"
)

type PublicParamsDeserializer[T driver.PublicParameters] interface {
	DeserializePublicParams(raw []byte, label string) (T, error)
}

type PublicParamsManager[T driver.PublicParameters] struct {
	publicParameters T
	// label of the public params
	PPLabel string
	ppHash  driver.PPHash
}

func NewPublicParamsManager[T driver.PublicParameters](
	PublicParamsDeserializer PublicParamsDeserializer[T],
	PPLabel string,
	ppRaw []byte,
) (*PublicParamsManager[T], error) {
	ppm := &PublicParamsManager[T]{
		PPLabel: PPLabel,
	}
	if len(ppRaw) == 0 {
		return nil, errors.Errorf("empty public parameters")
	}
	pp, err := PublicParamsDeserializer.DeserializePublicParams(ppRaw, PPLabel)
	if err != nil {
		return nil, err
	}
	if err := pp.Validate(); err != nil {
		return nil, errors.WithMessage(err, "invalid public parameters")
	}
	ppm.publicParameters = pp
	ppm.ppHash = utils.Hashable(ppRaw).Raw()

	return ppm, nil
}

func NewPublicParamsManagerFromParams[T driver.PublicParameters](pp T) (*PublicParamsManager[T], error) {
	if err := pp.Validate(); err != nil {
		return nil, errors.WithMessage(err, "invalid public parameters")
	}
	return &PublicParamsManager[T]{
		publicParameters: pp,
	}, nil
}

func (v *PublicParamsManager[T]) PublicParameters() driver.PublicParameters {
	return v.publicParameters
}

func (v *PublicParamsManager[T]) NewCertifierKeyPair() ([]byte, []byte, error) {
	return nil, nil, errors.Errorf("not supported")
}

func (v *PublicParamsManager[T]) PublicParams() T {
	return v.publicParameters
}

func (v *PublicParamsManager[T]) PublicParamsHash() driver.PPHash {
	return v.ppHash
}
