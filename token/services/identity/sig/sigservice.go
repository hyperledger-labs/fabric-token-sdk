/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sig

import (
	"reflect"
	"runtime/debug"
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	identity2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = logging.MustGetLogger("token-sdk.services.identity.sig")

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
	sync      sync.RWMutex
	signers   map[string]SignerEntry
	verifiers map[string]VerifierEntry
	storage   Storage

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

func (o *Service) RegisterSigner(identity driver.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error {
	if signer == nil {
		return errors.New("invalid signer, expected a valid instance")
	}

	idHash := identity.UniqueID()
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("register signer and verifier [%s]:[%s][%s]", idHash, GetIdentifier(signer), GetIdentifier(verifier))
	}
	o.sync.RLock()
	s, ok := o.signers[idHash]
	o.sync.RUnlock()
	if ok {
		logger.Warnf("another signer bound to [%s]:[%s][%s] from [%s]", identity, GetIdentifier(s), GetIdentifier(signer), string(s.DebugStack))
		return nil
	}

	o.sync.Lock()

	// check again the cache
	s, ok = o.signers[idHash]
	if ok {
		o.sync.Unlock()
		logger.Warnf("another signer bound to [%s]:[%s][%s] from [%s]", identity, GetIdentifier(s), GetIdentifier(signer), string(s.DebugStack))
		return nil
	}

	entry := SignerEntry{Signer: signer}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		entry.DebugStack = debug.Stack()
	}

	o.signers[idHash] = entry
	if o.storage != nil {
		if err := o.storage.StoreSignerInfo(identity, signerInfo); err != nil {
			o.sync.Unlock()
			return errors.Wrap(err, "failed to store entry in storage for the passed signer")
		}
	}
	o.sync.Unlock()

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

func (o *Service) RegisterVerifier(identity driver.Identity, verifier driver.Verifier) error {
	if verifier == nil {
		return errors.New("invalid verifier, expected a valid instance")
	}

	idHash := identity.UniqueID()
	o.sync.Lock()
	v, ok := o.verifiers[idHash]
	o.sync.Unlock()
	if ok {
		logger.Warnf("another verifier bound to [%s]:[%s][%s] from [%s]", idHash, GetIdentifier(v), GetIdentifier(verifier), string(v.DebugStack))
		return nil
	}

	o.sync.Lock()

	// check again
	v, ok = o.verifiers[idHash]
	if ok {
		logger.Warnf("another verifier bound to [%s]:[%s][%s] from [%s]", idHash, GetIdentifier(v), GetIdentifier(verifier), string(v.DebugStack))
		return nil
	}

	entry := VerifierEntry{Verifier: verifier}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		entry.DebugStack = debug.Stack()
	}
	o.verifiers[idHash] = entry
	o.sync.Unlock()

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("register verifier to [%s]:[%s]", idHash, GetIdentifier(verifier))
	}
	return nil
}

func (o *Service) IsMe(identity driver.Identity) bool {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("is me [%s]?", identity)
	}
	idHash := identity.UniqueID()

	// check local cache
	o.sync.RLock()
	_, ok := o.signers[idHash]
	o.sync.RUnlock()
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
	//	if logger.IsEnabledFor(zapcore.DebugLevel) {
	//		logger.Debugf("is me [%s]? no", identity)
	//	}
	//	return false
	//}
	//if logger.IsEnabledFor(zapcore.DebugLevel) {
	//	logger.Debugf("is me [%s]? %v", identity, signer != nil)
	//}
	//return signer != nil
	return false
}

func (o *Service) GetSigner(identity driver.Identity) (driver.Signer, error) {
	idHash := identity.UniqueID()
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get signer for [%s]", idHash)
	}
	// check the cache
	o.sync.RLock()
	entry, ok := o.signers[idHash]
	o.sync.RUnlock()
	if ok {
		logger.Debugf("signer for [%s] found", idHash)
		return entry.Signer, nil
	}
	o.sync.Lock()
	defer o.sync.Unlock()

	return o.getSigner(identity, idHash)
}

func (o *Service) getSigner(identity driver.Identity, idHash string) (driver.Signer, error) {
	// check again the cache
	entry, ok := o.signers[idHash]
	if ok {
		logger.Debugf("signer for [%s] found", idHash)
		return entry.Signer, nil
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("signer for [%s] not found, try to deserialize", idHash)
	}
	// ask the deserializer
	signer, err := o.deserializeSigner(identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed deserializing identity for signer [%s]", identity)
	}
	entry = SignerEntry{Signer: signer}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		entry.DebugStack = debug.Stack()
	}
	o.signers[idHash] = entry
	if o.storage != nil {
		if err := o.storage.StoreSignerInfo(identity, nil); err != nil {
			return nil, errors.Wrap(err, "failed to store entry in storage for the passed signer")
		}
	}
	return entry.Signer, nil
}

func (o *Service) deserializeSigner(identity driver.Identity) (driver.Signer, error) {
	if o.deserializer == nil {
		return nil, errors.Errorf("cannot find signer for [%s], no deserializer set", identity)
	}
	var err error
	signer, err := o.deserializer.DeserializeSigner(identity)
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
	signer, err = o.getSigner(ro.Identity, ro.Identity.UniqueID())
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting signer for identity [%s]", ro.Identity)
	}
	return signer, nil
}

func (o *Service) GetSignerInfo(identity driver.Identity) ([]byte, error) {
	return o.storage.GetSignerInfo(identity)
}

func (o *Service) GetVerifier(identity driver.Identity) (driver.Verifier, error) {
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
