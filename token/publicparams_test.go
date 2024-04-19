package token

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/stretchr/testify/assert"
)

func TestPublicParameters_Precision(t *testing.T) {
	pp := &PublicParameters{
		PublicParameters: &mock.PublicParameters{},
	}

	mockPP := pp.PublicParameters.(*mock.PublicParameters)
	mockPP.PrecisionReturns(uint64(6))

	precision := pp.Precision()

	assert.Equal(t, uint64(6), precision)
}

func TestPublicParameters_CertificationDriver(t *testing.T) {
	pp := &PublicParameters{
		PublicParameters: &mock.PublicParameters{},
	}

	mockPP := pp.PublicParameters.(*mock.PublicParameters)
	mockPP.CertificationDriverReturns("my_certification_driver")

	certDriver := pp.CertificationDriver()

	assert.Equal(t, "my_certification_driver", certDriver)
}

func TestPublicParameters_GraphHiding(t *testing.T) {
	pp := &PublicParameters{
		PublicParameters: &mock.PublicParameters{},
	}

	mockPP := pp.PublicParameters.(*mock.PublicParameters)
	mockPP.GraphHidingReturns(true)

	graphHiding := pp.GraphHiding()

	assert.True(t, graphHiding)
}

func TestPublicParameters_TokenDataHiding(t *testing.T) {
	pp := &PublicParameters{
		PublicParameters: &mock.PublicParameters{},
	}

	mockPP := pp.PublicParameters.(*mock.PublicParameters)
	mockPP.TokenDataHidingReturns(false)

	tokenDataHiding := pp.TokenDataHiding()

	assert.False(t, tokenDataHiding)
}

func TestPublicParameters_MaxTokenValue(t *testing.T) {
	pp := &PublicParameters{
		PublicParameters: &mock.PublicParameters{},
	}

	mockPP := pp.PublicParameters.(*mock.PublicParameters)
	mockPP.MaxTokenValueReturns(uint64(1000))

	maxTokenValue := pp.MaxTokenValue()

	assert.Equal(t, uint64(1000), maxTokenValue)
}

func TestPublicParameters_Serialize(t *testing.T) {
	pp := &PublicParameters{
		PublicParameters: &mock.PublicParameters{},
	}

	mockPP := pp.PublicParameters.(*mock.PublicParameters)
	mockPP.SerializeReturns([]byte("serialized_data"), nil)

	serializedData, err := pp.Serialize()

	assert.NoError(t, err)
	assert.Equal(t, []byte("serialized_data"), serializedData)
}

func TestPublicParameters_Identifier(t *testing.T) {
	pp := &PublicParameters{
		PublicParameters: &mock.PublicParameters{},
	}

	mockPP := pp.PublicParameters.(*mock.PublicParameters)
	mockPP.IdentifierReturns("my_identifier")

	identifier := pp.Identifier()

	assert.Equal(t, "my_identifier", identifier)
}

func TestPublicParameters_Auditors(t *testing.T) {
	pp := &PublicParameters{
		PublicParameters: &mock.PublicParameters{},
	}

	mockPP := pp.PublicParameters.(*mock.PublicParameters)
	mockPP.AuditorsReturns([]view.Identity{[]byte("auditor1"), []byte("auditor2")})

	auditors := pp.Auditors()

	expectedAuditors := []view.Identity{[]byte("auditor1"), []byte("auditor2")}
	assert.Equal(t, expectedAuditors, auditors)

}

func TestPublicParametersManager_PublicParameters(t *testing.T) {
	ppm := &PublicParametersManager{
		ppm: &mock.PublicParamsManager{},
	}

	mockPPM := ppm.ppm.(*mock.PublicParamsManager)
	mockPPM.PublicParametersReturns(&mock.PublicParameters{})

	pp := ppm.PublicParameters()

	assert.NotNil(t, pp)

}

func TestPublicParametersManager_PublicParameters_Nil(t *testing.T) {
	ppm := &PublicParametersManager{
		ppm: &mock.PublicParamsManager{},
	}
	
	mockPPM := ppm.ppm.(*mock.PublicParamsManager)
	mockPPM.PublicParametersReturns(nil)
	pp := ppm.PublicParameters()
	assert.Nil(t, pp)
}
