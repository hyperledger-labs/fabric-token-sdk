/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interactive

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	token "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/stretchr/testify/require"
)

// TestDriver_NewCertificationService_BackendFactoryError verifies that when
// the BackendFactory returns an error, NewCertificationService propagates it
// and leaves CertificationService nil.
func TestDriver_NewCertificationService_BackendFactoryError(t *testing.T) {
	factoryErr := errors.New("backend init failed")

	d := NewDriver(
		func(_ *token.ManagementService, _ string) (Backend, error) {
			return nil, factoryErr
		},
		nil, nil, &fakeViewManager{},
		&ResponderRegistryMock{},
		&disabled.Provider{},
	)

	_, err := d.NewCertificationService(nil, "wallet")
	require.Error(t, err)
	require.ErrorIs(t, err, factoryErr)
	require.Nil(t, d.CertificationService, "service must not be set after factory failure")
}
