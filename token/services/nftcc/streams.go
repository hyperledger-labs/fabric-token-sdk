/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nftcc

import (
	"encoding/base64"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/nftcc/marshaller"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type OutputStream struct {
	*token.OutputStream
}

// Filter returns a stream of output filtered applying the passed filter
func (o *OutputStream) Filter(f func(t *token.Output) bool) *OutputStream {
	return &OutputStream{OutputStream: o.OutputStream.Filter(f)}
}

func (o *OutputStream) ByRecipient(id view.Identity) *OutputStream {
	return &OutputStream{OutputStream: o.OutputStream.ByRecipient(id)}
}

func (o *OutputStream) ByType(typ string) *OutputStream {
	return &OutputStream{OutputStream: o.OutputStream.ByType(typ)}
}

func (o *OutputStream) ByEnrollmentID(id string) *OutputStream {
	return &OutputStream{OutputStream: o.OutputStream.ByEnrollmentID(id)}
}

func (o *OutputStream) StateAt(index int, state interface{}) error {
	output := o.OutputStream.At(index)
	decoded, err := base64.StdEncoding.DecodeString(output.Type)
	if err != nil {
		return errors.Wrap(err, "failed to decode type")
	}
	if err := marshaller.Unmarshal(decoded, state); err == nil {
		return errors.Wrap(err, "failed to unmarshal state")
	}
	return nil
}

func (o *OutputStream) Validate() error {
	// all outputs must have quantity set to 1
	for _, output := range o.OutputStream.Outputs() {
		q, err := token2.ToQuantity(output.Quantity, 64)
		if err != nil {
			return errors.Wrapf(err, "failed to parse quantity [%s]", output.Quantity)
		}
		if q.Cmp(token2.NewQuantityFromUInt64(1)) != 0 {
			return errors.New("all outputs must have quantity set to 1")
		}
	}
	return nil
}
