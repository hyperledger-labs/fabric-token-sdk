/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"
)

// Signature contains the input information to generate composite key
type Signature struct {
	ObjectType       string
	TransactionID    string
	LongTermIdentity view.Identity
}

type SignatureView struct {
	*Signature
}

func (t *SignatureView) Call(context view.Context) (interface{}, error) {

	//generate compositekey
	k := kvs.GetService(context)
	ackKey, err := kvs.CreateCompositeKey(t.ObjectType, []string{t.TransactionID, t.LongTermIdentity.String()})
	if err != nil {
		return nil, errors.Wrap(err, "failed creating composite key")
	}
	if k.Exists(ackKey) {
		return "Key Exist", nil
	} else {
		return nil, errors.WithMessagef(err, "failed to get key %s", ackKey)
	}
}

type SignatureViewFactory struct{}

func (p *SignatureViewFactory) NewView(in []byte) (view.View, error) {
	f := &SignatureView{Signature: &Signature{}}
	err := json.Unmarshal(in, f.Signature)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
