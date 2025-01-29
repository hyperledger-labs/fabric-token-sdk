/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"encoding/asn1"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	idemix2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509"
	"github.com/pkg/errors"
)

const Escrow = "Multisig"

type Deserializer struct {
	Type string
}

func (d *Deserializer) DeserializeVerifier(id view.Identity) (driver.Verifier, error) {
	multisigId := &MultiIdentity{}
	_, err := asn1.Unmarshal(id, multisigId)
	if err != nil {
		return nil, errors.New("failed to unmarshal multisig identity")
	}
	verifier := &MultiVerifier{}
	verifier.Verifiers = make([]driver.Verifier, len(multisigId.Identities))
	switch t := d.Type; t {
	case msp.X509Identity:
		x509Deserializer := &x509.MSPIdentityDeserializer{}
		for k, i := range multisigId.Identities {
			verifier.Verifiers[k], err = x509Deserializer.DeserializeVerifier(i)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to unmarshal "+
					"multisig identity")
			}
		}
	case msp.IdemixIdentity:
		idemixDeserializer := &idemix2.Deserializer{}
		for k, i := range multisigId.Identities {
			verifier.Verifiers[k], err = idemixDeserializer.DeserializeVerifier(i)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to unmarshal "+
					"multisig identity")
			}
		}
	default:
		return nil, errors.New("failed to unmarshal multisig identity: " +
			"invalid deserializer type")
	}
	return verifier, nil
}

func (t *Deserializer) Recipients(id driver.Identity, typ string, raw []byte) ([]driver.Identity, error) {
	if typ != Escrow {
		return nil, errors.New("unknown identity type")
	}
	escrow := &MultiIdentity{}
	err := escrow.Deserialize(raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal htlc script")
	}
	return escrow.Identities, nil
}
