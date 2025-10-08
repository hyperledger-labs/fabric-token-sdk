/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorsement

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabricx/core/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
)

// rwSet interface, used to read from, and write to, a rwset.
type rwSet interface {
	SetState(namespace driver.Namespace, key driver.PKey, value []byte) error
	GetState(namespace driver.Namespace, key driver.PKey, opts ...driver.GetStateOpt) ([]byte, error)
	DeleteState(namespace driver.Namespace, key driver.PKey) error
	AddReadAt(ns driver.Namespace, key driver.PKey, version driver.RawVersion) error
}

type RWSetWrapper struct {
	RWSet     rwSet
	Namespace translator.Namespace
	TxID      translator.TxID
	ppVersion uint64
}

func NewRWSetWrapper(RWSet rwSet, namespace translator.Namespace, txID translator.TxID, ppVersion uint64) *RWSetWrapper {
	return &RWSetWrapper{RWSet: RWSet, Namespace: namespace, TxID: txID, ppVersion: ppVersion}
}

func (w *RWSetWrapper) SetState(key translator.Key, value translator.Value) error {
	return w.RWSet.SetState(w.Namespace, key, value)
}

func (w *RWSetWrapper) GetState(key translator.Key) (translator.Value, error) {
	return w.RWSet.GetState(w.Namespace, key)
}

func (w *RWSetWrapper) DeleteState(key translator.Key) error {
	return w.RWSet.DeleteState(w.Namespace, key)
}

func (w *RWSetWrapper) StateMustNotExist(key translator.Key) error {
	return w.RWSet.AddReadAt(w.Namespace, key, nil)
}

func (w *RWSetWrapper) StateMustExist(key translator.Key, version translator.KeyVersion) error {
	switch version {
	case translator.VersionZero:
		return w.RWSet.AddReadAt(w.Namespace, key, vault.Marshal(0))
	case translator.Latest:
		// TODO: AF
		return w.RWSet.AddReadAt(w.Namespace, key, vault.Marshal(w.ppVersion))
		// // When StateMustExist is called on VersionZero, Latest behaviour is used instead.
		// // This works under the assumption that keys are used only once.
		// h, err := w.RWSet.GetState(w.Namespace, key)
		// if err != nil {
		//	return errors.Wrapf(err, "failed to read state [%s:%s] for [%s]", w.Namespace, key, w.TxID)
		// }
		// if len(h) == 0 {
		//	return errors.Errorf("state [%s:%s] does not exist for [%s]", w.Namespace, key, w.TxID)
		// }
		// return nil
	default:
		return errors.Errorf("invalid version [%d]", version)
	}
}
