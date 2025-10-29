/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/stretchr/testify/assert"
)

func TestMetadata_Validate(t *testing.T) {
	// Setup valid metadata for reference
	curve := math.BN254
	c := math.Curves[curve]
	rand, err := c.Rand()
	assert.NoError(t, err)

	validMetadata := &Metadata{
		Type:           "COIN",
		Value:          c.NewRandomZr(rand),
		BlindingFactor: c.NewRandomZr(rand),
		Issuer:         []byte("issuer1"),
	}

	tests := []struct {
		name    string
		meta    *Metadata
		wantErr string
	}{
		{
			name:    "valid metadata",
			meta:    validMetadata,
			wantErr: "",
		},
		{
			name: "missing type",
			meta: &Metadata{
				Type:           "",
				Value:          validMetadata.Value,
				BlindingFactor: validMetadata.BlindingFactor,
				Issuer:         validMetadata.Issuer,
			},
			wantErr: "missing Type",
		},
		{
			name: "missing value",
			meta: &Metadata{
				Type:           validMetadata.Type,
				Value:          nil,
				BlindingFactor: validMetadata.BlindingFactor,
				Issuer:         validMetadata.Issuer,
			},
			wantErr: "missing Value",
		},
		{
			name: "missing blinding factor",
			meta: &Metadata{
				Type:           validMetadata.Type,
				Value:          validMetadata.Value,
				BlindingFactor: nil,
				Issuer:         validMetadata.Issuer,
			},
			wantErr: "missing BlindingFactor",
		},
		{
			name: "missing issuer",
			meta: &Metadata{
				Type:           validMetadata.Type,
				Value:          validMetadata.Value,
				BlindingFactor: validMetadata.BlindingFactor,
				Issuer:         nil,
			},
			wantErr: "missing Issuer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.meta.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}
