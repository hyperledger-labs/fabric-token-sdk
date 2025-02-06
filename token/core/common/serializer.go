/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import "github.com/hyperledger-labs/fabric-token-sdk/token/driver"

type Serializer struct {
}

func NewSerializer() *Serializer {
	return &Serializer{}
}

func (s Serializer) MarshalTokenRequestToSign(anchor string, request *driver.TokenRequest, meta *driver.TokenRequestMetadata) ([]byte, error) {
	return request.MarshalToMessageToSign([]byte(anchor))
}

func (s Serializer) MarshalTokenRequestToAuditSign(anchor string, request *driver.TokenRequest, meta *driver.TokenRequestMetadata) ([]byte, error) {
	return request.MarshalToMessageToSign([]byte(anchor))
}
