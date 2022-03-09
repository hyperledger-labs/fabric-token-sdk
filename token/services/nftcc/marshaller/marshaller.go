/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package marshaller

import "encoding/json"

// Marshal marshals the given object
func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal unmarshalls the given bytes into the given object
func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
