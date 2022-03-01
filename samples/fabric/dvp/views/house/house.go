/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package house

type House struct {
	LinearID  string
	Address   string
	Valuation uint64
}

func (h *House) SetLinearID(id string) string {
	if len(h.LinearID) == 0 {
		h.LinearID = id
	}
	return h.LinearID
}
