/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"encoding/asn1"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/pkg/errors"
)

const Escrow = "ms"

type MultiIdentity struct {
	Identities []token.Identity
}

func (m *MultiIdentity) Serialize() ([]byte, error) {
	return asn1.Marshal(*m)
}

func (m *MultiIdentity) Deserialize(raw []byte) error {
	_, err := asn1.Unmarshal(raw, m)
	return err
}

func (m *MultiIdentity) Bytes() ([]byte, error) {
	return asn1.Marshal(*m)
}

func Wrap(ids ...token.Identity) (token.Identity, error) {
	mi := &MultiIdentity{Identities: ids}
	raw, err := mi.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed marshalling multi identity")
	}
	typedIdentity, err := (&identity.TypedIdentity{Type: Escrow, Identity: raw}).Bytes()
	if err != nil {
		return nil, err
	}
	return typedIdentity, nil
}

func Unwrap(raw []byte) (bool, []token.Identity, error) {
	ti, err := identity.UnmarshalTypedIdentity(raw)
	if err != nil {
		return false, nil, errors.Wrap(err, "failed unmarshalling typed identity")
	}
	if ti.Type != Escrow {
		return false, nil, nil
	}
	mi := &MultiIdentity{}
	err = mi.Deserialize(ti.Identity)
	if err != nil {
		return false, nil, errors.Wrap(err, "failed unmarshalling multi identity")
	}
	return true, mi.Identities, nil
}
