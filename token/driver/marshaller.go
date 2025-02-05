/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"encoding/asn1"

	jsoniter "github.com/json-iterator/go"
)

func Unmarshal(data []byte, v interface{}) error {
	return jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal(data, v)
}

func UnmarshalMeta(raw []byte) (map[string][]byte, error) {
	var metaSer metaSer
	_, err := asn1.Unmarshal(raw, &metaSer)
	if err != nil {
		return nil, err
	}
	v := make(map[string][]byte, len(metaSer.Keys))
	for i, k := range metaSer.Keys {
		v[k] = metaSer.Vals[i]
	}
	return v, nil
}

type metaSer struct {
	Keys []string
	Vals [][]byte
}
