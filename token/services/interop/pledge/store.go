/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type store interface {
	Exists(id string) bool
	Put(id string, state interface{}) error
	Get(id string, state interface{}) error
	Delete(id string) error
	GetByPartialCompositeID(prefix string, attrs []string) (kvs.Iterator, error)
}

type VaultStore struct {
	store store
}

func Vault(sf view.ServiceProvider) *VaultStore {
	return &VaultStore{
		store: kvs.GetService(sf),
	}
}

func (ps *VaultStore) Store(info *Info) error {
	raw, err := info.Bytes()
	if err != nil {
		return errors.Wrapf(err, "failed marshalling info to raw")
	}
	key, err := kvs.CreateCompositeKey(
		"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge",
		[]string{
			"pledge",
			"info",
			hash.Hashable(raw).String(),
		},
	)
	if err != nil {
		return errors.Wrapf(err, "failed creating key for info [%v]", info)
	}
	return ps.store.Put(
		key,
		info,
	)
}

func (ps *VaultStore) PledgeByTokenID(tokenID *token.ID) ([]*Info, error) {
	if tokenID == nil {
		return nil, errors.Errorf("passed nil token id")
	}
	it, err := ps.store.GetByPartialCompositeID(
		"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge",
		[]string{
			"pledge",
			"info",
		},
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting iterator over pledges")
	}

	var res []*Info
	for it.HasNext() {
		var info *Info
		if _, err := it.Next(&info); err != nil {
			return nil, errors.Wrapf(err, "failed getting next pledge info")
		}
		if info.TokenID.TxId == tokenID.TxId && info.TokenID.Index == tokenID.Index {
			res = append(res, info)
		}
	}

	return res, nil
}

func (ps *VaultStore) Delete(pledges []*Info) error {
	for _, info := range pledges {
		raw, err := info.Bytes()
		if err != nil {
			return errors.Wrapf(err, "failed marshalling info to raw")
		}
		key, err := kvs.CreateCompositeKey(
			"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge",
			[]string{
				"pledge",
				"info",
				hash.Hashable(raw).String(),
			},
		)
		if err != nil {
			return errors.Wrapf(err, "failed creating key for info [%v]", info)
		}
		if err := ps.store.Delete(key); err != nil {
			return errors.WithMessagef(err, "failed deleting [%s]", key)
		}
	}
	return nil
}
