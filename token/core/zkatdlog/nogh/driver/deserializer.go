/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	idemix2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type Deserializer struct {
	Service *nogh.Service
}

func (d *Deserializer) DeserializeVerifier(raw []byte) (driver2.Verifier, error) {
	des, err := idemix2.NewDeserializer(d.Service.PublicParams().IdemixPK)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting idemix deserializer for passed public params")
	}

	return identity.NewRawOwnerIdentityDeserializer(&idemixDeserializer{provider: des}).DeserializeVerifier(raw)
}

func (d *Deserializer) DeserializeSigner(raw []byte) (driver2.Signer, error) {
	return nil, errors.New("not supported")
}

func (d *Deserializer) Info(raw []byte, auditInfo []byte) (string, error) {
	return "", errors.New("not supported")
}

type idemixProvider interface {
	DeserializeVerifier(raw []byte) (driver2.Verifier, error)
}

type idemixDeserializer struct {
	provider idemixProvider
}

func (i *idemixDeserializer) GetVerifier(id view.Identity) (driver.Verifier, error) {
	return i.provider.DeserializeVerifier(id)
}
