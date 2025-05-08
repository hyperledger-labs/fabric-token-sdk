/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sig

import (
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/pkg/errors"
)

type MultiplexDeserializer struct {
	deserializersMutex sync.RWMutex
	deserializers      []idriver.Deserializer
}

func NewMultiplexDeserializer() *MultiplexDeserializer {
	return &MultiplexDeserializer{
		deserializers: []idriver.Deserializer{},
	}
}

func (d *MultiplexDeserializer) AddDeserializer(newD idriver.Deserializer) {
	d.deserializersMutex.Lock()
	d.deserializers = append(d.deserializers, newD)
	d.deserializersMutex.Unlock()
}

func (d *MultiplexDeserializer) DeserializeVerifier(raw []byte) (driver.Verifier, error) {
	return deserialize(d.threadSafeCopyDeserializers(), func(deserializer idriver.Deserializer) (driver.Verifier, error) {
		return deserializer.DeserializeVerifier(raw)
	})
}

func (d *MultiplexDeserializer) DeserializeSigner(raw []byte) (driver.Signer, error) {
	return deserialize(d.threadSafeCopyDeserializers(), func(deserializer idriver.Deserializer) (driver.Signer, error) {
		return deserializer.DeserializeSigner(raw)
	})
}

func (d *MultiplexDeserializer) Info(raw []byte, auditInfo []byte) (string, error) {
	return deserialize(d.threadSafeCopyDeserializers(), func(deserializer idriver.Deserializer) (string, error) {
		return deserializer.Info(raw, auditInfo)
	})
}

func (d *MultiplexDeserializer) threadSafeCopyDeserializers() []idriver.Deserializer {
	d.deserializersMutex.RLock()
	res := make([]idriver.Deserializer, len(d.deserializers))
	copy(res, d.deserializers)
	d.deserializersMutex.RUnlock()
	return res
}

func deserialize[V any](copyDeserial []idriver.Deserializer, extractor func(idriver.Deserializer) (V, error)) (V, error) {
	var defaultV V
	var errs []error

	for _, des := range copyDeserial {
		logger.Debugf("trying signer deserialization with [%s]", des)
		v, err := extractor(des)
		if err == nil {
			logger.Debugf("trying signer deserialization with [%s] succeeded", des)
			return v, nil
		}

		logger.Debugf("trying signer deserialization with [%s] failed [%s]", des, err)
		errs = append(errs, err)
	}

	return defaultV, errors.Errorf("failed signer deserialization [%v]", errs)
}
