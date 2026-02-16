/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/endpoint"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type Binding struct {
	FSCNodeIdentity view.Identity
	Alias           view.Identity
}

type SetBindingView struct {
	*Binding
}

func (s *SetBindingView) Call(context view.Context) (interface{}, error) {
	es := endpoint.GetService(context)
	if err := es.Bind(context.Context(), s.FSCNodeIdentity, s.Alias); err != nil {
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
