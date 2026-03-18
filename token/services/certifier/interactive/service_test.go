/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interactive

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: We use counterfeiter-generated mocks that are in the same package (not a subpackage).
// This avoids import cycles that would occur if mocks were in interactive/mock, since the
// Backend interface references *CertificationRequest from the interactive package.

// Test construction of a new CertificationService
func TestNewCertificationService(t *testing.T) {
	responderRegistry := &ResponderRegistryMock{}
	backend := &BackendMock{}
	mp := &disabled.Provider{}

	service := NewCertificationService(responderRegistry, mp, backend)

	assert.NotNil(t, service)
	assert.Equal(t, responderRegistry, service.ResponderRegistry)
	assert.Equal(t, backend, service.backend)
	assert.NotNil(t, service.wallets)
	assert.NotNil(t, service.metrics)
	assert.Empty(t, service.wallets)
}

// Test calling Start() for a new CertificationService
func TestCertificationService_Start_Success(t *testing.T) {
	responderRegistry := &ResponderRegistryMock{}
	backend := &BackendMock{}
	mp := &disabled.Provider{}

	service := NewCertificationService(responderRegistry, mp, backend)

	err := service.Start()
	require.NoError(t, err, "Start should succeed when RegisterResponder succeeds")

	// Verify RegisterResponder was called once
	assert.Equal(t, 1, responderRegistry.RegisterResponderCallCount())
}

// Test that calling Start() for a new CertificationService
// doesn't fail even when the RegisterResponder fails
func TestCertificationService_Start_RegistrationError(t *testing.T) {
	expectedErr := errors.New("registration failed")
	responderRegistry := &ResponderRegistryMock{}
	responderRegistry.RegisterResponderReturns(expectedErr)

	backend := &BackendMock{}
	mp := &disabled.Provider{}

	service := NewCertificationService(responderRegistry, mp, backend)

	err := service.Start()
	// Note: The Start() method returns nil even if RegisterResponder fails
	// This appears to be intentional behavior (logs error but doesn't fail)
	require.NoError(t, err)

	// Verify RegisterResponder was called once
	assert.Equal(t, 1, responderRegistry.RegisterResponderCallCount())
}

// Test that the String summary of a CertificationRequest is as expected
func TestCertificationRequest_String(t *testing.T) {
	cr := &CertificationRequest{
		Network:   "test-network",
		Channel:   "test-channel",
		Namespace: "test-namespace",
		IDs:       nil,
		Request:   []byte("test-request"),
	}

	str := cr.String()
	assert.NotEmpty(t, str, "String() should return non-empty string")
	assert.Contains(t, str, "CertificationRequest")
	assert.Contains(t, str, "test-channel")
	assert.Contains(t, str, "test-namespace")
}

// Test the construction of a new CertificationRequestView
func TestNewCertificationRequestView(t *testing.T) {
	channel := "test-channel"
	namespace := "test-namespace"
	certifier := view.Identity([]byte("certifier-identity"))
	ids := []*token.ID{
		{TxId: "tx1", Index: 0},
		{TxId: "tx2", Index: 1},
	}

	v := NewCertificationRequestView(channel, namespace, certifier, ids...)

	assert.NotNil(t, v)
	assert.Equal(t, channel, v.channel)
	assert.Equal(t, namespace, v.ns)
	assert.Equal(t, certifier, v.certifier)
	assert.Equal(t, ids, v.ids)
}
