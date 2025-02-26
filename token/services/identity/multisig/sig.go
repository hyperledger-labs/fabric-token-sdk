/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"encoding/asn1"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

type MultiSignature struct {
	Signatures [][]byte
}

func (m *MultiSignature) Bytes() ([]byte, error) {
	return asn1.Marshal(*m)
}

func (m *MultiSignature) FromBytes(raw []byte) error {
	_, err := asn1.Unmarshal(raw, m)
	return err
}

func JoinSignatures(sigmas [][]byte) ([]byte, error) {
	sig := &MultiSignature{
		Signatures: sigmas,
	}
	return sig.Bytes()
}

type Verifier struct {
	Verifiers []driver.Verifier
}

func (v *Verifier) Verify(msg, raw []byte) error {
	sig := &MultiSignature{}
	err := sig.FromBytes(raw)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal multisig [%s]", hash.Hashable(raw))
	}
	if len(v.Verifiers) != len(sig.Signatures) {
		return errors.Errorf("invalid multisig: expect [%d] signatures, but received [%d]", len(v.Verifiers), len(sig.Signatures))
	}
	for k, ver := range v.Verifiers {
		if err = ver.Verify(msg, sig.Signatures[k]); err != nil {
			return errors.Errorf("invalid multisig: signature at index [%d] does not verify", k)
		}
	}
	return nil
}
