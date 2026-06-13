/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package uniqueness

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"reflect"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	platformkvs "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/nfttx/marshaller"
)

type Backend interface {
	Exists(ctx context.Context, k string) bool
	Get(ctx context.Context, k string, v any) error
	Put(ctx context.Context, k string, key any) error
}

// ServiceOption configures the uniqueness service.
type ServiceOption func(*Service)

// WithBackend configures the uniqueness service to use a custom backend.
func WithBackend(kvs Backend) ServiceOption {
	if kvs == nil {
		panic("backend is nil")
	}

	return func(s *Service) {
		s.kvs = kvs
	}
}

// NewService returns a new uniqueness service backed by the passed backend.
func NewService(kvs Backend) *Service {
	if kvs == nil {
		panic("backend is nil")
	}

	return &Service{kvs: kvs}
}

// InMemoryBackend is an in-memory uniqueness backend.
type InMemoryBackend struct {
	lock  sync.RWMutex
	store map[string]any
}

// NewInMemoryBackend returns a new in-memory uniqueness backend.
func NewInMemoryBackend() Backend {
	return &InMemoryBackend{
		store: map[string]any{},
	}
}

func (b *InMemoryBackend) Exists(_ context.Context, k string) bool {
	b.lock.RLock()
	defer b.lock.RUnlock()

	_, ok := b.store[k]
	return ok
}

func (b *InMemoryBackend) Get(_ context.Context, k string, v any) error {
	b.lock.RLock()
	value, ok := b.store[k]
	b.lock.RUnlock()

	if !ok {
		return errors.WithMessagef(errors.New("key not found"), "key %s", k)
	}

	if v == nil {
		return errors.New("value must be a non-nil pointer")
	}

	target := reflect.ValueOf(v)
	if target.Kind() != reflect.Ptr || target.IsNil() {
		return errors.New("value must be a non-nil pointer")
	}

	valueVal := reflect.ValueOf(value)
	if !valueVal.Type().AssignableTo(target.Elem().Type()) {
		return errors.WithMessagef(errors.New("type mismatch"), "cannot assign %T to %T", value, v)
	}

	target.Elem().Set(valueVal)
	return nil
}

func (b *InMemoryBackend) Put(_ context.Context, k string, key any) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.store[k] = key
	return nil
}

// Service is a uniqueness service.
// The service computes a unique id for a given object by hashing the object's json representation together with a random salt.
// The random salt is used to avoid dictionary attacks.
// The random salt is stored in the backend and is generated on the first call to the service.
type Service struct {
	mutex sync.Mutex
	kvs   Backend
}

// ComputeID computes the unique ID of the given object.
func (s *Service) ComputeID(state any) (string, error) {
	if state == nil {
		return "", errors.New("state is nil")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	k := "github.com/hyperledger-labs/fabric-token-sdk/token/services/nfttx/uniqueness/key"
	var key []byte
	if s.kvs.Exists(context.Background(), k) {
		if err := s.kvs.Get(context.Background(), k, &key); err != nil {
			return "", errors.WithMessagef(err, "failed to get key %s", k)
		}
	} else {
		const size = 32
		key = make([]byte, size)
		n, err := rand.Read(key)
		if err != nil {
			return "", errors.Wrap(err, "error getting random bytes")
		}
		if n != size {
			return "", errors.New("error getting random bytes")
		}
		if err := s.kvs.Put(context.Background(), k, key); err != nil {
			return "", errors.WithMessagef(err, "failed to put key %s", k)
		}
	}

	raw, err := marshaller.Marshal(state)
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal state")
	}

	hash := sha256.New()
	n, err := hash.Write(key)
	if n != len(key) {
		return "", errors.New("error writing key to hash")
	}
	if err != nil {
		return "", errors.Wrapf(err, "error writing key to hash")
	}

	n, err = hash.Write(raw)
	if n != len(raw) {
		return "", errors.New("error writing raw state to hash")
	}
	if err != nil {
		return "", errors.Wrapf(err, "error writing raw state to hash")
	}

	digest := hash.Sum(nil)
	return base64.StdEncoding.EncodeToString(digest), nil
}

// GetService returns the uniqueness service.
func GetService(sp token.ServiceProvider, opts ...ServiceOption) *Service {
	service := &Service{}
	for _, opt := range opts {
		opt(service)
	}

	if service.kvs == nil {
		kvss, err := sp.GetService(&platformkvs.KVS{})
		if err != nil {
			panic(err)
		}

		service.kvs = kvss.(*platformkvs.KVS)
	}

	return service
}
