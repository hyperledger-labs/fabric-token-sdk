/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package certification

import (
	"strconv"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type Channel interface {
	Name() string
	Vault() *fabric.Vault
}

type Storage struct {
	sp        view.ServiceProvider
	channel   Channel
	namespace string
}

func NewStorage(sp view.ServiceProvider, channel Channel, namespace string) *Storage {
	return &Storage{sp: sp, channel: channel, namespace: namespace}
}

func (v *Storage) Exists(id *token.Id) bool {
	k := kvs.CreateCompositeKeyOrPanic(
		"token-sdk.certifier.certification",
		[]string{
			v.channel.Name(),
			v.namespace,
			id.TxId,
			strconv.FormatUint(uint64(id.Index), 10),
		},
	)
	return kvs.GetService(v.sp).Exists(k)
}

func (v *Storage) Store(certifications map[*token.Id][]byte) error {
	for id, certification := range certifications {
		k := kvs.CreateCompositeKeyOrPanic(
			"token-sdk.certifier.certification",
			[]string{
				v.channel.Name(),
				v.namespace,
				id.TxId,
				strconv.FormatUint(uint64(id.Index), 10),
			},
		)
		if err := kvs.GetService(v.sp).Put(k, certification); err != nil {
			return err
		}
	}
	return nil
}

func (v *Storage) Get(ids []*token.Id, callback func(*token.Id, []byte) error) error {
	for _, id := range ids {
		k := kvs.CreateCompositeKeyOrPanic(
			"token-sdk.certifier.certification",
			[]string{
				v.channel.Name(),
				v.namespace,
				id.TxId,
				strconv.FormatUint(uint64(id.Index), 10),
			},
		)
		var certification []byte
		if err := kvs.GetService(v.sp).Get(k, &certification); err != nil {
			return errors.WithMessagef(err, "failed getting certification from storage for [%s]", k)
		}
		if err := callback(id, certification); err != nil {
			return errors.WithMessagef(err, "failed call back for [%s]", k)
		}
	}
	return nil
}
