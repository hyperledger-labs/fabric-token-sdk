/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttxcc

import (
	"encoding/asn1"
	"sort"

	jsoniter "github.com/json-iterator/go"
)

func Marshal(v interface{}) ([]byte, error) {
	return jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(v)
}

func Unmarshal(data []byte, v interface{}) error {
	return jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal(data, v)
}

func MarshalMeta(v map[string][]byte) ([]byte, error) {
	metaSer := metaSer{
		Keys: make([]string, len(v)),
		Vals: make([][]byte, len(v)),
	}

	i := 0
	for k := range v {
		metaSer.Keys[i] = k
		i++
	}
	i = 0
	sort.Strings(metaSer.Keys)
	for _, key := range metaSer.Keys {
		metaSer.Vals[i] = v[key]
		i++
	}
	return asn1.Marshal(metaSer)
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
