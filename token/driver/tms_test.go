/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTMSID_String tests the string representation of TMSID
func TestTMSID_String(t *testing.T) {
	tmsid := TMSID{
		Network:   "network1",
		Channel:   "channel1",
		Namespace: "namespace1",
	}

	result := tmsid.String()
	assert.Equal(t, "network1,channel1,namespace1", result)
}

// TestTMSID_String_EmptyFields tests string representation with empty fields
func TestTMSID_String_EmptyFields(t *testing.T) {
	tmsid := TMSID{
		Network:   "",
		Channel:   "",
		Namespace: "",
	}

	result := tmsid.String()
	assert.Equal(t, ",,", result)
}

// TestTMSID_Equal tests equality comparison of TMSIDs
func TestTMSID_Equal(t *testing.T) {
	tmsid1 := TMSID{
		Network:   "network1",
		Channel:   "channel1",
		Namespace: "namespace1",
	}

	tmsid2 := TMSID{
		Network:   "network1",
		Channel:   "channel1",
		Namespace: "namespace1",
	}

	assert.True(t, tmsid1.Equal(tmsid2))
	assert.True(t, tmsid2.Equal(tmsid1))
}

// TestTMSID_Equal_DifferentNetwork tests inequality when networks differ
func TestTMSID_Equal_DifferentNetwork(t *testing.T) {
	tmsid1 := TMSID{
		Network:   "network1",
		Channel:   "channel1",
		Namespace: "namespace1",
	}

	tmsid2 := TMSID{
		Network:   "network2",
		Channel:   "channel1",
		Namespace: "namespace1",
	}

	assert.False(t, tmsid1.Equal(tmsid2))
}

// TestTMSID_Equal_DifferentChannel tests inequality when channels differ
func TestTMSID_Equal_DifferentChannel(t *testing.T) {
	tmsid1 := TMSID{
		Network:   "network1",
		Channel:   "channel1",
		Namespace: "namespace1",
	}

	tmsid2 := TMSID{
		Network:   "network1",
		Channel:   "channel2",
		Namespace: "namespace1",
	}

	assert.False(t, tmsid1.Equal(tmsid2))
}

// TestTMSID_Equal_DifferentNamespace tests inequality when namespaces differ
func TestTMSID_Equal_DifferentNamespace(t *testing.T) {
	tmsid1 := TMSID{
		Network:   "network1",
		Channel:   "channel1",
		Namespace: "namespace1",
	}

	tmsid2 := TMSID{
		Network:   "network1",
		Channel:   "channel1",
		Namespace: "namespace2",
	}

	assert.False(t, tmsid1.Equal(tmsid2))
}

// TestServiceOptions_String tests the string representation of ServiceOptions
func TestServiceOptions_String(t *testing.T) {
	opts := ServiceOptions{
		Network:   "network1",
		Channel:   "channel1",
		Namespace: "namespace1",
	}

	result := opts.String()
	assert.Equal(t, "network1,channel1,namespace1", result)
}

// TestServiceOptions_String_EmptyFields tests string representation with empty fields
func TestServiceOptions_String_EmptyFields(t *testing.T) {
	opts := ServiceOptions{}

	result := opts.String()
	assert.Equal(t, ",,", result)
}

// TestServiceOptions_ParamAsString tests retrieving string parameter
func TestServiceOptions_ParamAsString(t *testing.T) {
	opts := ServiceOptions{
		Params: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
	}

	value, err := opts.ParamAsString("key1")
	require.NoError(t, err)
	assert.Equal(t, "value1", value)

	value, err = opts.ParamAsString("key2")
	require.NoError(t, err)
	assert.Equal(t, "value2", value)
}

// TestServiceOptions_ParamAsString_NilParams tests behavior with nil params map
func TestServiceOptions_ParamAsString_NilParams(t *testing.T) {
	opts := ServiceOptions{
		Params: nil,
	}

	value, err := opts.ParamAsString("key1")
	require.NoError(t, err)
	assert.Empty(t, value)
}

