/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package house

// House is a struct that contains a house
type House struct {
	LinearID  string
	Address   string
	Valuation uint64
}

// SetLinearID sets the linear id of the house
func (h *House) SetLinearID(id string) string {
	if len(h.LinearID) == 0 {
		h.LinearID = id
	}
	return h.LinearID
}
