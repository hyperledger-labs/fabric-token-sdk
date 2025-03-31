/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"
)

type Binding struct {
	FSCNodeIdentity view.Identity
	Alias           view.Identity
}

type SetBindingView struct {
	*Binding
}

func (s *SetBindingView) Call(context view.Context) (interface{}, error) {
	es := view2.GetEndpointService(context)
	if err := es.Bind(s.FSCNodeIdentity, s.Alias); err != nil {
		return nil, errors.Wrap(err, `failed to bind fsc node identity`)
	}
	return nil, nil
}

type SetBindingViewFactory struct{}

func (p *SetBindingViewFactory) NewView(in []byte) (view.View, error) {
	f := &SetBindingView{Binding: &Binding{}}
	err := json.Unmarshal(in, f.Binding)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
