/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package crypto

import (
	"encoding/json"

	"github.com/IBM/mathlib"
)

type IssuingPolicy struct {
	Issuers       []*math.G1
	IssuersNumber int
	BitLength     int
}

func (ip *IssuingPolicy) Serialize() ([]byte, error) {
	return json.Marshal(ip)
}

func (ip *IssuingPolicy) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, ip)
}
