/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

import (
	"strconv"

	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/kvs"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type KVS = kvs2.KVS

type CertificationStorage struct {
	kvs   KVS
	tmsID token.TMSID
}

func NewCertificationStorage(kvs KVS, tmsID token.TMSID) *CertificationStorage {
	return &CertificationStorage{kvs: kvs, tmsID: tmsID}
}

func (v *CertificationStorage) Exists(id *token2.ID) bool {
	return v.kvs.Exists(v.certificationID(id))
}

func (v *CertificationStorage) Store(certifications map[*token2.ID][]byte) error {
	for id, certification := range certifications {
		if err := v.kvs.Put(v.certificationID(id), certification); err != nil {
			return err
		}
	}
	return nil
}

func (v *CertificationStorage) Get(ids []*token2.ID, callback func(*token2.ID, []byte) error) error {
	for _, id := range ids {
		k := v.certificationID(id)
		var certification []byte
		if err := v.kvs.Get(k, &certification); err != nil {
			return errors.WithMessagef(err, "failed getting certification from storage for [%s]", k)
		}
		if err := callback(id, certification); err != nil {
			return errors.WithMessagef(err, "failed call back for [%s]", k)
		}
	}
	return nil
}

func (v *CertificationStorage) certificationID(id *token2.ID) string {
	return kvs.CreateCompositeKeyOrPanic(
		"token-sdk.certifier.certification",
		[]string{
			v.tmsID.Network,
			v.tmsID.Channel,
			v.tmsID.Namespace,
			id.TxId,
			strconv.FormatUint(id.Index, 10),
		},
	)
}
