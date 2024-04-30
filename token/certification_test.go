/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCertificationManager_NewCertificationRequest(t *testing.T) {
	certificationService := &mock.CertificationService{}

	manager := &CertificationManager{
		c: certificationService,
	}

	expectedRequest := []byte("certification_request")
	certificationService.NewCertificationRequestReturns(expectedRequest, nil)

	ids := []*token.ID{{TxId: "a_transaction", Index: 0}}
	request, err := manager.NewCertificationRequest(ids)

	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, request)
}

func TestCertificationManager_Certify(t *testing.T) {
	certificationService := &mock.CertificationService{}
	wallet := &CertifierWallet{}

	manager := &CertificationManager{
		c: certificationService,
	}

	expectedCertifications := [][]byte{[]byte("a_certification")}
	certificationService.CertifyReturns(expectedCertifications, nil)

	ids := []*token.ID{{TxId: "a_transaction", Index: 0}}
	tokens := [][]byte{[]byte("a_token")}
	request := []byte("request")
	certifications, err := manager.Certify(wallet, ids, tokens, request)

	assert.NoError(t, err)
	assert.Equal(t, expectedCertifications, certifications)
}

func TestCertificationManager_VerifyCertifications(t *testing.T) {
	certificationService := &mock.CertificationService{}

	manager := &CertificationManager{
		c: certificationService,
	}

	expectedCertifications := [][]byte{[]byte("a_certification")}
	certificationService.VerifyCertificationsReturns(expectedCertifications, nil)

	ids := []*token.ID{{TxId: "a_transaction", Index: 0}}
	certifications := [][]byte{[]byte("a_certification")}
	verifiedCertifications, err := manager.VerifyCertifications(ids, certifications)

	assert.NoError(t, err)
	assert.Equal(t, expectedCertifications, verifiedCertifications)
}

func TestCertificationClient_IsCertified(t *testing.T) {
	certificationClient := &mock.CertificationClient{}

	client := &CertificationClient{
		cc: certificationClient,
	}

	certificationClient.IsCertifiedReturns(true)

	id := &token.ID{TxId: "a_transaction", Index: 0}
	isCertified := client.IsCertified(id)

	assert.True(t, isCertified)
}

func TestCertificationClient_RequestCertification(t *testing.T) {
	certificationClient := &mock.CertificationClient{}

	client := &CertificationClient{
		cc: certificationClient,
	}

	certificationClient.RequestCertificationReturns(nil)

	ids := []*token.ID{{TxId: "a_transaction", Index: 0}}
	err := client.RequestCertification(ids...)

	assert.NoError(t, err)
}

func TestCertificationManager_NewCertificationRequest_Error(t *testing.T) {
	certificationService := &mock.CertificationService{}

	manager := &CertificationManager{
		c: certificationService,
	}

	certificationService.NewCertificationRequestReturns(nil, errors.New("mocked error"))

	ids := []*token.ID{{TxId: "a_transaction", Index: 0}}
	request, err := manager.NewCertificationRequest(ids)

	assert.Error(t, err)
	assert.Nil(t, request)
}

func TestCertificationClient_RequestCertification_Error(t *testing.T) {
	certificationClient := &mock.CertificationClient{}

	client := &CertificationClient{
		cc: certificationClient,
	}

	certificationClient.RequestCertificationReturns(errors.New("mocked error"))

	ids := []*token.ID{{TxId: "a_transaction", Index: 0}}
	err := client.RequestCertification(ids...)

	assert.Error(t, err)
}