// TestServiceOptions_ParamAsString_KeyNotFound tests behavior when key doesn't exist
func TestServiceOptions_ParamAsString_KeyNotFound(t *testing.T) {
	opts := ServiceOptions{
		Params: map[string]interface{}{
			"key1": "value1",
		},
	}

	value, err := opts.ParamAsString("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, value)
}

// TestServiceOptions_ParamAsString_WrongType tests error when value is not a string
func TestServiceOptions_ParamAsString_WrongType(t *testing.T) {
	opts := ServiceOptions{
		Params: map[string]interface{}{
			"key1": 123,
			"key2": true,
			"key3": []byte("bytes"),
		},
	}

	_, err := opts.ParamAsString("key1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expecting string")

	_, err = opts.ParamAsString("key2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expecting string")

	_, err = opts.ParamAsString("key3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expecting string")
}

// TestServiceOptions_ParamAsString_EmptyString tests retrieving empty string value
func TestServiceOptions_ParamAsString_EmptyString(t *testing.T) {
	opts := ServiceOptions{
		Params: map[string]interface{}{
			"key1": "",
		},
	}

	value, err := opts.ParamAsString("key1")
	require.NoError(t, err)
	assert.Empty(t, value)
}

// TestTMSID_ZeroValue tests the zero value of TMSID
func TestTMSID_ZeroValue(t *testing.T) {
	var tmsid TMSID
	assert.Empty(t, tmsid.Network)
	assert.Empty(t, tmsid.Channel)
	assert.Empty(t, tmsid.Namespace)
	assert.Equal(t, ",,", tmsid.String())
}

// TestTMSID_StructFields tests that TMSID has the expected fields
func TestTMSID_StructFields(t *testing.T) {
	tmsid := TMSID{
		Network:   "test-network",
		Channel:   "test-channel",
		Namespace: "test-namespace",
	}

	// Verify all fields are accessible and have correct values
	assert.Equal(t, "test-network", tmsid.Network)
	assert.Equal(t, "test-channel", tmsid.Channel)
	assert.Equal(t, "test-namespace", tmsid.Namespace)
}

// TestServiceOptions_AllFields tests ServiceOptions with all fields populated
func TestServiceOptions_AllFields(t *testing.T) {
	publicParams := []byte("test-params")
	params := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	opts := ServiceOptions{
		Network:             "test-network",
		Channel:             "test-channel",
		Namespace:           "test-namespace",
		PublicParamsFetcher: nil, // Interface, can't instantiate without implementation
		PublicParams:        publicParams,
		Params:              params,
	}

	// Verify all fields are accessible
	assert.Equal(t, "test-network", opts.Network)
	assert.Equal(t, "test-channel", opts.Channel)
	assert.Equal(t, "test-namespace", opts.Namespace)
	assert.Nil(t, opts.PublicParamsFetcher)
	assert.Equal(t, publicParams, opts.PublicParams)
	assert.Equal(t, params, opts.Params)
}

// TestServiceOptions_ZeroValue tests the zero value of ServiceOptions
func TestServiceOptions_ZeroValue(t *testing.T) {
	var opts ServiceOptions
	assert.Empty(t, opts.Network)
	assert.Empty(t, opts.Channel)
	assert.Empty(t, opts.Namespace)
	assert.Nil(t, opts.PublicParamsFetcher)
	assert.Nil(t, opts.PublicParams)
	assert.Nil(t, opts.Params)
}

// TestServiceOptions_ParamAsString_MultipleTypes tests various non-string types
func TestServiceOptions_ParamAsString_MultipleTypes(t *testing.T) {
	opts := ServiceOptions{
		Params: map[string]interface{}{
			"int":     42,
			"float":   3.14,
			"bool":    false,
			"slice":   []string{"a", "b"},
			"map":     map[string]string{"k": "v"},
			"struct":  struct{ X int }{X: 1},
			"pointer": &struct{ Y string }{Y: "test"},
		},
	}

	// All non-string types should return an error
	for key := range opts.Params {
		_, err := opts.ParamAsString(key)
		require.Error(t, err, "Expected error for key: %s", key)
		assert.Contains(t, err.Error(), "expecting string", "Error message should mention 'expecting string' for key: %s", key)
	}
}

// TestServiceOptions_PublicParams tests PublicParams field
func TestServiceOptions_PublicParams(t *testing.T) {
	testCases := []struct {
		name   string
		params []byte
	}{
		{"nil params", nil},
		{"empty params", []byte{}},
		{"small params", []byte("small")},
		{"large params", make([]byte, 1024)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := ServiceOptions{
				PublicParams: tc.params,
			}
			assert.Equal(t, tc.params, opts.PublicParams)
		})
	}
}

// TestTMSID_Equal_SelfComparison tests that a TMSID equals itself
func TestTMSID_Equal_SelfComparison(t *testing.T) {
	tmsid := TMSID{
		Network:   "network",
		Channel:   "channel",
		Namespace: "namespace",
	}
	assert.True(t, tmsid.Equal(tmsid))
}

// TestTMSID_Equal_AllEmpty tests equality of two empty TMSIDs
func TestTMSID_Equal_AllEmpty(t *testing.T) {
	tmsid1 := TMSID{}
	tmsid2 := TMSID{}
	assert.True(t, tmsid1.Equal(tmsid2))
}

// TestServiceOptions_ParamsMap tests various Params map scenarios
func TestServiceOptions_ParamsMap(t *testing.T) {
	testCases := []struct {
		name   string
		params map[string]interface{}
	}{
		{"nil map", nil},
		{"empty map", map[string]interface{}{}},
		{"single entry", map[string]interface{}{"key": "value"}},
		{"multiple entries", map[string]interface{}{
			"string": "value",
			"int":    42,
			"bool":   true,
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := ServiceOptions{
				Params: tc.params,
			}

			if tc.params == nil {
				assert.Nil(t, opts.Params)
			} else {
				assert.Len(t, opts.Params, len(tc.params))
			}
		})
	}
}
