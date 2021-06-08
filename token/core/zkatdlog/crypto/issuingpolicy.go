/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package crypto

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
)

type IssuingPolicy struct {
	Issuers       []*bn256.G1
	IssuersNumber int
	BitLength     int
}

func (ip *IssuingPolicy) Serialize() ([]byte, error) {
	return json.Marshal(ip)
}

func (ip *IssuingPolicy) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, ip)
}
