/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sig

import (
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type Deserializer interface {
	DeserializeVerifier(raw []byte) (driver.Verifier, error)
	DeserializeSigner(raw []byte) (driver.Signer, error)
	Info(raw []byte, auditInfo []byte) (string, error)
}

type Manager interface {
	AddDeserializer(deserializer Deserializer)
	DeserializeSigner(raw []byte) (driver.Signer, error)
}

type MultiplexDeserializer struct {
	deserializersMutex sync.RWMutex
	deserializers      []Deserializer
}

func NewMultiplexDeserializer() *MultiplexDeserializer {
	return &MultiplexDeserializer{
		deserializers: []Deserializer{},
	}
}

func (d *MultiplexDeserializer) AddDeserializer(newD Deserializer) {
	d.deserializersMutex.Lock()
	d.deserializers = append(d.deserializers, newD)
	d.deserializersMutex.Unlock()
}

func (d *MultiplexDeserializer) DeserializeVerifier(raw []byte) (driver.Verifier, error) {
	return deserialize(d.threadSafeCopyDeserializers(), func(deserializer Deserializer) (driver.Verifier, error) {
		return deserializer.DeserializeVerifier(raw)
	})
}

func (d *MultiplexDeserializer) DeserializeSigner(raw []byte) (driver.Signer, error) {
	return deserialize(d.threadSafeCopyDeserializers(), func(deserializer Deserializer) (driver.Signer, error) {
		return deserializer.DeserializeSigner(raw)
	})
}

func (d *MultiplexDeserializer) Info(raw []byte, auditInfo []byte) (string, error) {
	return deserialize(d.threadSafeCopyDeserializers(), func(deserializer Deserializer) (string, error) {
		return deserializer.Info(raw, auditInfo)
	})
}

func (d *MultiplexDeserializer) threadSafeCopyDeserializers() []Deserializer {
	d.deserializersMutex.RLock()
	res := make([]Deserializer, len(d.deserializers))
	copy(res, d.deserializers)
	d.deserializersMutex.RUnlock()
	return res
}

func deserialize[V any](copyDeserial []Deserializer, extractor func(Deserializer) (V, error)) (V, error) {
	var defaultV V
	var errs []error

	for _, des := range copyDeserial {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("trying signer deserialization with [%s]", des)
		}
		v, err := extractor(des)
		if err == nil {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("trying signer deserialization with [%s] succeeded", des)
			}
			return v, nil
		}

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("trying signer deserialization with [%s] failed [%s]", des, err)
		}
		errs = append(errs, err)
	}

	return defaultV, errors.Errorf("failed signer deserialization [%v]", errs)
}
