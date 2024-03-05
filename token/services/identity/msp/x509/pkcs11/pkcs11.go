/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pkcs11

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/config"
	"github.com/hyperledger/fabric/bccsp/pkcs11"
)

func ToOpts(o *config.PKCS11) *pkcs11.PKCS11Opts {
	res := &pkcs11.PKCS11Opts{
		Security:       o.Security,
		Hash:           o.Hash,
		Library:        o.Library,
		Label:          o.Label,
		Pin:            o.Pin,
		SoftwareVerify: o.SoftwareVerify,
		Immutable:      o.Immutable,
		AltID:          o.AltID,
	}
	for _, d := range o.KeyIDs {
		res.KeyIDs = append(res.KeyIDs, pkcs11.KeyIDMapping{
			SKI: d.SKI,
			ID:  d.ID,
		})
	}
	return res
}
