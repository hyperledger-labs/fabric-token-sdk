/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package processor

import "encoding/json"

func Marshal(o interface{}) ([]byte, error) {
	data, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func Unmarshal(raw []byte, o interface{}) error {
	return json.Unmarshal(raw, o)
}
