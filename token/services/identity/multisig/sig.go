/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

type MultiIdentity struct {
	Identities []driver.Identity
}

type MultiVerifier struct {
	Verifiers []driver.Verifier
}

type MultiSignature struct {
	Signatures [][]byte
}

func (v *MultiVerifier) Verify(msg, raw []byte) error {
	sig := &MultiSignature{}
	err := json.Unmarshal(raw, sig)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal multisig")
	}
	if len(v.Verifiers) != len(sig.Signatures) {
		return errors.Errorf("Invalid multisig:  expect % d signatures, "+
			"but received % d", len(v.Verifiers), len(sig.Signatures))
	}
	for k, ver := range v.Verifiers {
		if err = ver.Verify(msg, sig.Signatures[k]); err != nil {
			return errors.Errorf("Invalid multisig: signature at index "+
				"%d does not verify", k)
		}
	}
	return nil
}

func (id *MultiIdentity) Serialize() ([]byte, error) {
	raw, err := json.Marshal(id)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (id *MultiIdentity) Deserialize(raw []byte) error {
	err := json.Unmarshal(raw, id)
	if err != nil {
		return err
	}
	return nil
}
