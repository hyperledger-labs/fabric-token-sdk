/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package translator

import (
	"crypto/sha256"

	"github.com/gobuffalo/packr/v2/file/resolver/encoding/hex"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

//go:generate counterfeiter -o mock/rwset.go -fake-name RWSet . RWSet

type KeyTranslator interface {
	CreateTokenRequestKey(id string) (string, error)
	CreateSetupKey() (string, error)
	CreateSetupHashKey() (string, error)
	CreateTokenKey(id string, index uint64, output []byte) (string, error)
	GetTransferMetadataSubKey(k string) (string, error)
	CreateSNKey(id string) (string, error)
	CreateIssueActionMetadataKey(key string) (string, error)
	// CreateTransferActionMetadataKey returns the transfer action metadata key built from the passed
	// transaction id, subkey, and index. Index is used to make sure the key is unique with the respect to the
	// token request this key appears.
	CreateTransferActionMetadataKey(key string) (string, error)
}

// RWSet interface, used to read from, and write to, a rwset.
type RWSet interface {
	SetState(namespace string, key string, value []byte) error
	GetState(namespace string, key string) ([]byte, error)
	DeleteState(namespace string, key string) error
}

// KeyVersion defines the supported key versions
type KeyVersion = int

const (
	// Any value, any version of the key would work
	Any KeyVersion = iota
	// VersionZero value,  version `zero` of the key
	VersionZero
)

// ExRWSet interface, used to manipulate the rwset in a more friendly way
type ExRWSet interface {
	// SetState adds a write entry to the rwset that write to given value to given key.
	SetState(key string, value []byte) error
	// GetState returns the value bound to the passed key
	GetState(key string) ([]byte, error)
	// DeleteState adds a write entry to the rwset that deletes the passed key
	DeleteState(key string) error
	// StateMustNotExist adds a read dependency that enforces that the passed key does not exist
	StateMustNotExist(key string) error
	// StateMustExist adds a read dependency that enforces that the passed key does exist
	StateMustExist(key string, version KeyVersion) error
}

type RWSetWrapper struct {
	RWSet     RWSet
	Namespace string
	TxID      string
}

func NewRWSetWrapper(RWSet RWSet, namespace string, txID string) *RWSetWrapper {
	return &RWSetWrapper{RWSet: RWSet, Namespace: namespace, TxID: txID}
}

func (w *RWSetWrapper) SetState(key string, value []byte) error {
	return w.RWSet.SetState(w.Namespace, key, value)
}

func (w *RWSetWrapper) GetState(key string) ([]byte, error) {
	return w.RWSet.GetState(w.Namespace, key)
}

func (w *RWSetWrapper) DeleteState(key string) error {
	return w.RWSet.DeleteState(w.Namespace, key)
}

func (w *RWSetWrapper) StateMustNotExist(key string) error {
	tr, err := w.RWSet.GetState(w.Namespace, key)
	if err != nil {
		return errors.Wrapf(err, "failed to read state [%s:%s] for [%s]", w.Namespace, key, w.TxID)
	}
	if len(tr) != 0 {
		return errors.Errorf("state [%s:%s] already exists for [%s]", w.Namespace, key, w.TxID)
	}
	return nil
}

func (w *RWSetWrapper) StateMustExist(key string, version KeyVersion) error {
	h, err := w.RWSet.GetState(w.Namespace, key)
	if err != nil {
		return errors.Wrapf(err, "failed to read state [%s:%s] for [%s]", w.Namespace, key, w.TxID)
	}
	if len(h) == 0 {
		return errors.Errorf("state [%s:%s] does not exist for [%s]", w.Namespace, key, w.TxID)
	}
	return nil
}

type HashedKeyTranslator struct {
	KT KeyTranslator
}

func (h *HashedKeyTranslator) CreateTokenRequestKey(id string) (string, error) {
	k, err := h.KT.CreateTokenRequestKey(id)
	if err != nil {
		return "", err
	}
	return h.hash(0, k)
}

func (h *HashedKeyTranslator) CreateSetupKey() (string, error) {
	k, err := h.KT.CreateSetupKey()
	if err != nil {
		return "", err
	}
	return h.hash(1, k)
}

func (h *HashedKeyTranslator) CreateSetupHashKey() (string, error) {
	k, err := h.KT.CreateSetupHashKey()
	if err != nil {
		return "", err
	}
	return h.hash(2, k)
}

func (h *HashedKeyTranslator) CreateTokenKey(id string, index uint64, output []byte) (string, error) {
	k, err := h.KT.CreateTokenKey(id, index, output)
	if err != nil {
		return "", err
	}
	return h.hash(3, k)
}

func (h *HashedKeyTranslator) GetTransferMetadataSubKey(k string) (string, error) {
	key, err := h.KT.GetTransferMetadataSubKey(k)
	if err != nil {
		return "", err
	}
	return h.hash(4, key)
}

func (h *HashedKeyTranslator) CreateSNKey(id string) (string, error) {
	k, err := h.KT.CreateSNKey(id)
	if err != nil {
		return "", err
	}
	return h.hash(5, k)
}

func (h *HashedKeyTranslator) CreateIssueActionMetadataKey(key string) (string, error) {
	k, err := h.KT.CreateIssueActionMetadataKey(key)
	if err != nil {
		return "", err
	}
	return h.hash(6, k)
}

func (h *HashedKeyTranslator) CreateTransferActionMetadataKey(key string) (string, error) {
	k, err := h.KT.CreateTransferActionMetadataKey(key)
	if err != nil {
		return "", err
	}
	return h.hash(7, k)
}

func (h *HashedKeyTranslator) hash(code byte, k string) (string, error) {
	hf := sha256.New()
	hf.Write([]byte{code})
	hf.Write([]byte(k))
	return hex.EncodeToString(hf.Sum(nil)), nil
}
