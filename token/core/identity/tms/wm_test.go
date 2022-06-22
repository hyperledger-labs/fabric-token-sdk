/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tms_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/tms"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/tms/mock"
	"github.com/stretchr/testify/assert"
)

//go:generate counterfeiter -o mock/viewcm.go -fake-name ViewCM . ViewCM
//go:generate counterfeiter -o mock/idemixss.go -fake-name IdemixSignerService . IdemixSignerService

// ViewConfigService is used to generate a mock implementation of the driver.ConfigService interface
type ViewConfigService interface {
	driver.ConfigService
}

// IdemixSignerService is used to generate a mock implementation of the idemix.SignerService interface.
type IdemixSignerService interface {
	idemix.SignerService
}

// TestRegisterIdentityIdemixGurvy tests the RegisterIdentity function on input an idemix identity
// created with fabric-ca using Gurvy.
func TestRegisterIdentityIdemixGurvy(t *testing.T) {
	lm, cm := initLocalMembership(t)

	// testdata/idemixGurvy* has been generated using fabric-ca (aa1040cefc5f4d5d7cc4e17c8b2f8dddaae9cea2) in the following way:
	// Server:
	//   Configuration file with `idemix.curve: gurvy.Bn254`
	//   ./fabric-ca-server start -b admin:adminpw
	// Client:
	//   export FABRIC_CA_CLIENT_HOME=$HOME/fabric-ca/clients/token
	//   ./fabric-ca-client enroll -u http://admin:adminpw@localhost:7054 --enrollment.type idemix --idemix.curve gurvy.Bn254

	// program the config manager
	cm.TranslatePathReturnsOnCall(0, "./testdata/idemixGurvy")
	cm.TranslatePathReturnsOnCall(1, "./testdata/idemixGurvy")
	cm.TranslatePathReturnsOnCall(2, "./testdata/idemixGurvy")
	cm.TranslatePathReturnsOnCall(3, "./testdata/idemixGurvy2")
	cm.TranslatePathReturnsOnCall(4, "./testdata/idemixGurvy2")
	cm.TranslatePathReturnsOnCall(5, "./testdata/idemixGurvy2")

	// register identities
	for _, path := range []string{"testdata/idemixGurvy", "testdata/idemixGurvy2"} {
		err := lm.RegisterIdentity("pinepple", "idemix:IdemixOrgMSP:BN254", path)
		assert.NoError(t, err, "Failed to register identity")

		err = lm.RegisterIdentity("pinepple", "idemix:IdemixOrgMSP:FP256BN_AMCL", path)
		assert.Errorf(t, err, "Should have failed to register identity")
		assert.Contains(t, err.Error(), "invalid issuer public key: some part of the public key is undefined")
	}
	assert.Equal(t, 6, cm.TranslatePathCallCount())
}

func initLocalMembership(t *testing.T) (*tms.LocalMembership, *mock.ConfigManager) {
	sp := registry.New()
	err := sp.RegisterService(&mock.DeserializerManager{})
	assert.NoError(t, err)
	viewCM := &mock.ViewCM{}
	viewCM.UnmarshalKeyReturns(nil)
	err = sp.RegisterService(viewCM)
	assert.NoError(t, err)
	sigService := &mock.IdemixSignerService{}
	err = sp.RegisterService(sigService)
	assert.NoError(t, err)

	kvs, err := kvs.New("memory", "default", sp)
	assert.NoError(t, err)
	err = sp.RegisterService(kvs)
	assert.NoError(t, err)
	cm := &mock.ConfigManager{}

	return tms.NewLocalMembership(sp, cm, nil, nil, nil), cm
}
