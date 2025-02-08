/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package json

import (
	"bytes"
	"encoding/json"
)

// Unmarshal is json.Unmarshal with unknown fields disallowed.
func Unmarshal(data []byte, v interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

var (
	Marshal       = json.Marshal
	MarshalIndent = json.MarshalIndent
)
