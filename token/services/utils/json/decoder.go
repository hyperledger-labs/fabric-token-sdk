/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package json

import (
	"bytes"
	"encoding/json"
)

// UnmarshalWithDisallowUnknownFields is json.Unmarshal with unknown fields disallowed.
func UnmarshalWithDisallowUnknownFields(data []byte, v interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}
