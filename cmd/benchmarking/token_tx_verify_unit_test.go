/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmarking

import (
	"encoding/json"
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/stretchr/testify/require"
)

func TestApplyDefaults_AllDefaultValues(t *testing.T) {
	p := &TokenTxVerifyMetadata{}
	p.applyDefaults()

	require.Equal(t, deafultNumOutputs, p.NumOutputTokens)
	require.Equal(t, uint64(defaultBitLength), p.BitLength)
	require.Equal(t, defaultTokenType, p.TokenType)
	require.Equal(t, int(defaultCurveID), p.CurveID)
}

func TestApplyDefaults_PreservesExplicitValues(t *testing.T) {
	p := &TokenTxVerifyMetadata{
		NumOutputTokens: 3,
		BitLength:       64,
		TokenType:       "my-token",
		CurveID:         int(math.BN254),
	}
	p.applyDefaults()

	require.Equal(t, 3, p.NumOutputTokens)
	require.Equal(t, uint64(64), p.BitLength)
	require.Equal(t, "my-token", p.TokenType)
	require.Equal(t, int(math.BN254), p.CurveID)
}

func TestApplyDefaults_NegativeInputsOutputs(t *testing.T) {
	p := &TokenTxVerifyMetadata{NumOutputTokens: -5}
	p.applyDefaults()

	require.Equal(t, deafultNumOutputs, p.NumOutputTokens)
}

func TestNewView_DefaultParams(t *testing.T) {
	factory := &TokenTxVerifyViewFactory{}
	input, err := json.Marshal(&TokenTxVerifyMetadata{})
	require.NoError(t, err)

	v, err := factory.NewView(input)
	require.NoError(t, err)
	require.NotNil(t, v)

	tv := v.(*TokenTxVerifyView)
	require.Equal(t, deafultNumOutputs, tv.meta.NumOutputTokens)
	require.Equal(t, uint64(defaultBitLength), tv.meta.BitLength)
	require.Equal(t, defaultTokenType, tv.meta.TokenType)
	require.Equal(t, int(defaultCurveID), tv.meta.CurveID)
	require.NotNil(t, tv.pubParams)
	require.NotEmpty(t, tv.actionRaw)
}

func TestNewView_CustomParams(t *testing.T) {
	factory := &TokenTxVerifyViewFactory{}
	p := &TokenTxVerifyMetadata{
		NumOutputTokens: 3,
		BitLength:       64,
		TokenType:       "gold",
		CurveID:         int(math.BLS12_381_BBS_GURVY),
	}
	input, err := json.Marshal(p)
	require.NoError(t, err)

	v, err := factory.NewView(input)
	require.NoError(t, err)
	require.NotNil(t, v)

	tv := v.(*TokenTxVerifyView)
	require.Equal(t, 3, tv.meta.NumOutputTokens)
	require.Equal(t, uint64(64), tv.meta.BitLength)
	require.Equal(t, "gold", tv.meta.TokenType)
}

func TestNewView_InvalidJSON(t *testing.T) {
	factory := &TokenTxVerifyViewFactory{}
	_, err := factory.NewView([]byte(`{invalid`))
	require.Error(t, err)
}

func TestCall_VerifiesProofSuccessfully(t *testing.T) {
	factory := &TokenTxVerifyViewFactory{}

	tests := []struct {
		name       string
		numOutputs int
	}{
		{"single_output", 1},
		{"two_outputs", 2},
		{"three_outputs", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &TokenTxVerifyMetadata{NumOutputTokens: tt.numOutputs}
			input, err := json.Marshal(p)
			require.NoError(t, err)

			v, err := factory.NewView(input)
			require.NoError(t, err)

			result, err := v.Call(nil)
			require.NoError(t, err)
			require.Nil(t, result)
		})
	}
}

func TestCall_TamperedProofFails(t *testing.T) {
	factory := &TokenTxVerifyViewFactory{}
	p := &TokenTxVerifyMetadata{NumOutputTokens: 2}
	input, err := json.Marshal(p)
	require.NoError(t, err)

	v, err := factory.NewView(input)
	require.NoError(t, err)

	tv := v.(*TokenTxVerifyView)
	// Corrupt the serialized action to invalidate the proof.
	for i := len(tv.actionRaw) - 1; i >= len(tv.actionRaw)-8 && i >= 0; i-- {
		tv.actionRaw[i] ^= 0xFF
	}

	_, err = tv.Call(nil)
	require.Error(t, err)
}

func TestCall_EmptyActionRawFails(t *testing.T) {
	v := &TokenTxVerifyView{actionRaw: []byte{}}
	_, err := v.Call(nil)
	require.Error(t, err)
}

func TestCall_NilActionRawFails(t *testing.T) {
	v := &TokenTxVerifyView{}
	_, err := v.Call(nil)
	require.Error(t, err)
}

func TestNewView_MultipleOutputCounts(t *testing.T) {
	factory := &TokenTxVerifyViewFactory{}

	for _, numOutputs := range []int{1, 2, 4} {
		t.Run("outputs_"+string(rune('0'+numOutputs)), func(t *testing.T) {
			p := &TokenTxVerifyMetadata{NumOutputTokens: numOutputs}
			input, err := json.Marshal(p)
			require.NoError(t, err)

			v, err := factory.NewView(input)
			require.NoError(t, err)

			result, err := v.Call(nil)
			require.NoError(t, err)
			require.Nil(t, result)
		})
	}
}

func TestParamsJSON_RoundTrip(t *testing.T) {
	original := &TokenTxVerifyMetadata{
		NumOutputTokens: 5,
		BitLength:       64,
		TokenType:       "silver",
		CurveID:         int(math.BN254),
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	decoded := &TokenTxVerifyMetadata{}
	require.NoError(t, json.Unmarshal(data, decoded))

	require.Equal(t, original, decoded)
}
