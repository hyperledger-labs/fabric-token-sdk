/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sig

import (
	"context"
	"runtime/debug"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	identity2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"go.uber.org/zap/zapcore"
)

var logger = logging.MustGetLogger()

type Storage interface {
	StoreSignerInfo(ctx context.Context, id, info []byte) error
	GetExistingSignerInfo(ctx context.Context, ids ...driver.Identity) ([]string, error)
	SignerInfoExists(ctx context.Context, id []byte) (bool, error)
	GetSignerInfo(ctx context.Context, identity []byte) ([]byte, error)
}

type VerifierEntry struct {
	Verifier   driver.Verifier
	DebugStack []byte
}

type SignerEntry struct {
	Signer     driver.Signer
	DebugStack []byte
}

type Service struct {
	sync      sync.RWMutex
	signers   map[string]SignerEntry
	verifiers map[string]VerifierEntry

	storage      Storage
	deserializer idriver.Deserializer
}

func NewService(deserializer idriver.Deserializer, storage Storage) *Service {
	return &Service{
		signers:      map[string]SignerEntry{},
		verifiers:    map[string]VerifierEntry{},
		deserializer: deserializer,
		storage:      storage,
	}
}

func (o *Service) RegisterSigner(ctx context.Context, identity driver.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error {
	if signer == nil {
		return errors.New("invalid signer, expected a valid instance")
	}

	idHash := identity.UniqueID()
	logger.DebugfContext(ctx, "register signer and verifier [%s]:[%s][%s]", idHash, logging.Identifier(signer), logging.Identifier(verifier))
	// First check with read lock
	o.sync.RLock()
	s, ok := o.signers[idHash]
	o.sync.RUnlock()
	if ok {
		logger.Warnf("another signer bound to [%s]:[%s][%s] from [%s]", identity, logging.Identifier(s), logging.Identifier(signer), string(s.DebugStack))
		return nil
	}

	// write lock
	o.sync.Lock()

	// check again the cache
	s, ok = o.signers[idHash]
	if ok {
		o.sync.Unlock()
		logger.Warnf("another signer bound to [%s]:[%s][%s] from [%s]", identity, logging.Identifier(s), logging.Identifier(signer), string(s.DebugStack))
		return nil
	}

	entry := SignerEntry{Signer: signer}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		entry.DebugStack = debug.Stack()
	}
	o.signers[idHash] = entry
	o.sync.Unlock()

	// store, if a failure happens then remove the entry
	logger.DebugfContext(ctx, "checks done, store signer info")
	if err := o.storage.StoreSignerInfo(ctx, identity, signerInfo); err != nil {
		o.deleteSigner(idHash)
		return errors.Wrap(err, "failed to store entry in storage for the passed signer")
	}

	if verifier != nil {
		// store verifier
		logger.DebugfContext(ctx, "checks done, store verifier")
		if err := o.RegisterVerifier(ctx, identity, verifier); err != nil {
			o.deleteSigner(idHash)
			return err
		}
	}

	logger.DebugfContext(ctx, "register signer and verifier [%s]:[%s][%s], done", idHash, logging.Identifier(signer), logging.Identifier(verifier))
	return nil
}

func (o *Service) RegisterVerifier(ctx context.Context, identity driver.Identity, verifier driver.Verifier) error {
	if verifier == nil {
		return errors.New("invalid verifier, expected a valid instance")
	}

	// First check with read lock
	idHash := identity.UniqueID()
	o.sync.RLock()
	v, ok := o.verifiers[idHash]
	o.sync.RUnlock()
	if ok {
		logger.Warnf("another verifier bound to [%s]:[%s][%s] from [%s]", idHash, logging.Identifier(v), logging.Identifier(verifier), string(v.DebugStack))
		return nil
	}

	// write lock
	o.sync.Lock()

	// check again
	v, ok = o.verifiers[idHash]
	if ok {
		o.sync.Unlock()
		logger.Warnf("another verifier bound to [%s]:[%s][%s] from [%s]", idHash, logging.Identifier(v), logging.Identifier(verifier), string(v.DebugStack))
		return nil
	}

	entry := VerifierEntry{Verifier: verifier}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		entry.DebugStack = debug.Stack()
	}
	o.verifiers[idHash] = entry
	o.sync.Unlock()

	logger.DebugfContext(ctx, "register verifier to [%s]:[%s]", idHash, logging.Identifier(verifier))
	return nil
}

func (o *Service) AreMe(ctx context.Context, identities ...driver.Identity) []string {
	logger.DebugfContext(ctx, "is me [%s]?", identities)
	idHashes := make([]string, len(identities))
	for i, id := range identities {
		idHashes[i] = id.UniqueID()
	}

	result := collections.NewSet[string]()
	notFound := make([]driver.Identity, 0)

	// check local cache
	o.sync.RLock()
	for _, id := range identities {
		if _, ok := o.signers[id.UniqueID()]; ok {
			logger.DebugfContext(ctx, "is me [%s]? yes, from cache", id)
			result.Add(id.UniqueID())
		} else {
			notFound = append(notFound, id)
		}
	}
	o.sync.RUnlock()

	if len(notFound) == 0 {
		return result.ToSlice()
	}

	// check storage
	found, err := o.storage.GetExistingSignerInfo(ctx, notFound...)
	if err != nil {
		logger.Errorf("failed checking if a signer exists [%s]", err)
		return result.ToSlice()
	}
	result.Add(found...)
	return result.ToSlice()
}

