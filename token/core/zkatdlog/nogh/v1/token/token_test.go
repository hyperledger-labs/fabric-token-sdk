/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/test-go/testify/require"
)

func TestMetadata_Validate(t *testing.T) {
	// Setup valid metadata for reference
	curve := math.BN254
	c := math.Curves[curve]
	rand, err := c.Rand()
	require.NoError(t, err)

	validMetadata := &Metadata{
		Type:           "COIN",
		Value:          c.NewRandomZr(rand),
		BlindingFactor: c.NewRandomZr(rand),
		Issuer:         []byte("issuer1"),
	}

	tests := []struct {
		name        string
		meta        *Metadata
		checkIssuer bool
		wantErr     string
	}{
		{
			name:        "valid metadata",
			meta:        validMetadata,
			checkIssuer: true,
			wantErr:     "",
		},
		{
			name: "missing type",
			meta: &Metadata{
				Type:           "",
				Value:          validMetadata.Value,
				BlindingFactor: validMetadata.BlindingFactor,
				Issuer:         validMetadata.Issuer,
			},
			checkIssuer: true,
			wantErr:     ErrEmptyType.Error(),
		},
		{
			name: "missing value",
			meta: &Metadata{
				Type:           validMetadata.Type,
				Value:          nil,
				BlindingFactor: validMetadata.BlindingFactor,
				Issuer:         validMetadata.Issuer,
			},
			checkIssuer: true,
			wantErr:     ErrEmptyValue.Error(),
		},
		{
			name: "missing blinding factor",
			meta: &Metadata{
				Type:           validMetadata.Type,
				Value:          validMetadata.Value,
				BlindingFactor: nil,
				Issuer:         validMetadata.Issuer,
			},
			checkIssuer: true,
			wantErr:     ErrEmptyBlindingFactor.Error(),
		},
		{
			name: "missing issuer",
			meta: &Metadata{
				Type:           validMetadata.Type,
				Value:          validMetadata.Value,
				BlindingFactor: validMetadata.BlindingFactor,
				Issuer:         nil,
			},
			checkIssuer: true,
			wantErr:     ErrMissingIssuer.Error(),
		},
		{
			name: "should not have the issuer",
			meta: &Metadata{
				Type:           validMetadata.Type,
				Value:          validMetadata.Value,
				BlindingFactor: validMetadata.BlindingFactor,
				Issuer:         validMetadata.Issuer,
			},
			checkIssuer: false,
			wantErr:     ErrUnexpectedIssuer.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.meta.Validate(tt.checkIssuer)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr)
			}
		})
	}
}
