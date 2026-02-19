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
	p := &TokenTxValidateParams{}
	p.applyDefaults()

	require.Equal(t, deafultNumOutputs, p.NumOutputs)
	require.Equal(t, uint64(defaultBitLength), p.BitLength)
	require.Equal(t, defaultTokenType, p.TokenType)
	require.Equal(t, int(defaultCurveID), p.CurveID)
}

func TestApplyDefaults_PreservesExplicitValues(t *testing.T) {
	p := &TokenTxValidateParams{
		NumOutputs: 3,
		BitLength:  64,
		TokenType:  "my-token",
		CurveID:    int(math.BN254),
	}
	p.applyDefaults()

	require.Equal(t, 3, p.NumOutputs)
	require.Equal(t, uint64(64), p.BitLength)
	require.Equal(t, "my-token", p.TokenType)
	require.Equal(t, int(math.BN254), p.CurveID)
}

func TestApplyDefaults_NegativeInputsOutputs(t *testing.T) {
	p := &TokenTxValidateParams{NumOutputs: -5}
	p.applyDefaults()

	require.Equal(t, deafultNumOutputs, p.NumOutputs)
}

func TestNewView_DefaultParams(t *testing.T) {
	factory := &TokenTxValidateViewFactory{}
	input, err := json.Marshal(&TokenTxValidateParams{})
	require.NoError(t, err)

	v, err := factory.NewView(input)
	require.NoError(t, err)
	require.NotNil(t, v)

	tv := v.(*TokenTxValidateView)
	require.Equal(t, deafultNumOutputs, tv.params.NumOutputs)
	require.Equal(t, uint64(defaultBitLength), tv.params.BitLength)
	require.Equal(t, defaultTokenType, tv.params.TokenType)
	require.Equal(t, int(defaultCurveID), tv.params.CurveID)
	require.NotNil(t, tv.pubParams)
	require.NotEmpty(t, tv.actionRaw)
}

func TestNewView_CustomParams(t *testing.T) {
	factory := &TokenTxValidateViewFactory{}
	p := &TokenTxValidateParams{
		NumOutputs: 3,
		BitLength:  64,
		TokenType:  "gold",
		CurveID:    int(math.BLS12_381_BBS_GURVY),
	}
	input, err := json.Marshal(p)
	require.NoError(t, err)

	v, err := factory.NewView(input)
	require.NoError(t, err)
	require.NotNil(t, v)

	tv := v.(*TokenTxValidateView)
	require.Equal(t, 3, tv.params.NumOutputs)
	require.Equal(t, uint64(64), tv.params.BitLength)
	require.Equal(t, "gold", tv.params.TokenType)
}

func TestNewView_InvalidJSON(t *testing.T) {
	factory := &TokenTxValidateViewFactory{}
	_, err := factory.NewView([]byte(`{invalid`))
	require.Error(t, err)
}

func TestCall_VerifiesProofSuccessfully(t *testing.T) {
	factory := &TokenTxValidateViewFactory{}

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
			p := &TokenTxValidateParams{NumOutputs: tt.numOutputs}
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
	factory := &TokenTxValidateViewFactory{}
	p := &TokenTxValidateParams{NumOutputs: 2}
	input, err := json.Marshal(p)
	require.NoError(t, err)

	v, err := factory.NewView(input)
	require.NoError(t, err)

	tv := v.(*TokenTxValidateView)
	// Corrupt the serialized action to invalidate the proof.
	for i := len(tv.actionRaw) - 1; i >= len(tv.actionRaw)-8 && i >= 0; i-- {
		tv.actionRaw[i] ^= 0xFF
	}

	_, err = tv.Call(nil)
	require.Error(t, err)
}

func TestCall_EmptyActionRawFails(t *testing.T) {
	v := &TokenTxValidateView{actionRaw: []byte{}}
	_, err := v.Call(nil)
	require.Error(t, err)
}

func TestCall_NilActionRawFails(t *testing.T) {
	v := &TokenTxValidateView{}
	_, err := v.Call(nil)
	require.Error(t, err)
}

func TestNewView_MultipleOutputCounts(t *testing.T) {
	factory := &TokenTxValidateViewFactory{}

	for _, numOutputs := range []int{1, 2, 4} {
		t.Run("outputs_"+string(rune('0'+numOutputs)), func(t *testing.T) {
			p := &TokenTxValidateParams{NumOutputs: numOutputs}
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
	original := &TokenTxValidateParams{
		NumOutputs: 5,
		BitLength:  64,
		TokenType:  "silver",
		CurveID:    int(math.BN254),
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	decoded := &TokenTxValidateParams{}
	require.NoError(t, json.Unmarshal(data, decoded))

	require.Equal(t, original, decoded)
}