func (o *Service) IsMe(ctx context.Context, identity driver.Identity) bool {
	logger.DebugfContext(ctx, "is me [%s]?", identity)
	idHash := identity.UniqueID()

	// check local cache
	o.sync.RLock()
	_, ok := o.signers[idHash]
	o.sync.RUnlock()
	if ok {
		logger.DebugfContext(ctx, "is me [%s]? yes, from cache", identity)
		return true
	}

	// check storage
	logger.DebugfContext(ctx, "is me [%s]? ask the storage", identity)
	exists, err := o.storage.SignerInfoExists(ctx, identity)
	if err != nil {
		logger.Errorf("failed checking if a signer exists [%s]", err)
	}
	if exists {
		logger.DebugfContext(ctx, "is me [%s]? yes, from storage", identity)
		return true
	}
	return false
}

func (o *Service) GetSigner(ctx context.Context, identity driver.Identity) (driver.Signer, error) {
	idHash := identity.UniqueID()
	logger.DebugfContext(ctx, "get signer for [%s]", idHash)
	// check the cache
	o.sync.RLock()
	entry, ok := o.signers[idHash]
	o.sync.RUnlock()
	if ok {
		logger.DebugfContext(ctx, "signer for [%s] found", idHash)
		return entry.Signer, nil
	}
	o.sync.Lock()
	defer o.sync.Unlock()

	return o.getSigner(ctx, identity, idHash)
}

func (o *Service) getSigner(ctx context.Context, identity driver.Identity, idHash string) (driver.Signer, error) {
	// check again the cache
	entry, ok := o.signers[idHash]
	if ok {
		logger.DebugfContext(ctx, "signer for [%s] found", idHash)
		return entry.Signer, nil
	}

	logger.DebugfContext(ctx, "signer for [%s] not found, try to deserialize", idHash)
	// ask the deserializer
	signer, err := o.deserializeSigner(ctx, identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed deserializing identity for signer [%s]", identity)
	}
	entry = SignerEntry{Signer: signer}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		entry.DebugStack = debug.Stack()
	}
	o.signers[idHash] = entry
	if err := o.storage.StoreSignerInfo(ctx, identity, nil); err != nil {
		return nil, errors.Wrap(err, "failed to store entry in storage for the passed signer")
	}
	return entry.Signer, nil
}

func (o *Service) deserializeSigner(ctx context.Context, identity driver.Identity) (driver.Signer, error) {
	if o.deserializer == nil {
		return nil, errors.Errorf("cannot find signer for [%s], no deserializer set", identity)
	}
	var err error
	signer, err := o.deserializer.DeserializeSigner(ctx, identity)
	if err == nil {
		return signer, nil
	}

	// give it a second chance

	// is the identity wrapped in TypedIdentity?
	ro, err2 := identity2.UnmarshalTypedIdentity(identity)
	if err2 != nil {
		// No
		return nil, errors.Wrapf(err2, "failed to unmarshal raw owner for identity [%s] and failed deserialization [%s]", identity.String(), err)
	}

	// yes, check ro.Identity
	signer, err = o.getSigner(ctx, ro.Identity, ro.Identity.UniqueID())
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting signer for identity [%s]", ro.Identity)
	}
	return signer, nil
}

func (o *Service) GetSignerInfo(ctx context.Context, identity driver.Identity) ([]byte, error) {
	return o.storage.GetSignerInfo(ctx, identity)
}

func (o *Service) GetVerifier(ctx context.Context, identity driver.Identity) (driver.Verifier, error) {
	idHash := identity.UniqueID()

	// check cache
	o.sync.RLock()
	entry, ok := o.verifiers[idHash]
	o.sync.RUnlock()
	if ok {
		return entry.Verifier, nil
	}

	o.sync.Lock()
	defer o.sync.Unlock()

	// check cache again
	entry, ok = o.verifiers[idHash]
	if ok {
		return entry.Verifier, nil
	}

	// ask the deserializer
	if o.deserializer == nil {
		return nil, errors.Errorf("cannot find verifier for [%s], no deserializer set", identity)
	}
	var err error
	verifier, err := o.deserializer.DeserializeVerifier(ctx, identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed deserializing identity for verifier %v", identity)
	}

	// store entry
	entry = VerifierEntry{Verifier: verifier}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		entry.DebugStack = debug.Stack()
	}
	logger.DebugfContext(ctx, "add deserialized verifier for [%s]:[%s]", idHash, logging.Identifier(verifier))
	o.verifiers[idHash] = entry
	return verifier, nil
}

func (o *Service) deleteSigner(id string) {
	o.sync.Lock()
	defer o.sync.Unlock()
	delete(o.signers, id)
}
