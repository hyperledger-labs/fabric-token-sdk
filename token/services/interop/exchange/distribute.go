/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package exchange

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/pkg/errors"
)

type Terms struct {
	ReclamationDeadline time.Duration
	TMSID1              token.TMSID
	Type1               string
	Amount1             uint64
	TMSID2              token.TMSID
	Type2               string
	Amount2             uint64
}

func (t *Terms) Bytes() ([]byte, error) {
	return json.Marshal(t)
}

func (t *Terms) FromBytes(raw []byte) error {
	return json.Unmarshal(raw, t)
}

type DistributeTermsView struct {
	recipient view.Identity
	terms     *Terms
}

func NewDistributeTermsView(recipient view.Identity, terms *Terms) *DistributeTermsView {
	return &DistributeTermsView{
		recipient: recipient,
		terms:     terms,
	}
}

func (v *DistributeTermsView) Call(context view.Context) (interface{}, error) {
	session, err := context.GetSession(context.Initiator(), v.recipient)
	if err != nil {
		return nil, err
	}
	termsRaw, err := v.terms.Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling terms")
	}
	err = session.Send(termsRaw)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

type termsReceiverView struct{}

func ReceiveTerms(context view.Context) (*Terms, error) {
	terms, err := context.RunView(&termsReceiverView{})
	if err != nil {
		return nil, err
	}
	return terms.(*Terms), nil
}

func (v *termsReceiverView) Call(context view.Context) (interface{}, error) {
	_, payload, err := session.ReadFirstMessage(context)
	if err != nil {
		return nil, err
	}
	terms := &Terms{}
	if err := terms.FromBytes(payload); err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling terms")
	}
	// TODO review terms and accept
	return terms, nil
}
