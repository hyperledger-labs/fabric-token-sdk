/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/state"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type House struct {
	Address   string
	Valuation uint64

	LinearID string
	Owner    view.Identity
}

func (h *House) SetLinearID(id string) string {
	if len(h.LinearID) == 0 {
		h.LinearID = id
	}
	return h.LinearID
}

func (h *House) Owners() state.Identities {
	return []view.Identity{h.Owner}
}

func (h House) ToBytes() []byte {
	raw, err := json.Marshal(h)
	if err != nil {
		panic("struct should be able to marshal itself")
	}
	return raw
}

func (h *House) FromBytes(raw []byte) error {
	return json.Unmarshal(raw, h)
}
