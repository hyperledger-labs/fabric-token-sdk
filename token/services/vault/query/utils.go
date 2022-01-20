/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package query

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/cache/secondcache"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var (
	cache = secondcache.New(10000)
)

func hash(v []byte) string {
	if len(v) == 0 {
		return ""
	}
	hash := sha256.New()
	n, err := hash.Write(v)
	if n != len(v) {
		panic("hash failure")
	}
	if err != nil {
		panic(err)
	}
	digest := hash.Sum(nil)
	return base64.StdEncoding.EncodeToString(digest)
}

func UnmarshallFabtoken(raw []byte) (*token2.Token, error) {
	k := hash(raw)
	if v, ok := cache.Get(k); ok {
		return v.(*token2.Token), nil
	}
	v := &token2.Token{}
	err := json.Unmarshal(raw, v)
	if err != nil {
		return nil, err
	}
	cache.Add(k, v)

	return v, nil
}

func UnmarshallIssuedToken(raw []byte) (*token2.IssuedToken, error) {
	t := &token2.IssuedToken{}
	err := json.Unmarshal(raw, t)
	if err != nil {
		return nil, err
	}
	return t, nil
}
