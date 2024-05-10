/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sig

import (
	"reflect"
	"runtime/debug"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = flogging.MustGetLogger("token-sdk.services.identity.sig")

type Storage interface {
	StoreIdentityData(id []byte, identityAudit []byte, tokenMetadata []byte, tokenMetadataAudit []byte) error
	GetAuditInfo(id []byte) ([]byte, error)
	StoreSignerInfo(id, info []byte) error
	SignerInfoExists(id []byte) (bool, error)
	GetSignerInfo(identity []byte) ([]byte, error)
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
	signers      map[string]SignerEntry
	verifiers    map[string]VerifierEntry
	deserializer Deserializer
	viewsSync    sync.RWMutex
	storage      Storage
}

func NewService(deserializer Deserializer, storage Storage) *Service {
	return &Service{
		signers:      map[string]SignerEntry{},
		verifiers:    map[string]VerifierEntry{},
		deserializer: deserializer,
		storage:      storage,
	}
}

func (o *Service) RegisterSigner(identity view.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error {
	if signer == nil {
		return errors.New("invalid signer, expected a valid instance")
	}

	idHash := identity.UniqueID()
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("register signer and verifier [%s]:[%s][%s]", idHash, GetIdentifier(signer), GetIdentifier(verifier))
	}
	o.viewsSync.RLock()
	s, ok := o.signers[idHash]
	o.viewsSync.RUnlock()
	if ok {
		logger.Warnf("another signer bound to [%s]:[%s][%s] from [%s]", identity, GetIdentifier(s), GetIdentifier(signer), string(s.DebugStack))
		return nil
	}

	o.viewsSync.Lock()
	entry := SignerEntry{Signer: signer}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		entry.DebugStack = debug.Stack()
	}

	o.signers[idHash] = entry
	if o.storage != nil {
		if err := o.storage.StoreSignerInfo(identity, signerInfo); err != nil {
			o.viewsSync.Unlock()
			return errors.Wrap(err, "failed to store entry in storage for the passed signer")
		}
	}
	o.viewsSync.Unlock()

	if verifier != nil {
		if err := o.RegisterVerifier(identity, verifier); err != nil {
			return err
		}
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("register signer and verifier [%s]:[%s][%s], done", idHash, GetIdentifier(signer), GetIdentifier(verifier))
	}
	return nil
}

func (o *Service) RegisterVerifier(identity view.Identity, verifier driver.Verifier) error {
	if verifier == nil {
		return errors.New("invalid verifier, expected a valid instance")
	}

	idHash := identity.UniqueID()
	o.viewsSync.Lock()
	v, ok := o.verifiers[idHash]
	o.viewsSync.Unlock()
	if ok {
		logger.Warnf("another verifier bound to [%s]:[%s][%s] from [%s]", idHash, GetIdentifier(v), GetIdentifier(verifier), string(v.DebugStack))
		return nil
	}

	entry := VerifierEntry{Verifier: verifier}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		entry.DebugStack = debug.Stack()
	}
	o.viewsSync.Lock()
	o.verifiers[idHash] = entry
	o.viewsSync.Unlock()
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("register verifier to [%s]:[%s]", idHash, GetIdentifier(verifier))
	}
	return nil
}

func (o *Service) IsMe(identity view.Identity) bool {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("is me [%s]?", identity)
	}
	idHash := identity.UniqueID()
	// check local cache
	o.viewsSync.RLock()
	_, ok := o.signers[idHash]
	o.viewsSync.RUnlock()
	if ok {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("is me [%s]? yes, from cache", identity)
		}
		return true
	}

	// check storage
	if o.storage != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("is me [%s]? ask the storage", identity)
		}
		exists, err := o.storage.SignerInfoExists(identity)
		if err != nil {
			logger.Errorf("failed checking if a signer exists [%s]", err)
		}
		if exists {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("is me [%s]? yes, from storage", identity)
			}
			return true
		}
	}
	// last chance, deserialize
	//signer, err := o.GetSigner(identity)
	//if err != nil {
	//	return false
	//}
	//return signer != nil
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("is me [%s]? no", identity)
	}
	return false
}

func (o *Service) GetSigner(identity view.Identity) (driver.Signer, error) {
	idHash := identity.UniqueID()
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get signer for [%s]", idHash)
	}
	// check the cache
	o.viewsSync.RLock()
	entry, ok := o.signers[idHash]
	o.viewsSync.RUnlock()
	if ok {
		logger.Debugf("signer for [%s] found", idHash)
		return entry.Signer, nil
	}

	o.viewsSync.Lock()
	defer o.viewsSync.Unlock()

	// check again the cache
	entry, ok = o.signers[idHash]
	if ok {
		logger.Debugf("signer for [%s] found", idHash)
		return entry.Signer, nil
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("signer for [%s] not found, try to deserialize", idHash)
	}
	// ask the deserializer
	if o.deserializer == nil {
		return nil, errors.Errorf("cannot find signer for [%s], no deserializer set", identity)
	}
	var err error
	signer, err := o.deserializer.DeserializeSigner(identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed deserializing identity for signer [%s]", identity)
	}
	entry = SignerEntry{Signer: signer}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		entry.DebugStack = debug.Stack()
	}
	o.signers[idHash] = entry

	return entry.Signer, nil
}

func (o *Service) GetSignerInfo(identity view.Identity) ([]byte, error) {
	return o.storage.GetSignerInfo(identity)
}

func (o *Service) GetVerifier(identity view.Identity) (driver.Verifier, error) {
	idHash := identity.UniqueID()

	// check cache
	o.viewsSync.RLock()
	entry, ok := o.verifiers[idHash]
	o.viewsSync.RUnlock()
	if ok {
		return entry.Verifier, nil
	}

	o.viewsSync.Lock()
	defer o.viewsSync.Unlock()

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
	verifier, err := o.deserializer.DeserializeVerifier(identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed deserializing identity for verifier %v", identity)
	}

	// store entry
	entry = VerifierEntry{Verifier: verifier}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		entry.DebugStack = debug.Stack()
		logger.Debugf("add deserialized verifier for [%s]:[%s]", idHash, GetIdentifier(verifier))
	}
	o.verifiers[idHash] = entry
	return verifier, nil
}

func GetIdentifier(f any) string {
	if f == nil {
		return "<nil>"
	}
	t := reflect.TypeOf(f)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.PkgPath() + "/" + t.Name()
}
