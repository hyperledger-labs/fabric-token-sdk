/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"errors"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServiceOptions_TMSID verifies TMSID returns correct identifier
func TestServiceOptions_TMSID(t *testing.T) {
	opts := ServiceOptions{
		Network:   "testnet",
		Channel:   "testchannel",
		Namespace: "testns",
	}

	tmsID := opts.TMSID()

	assert.Equal(t, "testnet", tmsID.Network)
	assert.Equal(t, "testchannel", tmsID.Channel)
	assert.Equal(t, "testns", tmsID.Namespace)
}

// TestServiceOptions_ParamAsString_Success verifies successful string parameter retrieval
func TestServiceOptions_ParamAsString_Success(t *testing.T) {
	opts := ServiceOptions{
		Params: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
	}

	val, err := opts.ParamAsString("key1")
	require.NoError(t, err)
	assert.Equal(t, "value1", val)
}

// TestServiceOptions_ParamAsString_NotFound verifies empty string for missing key
func TestServiceOptions_ParamAsString_NotFound(t *testing.T) {
	opts := ServiceOptions{
		Params: map[string]interface{}{
			"key1": "value1",
		},
	}

	val, err := opts.ParamAsString("missing")
	require.NoError(t, err)
	assert.Empty(t, val)
}

// TestServiceOptions_ParamAsString_NilParams verifies empty string for nil params
func TestServiceOptions_ParamAsString_NilParams(t *testing.T) {
	opts := ServiceOptions{}

	val, err := opts.ParamAsString("any")
	require.NoError(t, err)
	assert.Empty(t, val)
}

// TestServiceOptions_ParamAsString_WrongType verifies error for non-string value
func TestServiceOptions_ParamAsString_WrongType(t *testing.T) {
	opts := ServiceOptions{
		Params: map[string]interface{}{
			"key1": 123,
		},
	}

	val, err := opts.ParamAsString("key1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expecting string")
	assert.Empty(t, val)
}

// TestCompileServiceOptions_Success verifies successful option compilation
func TestCompileServiceOptions_Success(t *testing.T) {
	opts, err := CompileServiceOptions(
		WithNetwork("net1"),
		WithChannel("ch1"),
		WithNamespace("ns1"),
	)

	require.NoError(t, err)
	assert.Equal(t, "net1", opts.Network)
	assert.Equal(t, "ch1", opts.Channel)
	assert.Equal(t, "ns1", opts.Namespace)
}

// TestCompileServiceOptions_Empty verifies empty options compilation
func TestCompileServiceOptions_Empty(t *testing.T) {
	opts, err := CompileServiceOptions()

	require.NoError(t, err)
	assert.NotNil(t, opts)
	assert.Empty(t, opts.Network)
}

// TestWithNetwork verifies WithNetwork option sets network
func TestWithNetwork(t *testing.T) {
	opts := &ServiceOptions{}
	err := WithNetwork("mynetwork")(opts)

	require.NoError(t, err)
	assert.Equal(t, "mynetwork", opts.Network)
}

// TestWithChannel verifies WithChannel option sets channel
func TestWithChannel(t *testing.T) {
	opts := &ServiceOptions{}
	err := WithChannel("mychannel")(opts)

	require.NoError(t, err)
	assert.Equal(t, "mychannel", opts.Channel)
}

// TestWithNamespace verifies WithNamespace option sets namespace
func TestWithNamespace(t *testing.T) {
	opts := &ServiceOptions{}
	err := WithNamespace("mynamespace")(opts)

	require.NoError(t, err)
	assert.Equal(t, "mynamespace", opts.Namespace)
}

// TestWithPublicParameterFetcher verifies WithPublicParameterFetcher option
func TestWithPublicParameterFetcher(t *testing.T) {
	opts := &ServiceOptions{}
	fetcher := &mockPublicParamsFetcher{}
	err := WithPublicParameterFetcher(fetcher)(opts)

	require.NoError(t, err)
	assert.Equal(t, fetcher, opts.PublicParamsFetcher)
}

// TestWithPublicParameter verifies WithPublicParameter option sets params
func TestWithPublicParameter(t *testing.T) {
	opts := &ServiceOptions{}
	params := []byte("public params")
	err := WithPublicParameter(params)(opts)

	require.NoError(t, err)
	assert.Equal(t, params, opts.PublicParams)
}

// TestWithTMS verifies WithTMS option sets all TMS identifiers
func TestWithTMS(t *testing.T) {
	opts := &ServiceOptions{}
	err := WithTMS("net", "ch", "ns")(opts)

	require.NoError(t, err)
	assert.Equal(t, "net", opts.Network)
	assert.Equal(t, "ch", opts.Channel)
	assert.Equal(t, "ns", opts.Namespace)
}

// TestWithTMSID verifies WithTMSID option sets identifiers from TMSID
func TestWithTMSID(t *testing.T) {
	opts := &ServiceOptions{}
	tmsID := TMSID{
		Network:   "testnet",
		Channel:   "testch",
		Namespace: "testns",
	}
	err := WithTMSID(tmsID)(opts)

	require.NoError(t, err)
	assert.Equal(t, "testnet", opts.Network)
	assert.Equal(t, "testch", opts.Channel)
	assert.Equal(t, "testns", opts.Namespace)
}

// TestWithTMSIDPointer_NonNil verifies WithTMSIDPointer with non-nil pointer
func TestWithTMSIDPointer_NonNil(t *testing.T) {
	opts := &ServiceOptions{}
	tmsID := &TMSID{
		Network:   "testnet",
		Channel:   "testch",
		Namespace: "testns",
	}
	err := WithTMSIDPointer(tmsID)(opts)

	require.NoError(t, err)
	assert.Equal(t, "testnet", opts.Network)
	assert.Equal(t, "testch", opts.Channel)
	assert.Equal(t, "testns", opts.Namespace)
}

// TestWithTMSIDPointer_Nil verifies WithTMSIDPointer with nil pointer does nothing
func TestWithTMSIDPointer_Nil(t *testing.T) {
	opts := &ServiceOptions{
		Network: "existing",
	}
	err := WithTMSIDPointer(nil)(opts)

	require.NoError(t, err)
	assert.Equal(t, "existing", opts.Network)
}

// TestWithInitiator verifies WithInitiator option sets initiator
func TestWithInitiator(t *testing.T) {
	opts := &ServiceOptions{}
	initiator := &mockView{}
	err := WithInitiator(initiator)(opts)

	require.NoError(t, err)
	assert.Equal(t, initiator, opts.Initiator)
}

// TestWithDuration verifies WithDuration option sets duration
func TestWithDuration(t *testing.T) {
	opts := &ServiceOptions{}
	duration := 5 * time.Second
	err := WithDuration(duration)(opts)

	require.NoError(t, err)
	assert.Equal(t, duration, opts.Duration)
}

// Mock types for testing
type mockPublicParamsFetcher struct{}

func (m *mockPublicParamsFetcher) Fetch() ([]byte, error) {
	return nil, nil
}

type mockView struct{}

func (m *mockView) Call(view.Context) (interface{}, error) {
	return nil, nil
}

// TestCompileServiceOptions_WithError verifies error handling in option compilation
func TestCompileServiceOptions_WithError(t *testing.T) {
	expectedErr := errors.New("option error")
	errorOption := func(opts *ServiceOptions) error {
		return expectedErr
	}

	result, err := CompileServiceOptions(errorOption)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}
