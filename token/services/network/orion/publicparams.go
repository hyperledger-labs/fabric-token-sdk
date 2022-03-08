/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/translator"
	"github.com/pkg/errors"
)

type PublicParamsRequestView struct {
}

func (v *PublicParamsRequestView) Call(context view.Context) (interface{}, error) {

	custodian, err := config.GetCustodian(view2.GetConfigService(context))
	if err != nil {
		return nil, err
	}
	session, err := context.GetSession(context.Initiator(), view.Identity(custodian))
	if err != nil {
		return nil, err
	}

	// Wait to receive the public params
	ch := session.Receive()
	var payload []byte
	select {
	case msg := <-ch:
		payload = msg.Payload
	case <-time.After(60 * time.Second):
		return nil, errors.New("time out reached")
	}

	return payload, nil
}

type RespondPublicParamsRequestView struct {
}

func (v *RespondPublicParamsRequestView) Call(context view.Context) (interface{}, error) {

	rwset := &orion.RWSWrapper{}
	issuingValidator := &AllIssuersValid{}
	w := translator.New(issuingValidator, "", rwset, "")
	ppRaw, err := w.ReadSetupParameters()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve public parameters")
	}
	if len(ppRaw) == 0 {
		return nil, errors.Errorf("public parameters are not initiliazed yet")
	}

	session := context.Session()

	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling public params")
	}
	err = session.Send(ppRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send public params")
	}

	return nil, nil
}

type AllIssuersValid struct{}

func (i *AllIssuersValid) Validate(creator view.Identity, tokenType string) error {
	return nil
}
